# Trust Anchor —— 跨平台根证书信任库工具技术方案

> 状态：已实施（根证书信任库模块与项目骨架完成，通过构建、测试与启动验证）
> 更新日期：2026-11-16
> 目标平台：Windows / macOS / Linux

## 1. 选型结论

采用 **Fyne v2（纯 Go 自绘 UI）+ go-selfupdate（GitHub Releases 自更新）+ Makefile + 原生工具链本地构建**。

不引入 Web 前端、不引入 Node/Electron/WebView 依赖，整个应用就是一个 Go 二进制。

### 备选方案对比（存档）

| 方案 | 原理 | 优点 | 缺点 |
|---|---|---|---|
| **Fyne v2（选用）** | 纯 Go、自绘 UI（OpenGL/软件渲染） | 全部代码为 Go；三平台外观一致；单文件发布；无需前端工具链 | 依赖 CGO，跨平台编译需对应工具链或 fyne-cross；控件偏 Material 风格，复杂自定义 UI 开发量大 |
| Wails v2 | Go 后端 + 系统 WebView 渲染 HTML/CSS 前端 | 界面自由度最高，可复用前端技能，打包链路成熟 | Linux 运行依赖 webkit2gtk；需同时维护前端工程 |
| Gio | 纯 Go 自绘，低层 API | 跨编译最干净、启动快、体积小 | 所有控件自行绘制，开发成本最高 |
| webview_go | 极简 WebView 绑定 | 极小（约 3MB）、灵活 | 仅一层绑定，菜单/托盘/打包全部自建 |

选择理由：需求以原生桌面工具为主，界面复杂度不高，纯 Go 能显著降低构建与分发复杂度。

## 2. 目标目录结构

```text
trustanchor/
├── FyneApp.toml            # 应用名/版本/图标等元数据，fyne package 读取
├── Makefile                # run / build / package / install / test / vet / cross / clean
├── README.md               # 各平台依赖、构建打包、图标与签名说明
├── go.mod / go.sum
├── main.go                 # 入口：提权子进程参数分发 + 启动界面
├── assets/
│   ├── assets.go           # go:embed 图标
│   ├── icon.png            # 应用图标（512x512）
│   └── icons/              # 16~512 多尺寸图标
├── packaging/linux/
│   └── io.github.yinrg.trustanchor.desktop  # Linux 桌面集成
├── tools/genicon/          # 纯 Go 图标生成器
└── internal/
    ├── version/version.go  # 版本号，ldflags 注入 git describe
    ├── updater/updater.go  # 检查更新 / 下载替换 / 打开下载页
    ├── truststore/         # 内嵌根证书 + 三平台信任库安装 / 移除 / 提权
    └── ui/
        ├── app.go          # 生命周期、安装/移除、检查更新
        ├── view.go         # 极简单页主界面、证书详情与关于弹窗
        └── tray.go         # 系统托盘菜单
```

## 3. 依赖

| 依赖 | 用途 |
|---|---|
| `fyne.io/fyne/v2` | GUI 框架、窗口、菜单、托盘 |
| `github.com/creativeprojects/go-selfupdate` | 基于 GitHub Releases 的版本检测与自更新 |
| `fyne.io/tools/cmd/fyne`（工具） | 本地打包（`fyne package`）、图标资源 |
| `fyne-cross`（可选工具，Docker） | Linux 上交叉编译 Windows/Linux 目标 |

## 4. 自动更新设计

### 流程

1. 启动后异步请求 GitHub Releases，取最新 tag，与当前版本比较（语义化版本）。
2. 有新版本 → 弹窗展示版本号与 Release Notes，提供两个操作：
   - `前往下载`：打开浏览器跳转 Release 页面（三平台通用兜底）。
   - `立即更新`：进入自替换流程。
3. 设置项 `启动时检查更新`，默认开启。

### 平台差异（必须如实处理）

| 平台 | 自替换 | 说明 |
|---|---|---|
| Windows | 支持 | 无法直接覆盖运行中的 exe，采用「重命名旧 exe → 写入新 exe → 退出后清理」策略 |
| Linux | 支持 | AppImage/单二进制场景最干净；需对安装目录有写权限 |
| macOS | **不支持自替换** | 替换 `.app` 内的可执行文件会破坏代码签名与公证，改为引导用户下载 dmg |

### 配置项

- 仓库地址 `owner/repo` 作为 `updater` 包内的可配置常量（后续可迁到 `FyneApp.toml` 或用户配置）。
- 未配置仓库地址时，更新功能静默关闭，不报错、不影响启动。
- 网络失败仅记录日志，不弹窗打扰用户。

## 5. 构建与打包（本地）

### 各平台前置依赖

| 平台 | 前置依赖 | 产物 |
|---|---|---|
| Windows | Go、MinGW-w64 gcc（CGO 需要） | `.exe` |
| macOS | Go、Xcode Command Line Tools | `.app`（可选打包 dmg） |
| Linux | Go、gcc、`libgl1-mesa-dev xorg-dev libxkbcommon-dev` | `.tar.xz` / `.deb` |

### Makefile 目标

| 目标 | 作用 |
|---|---|
| `make run` | 本地运行调试 |
| `make test` | 单元测试 |
| `make vet` / `make fmt` | 静态检查与格式化 |
| `make build` | 构建当前平台二进制（版本号由 `git describe` 注入） |
| `make package` | 当前平台安装包（exe / app / tar.xz+deb） |
| `make cross` | 在 Linux 上用 fyne-cross（Docker）交叉构建 Windows、Linux 目标 |
| `make clean` | 清理产物 |

### 跨平台限制

- **macOS 目标必须在 macOS 上编译**（无法在 Linux/Windows 上合法生成 `.app` 并进行签名）。
- Windows/Linux 目标可在 Linux 上通过 fyne-cross 交叉构建。
- 代码签名与公证不在本次范围内，README 只提供操作步骤与命令参考。

## 6. 本次交付范围

- [x] 方案文档（本文件）
- [x] 根证书信任库模块（三平台安装 / 移除 / 状态检测 / 提权）
- [x] 项目骨架与可运行界面（版本信息、根证书卡片、检查更新）
- [x] 托盘菜单与主菜单
- [x] 更新模块（含平台差异处理）
- [x] Makefile 与 README
- [x] 通过 `go build`、`go test`、`go vet` 验证

不在范围内：CI/CD 自动构建、数据库、文件关联、代码签名。

## 7. 已确认事项

1. **GitHub 仓库地址**：`github.com/yinrg/trustanchor`，已写入 `updater.RepositorySlug`。
2. **应用实际业务功能**：系统根证书信任库管理（安装、移除、状态检测）。
3. **应用名称与图标**：名称定为 Trust Anchor，AppID `io.github.yinrg.trustanchor`；图标仍为占位稿，可后续替换。
