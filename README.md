# Trust Anchor

把内嵌根证书安装到系统受信任根证书存储的桌面客户端，基于 Go + [Fyne](https://fyne.io/)，一套代码同时支持 Windows、macOS、Linux。

## 功能

- 根证书一键安装：写入系统受信任根证书存储（Windows 证书存储、macOS 钥匙串、Linux ca-certificates 体系）
- 安装状态检测与移除：展示主体、SHA-256 指纹与有效期，可自助卸载
- 管理员授权：Windows UAC、macOS 系统授权框、Linux pkexec；无法自动提权时给出可复制的终端命令
- 原生窗口、主菜单、系统托盘（Fyne 自绘 UI，无 WebView / 前端依赖）
- 版本、提交、构建时间、运行平台信息展示（`ldflags` 注入）
- 基于 GitHub Releases 的自动检查更新
- Windows / Linux 原地自更新；macOS 引导下载安装包（签名原因）
- `FyneApp.toml` 打包元数据 + Makefile 本地构建脚本

## 目录结构

```text
.
├── FyneApp.toml            # 应用名、ID、版本、图标等打包元数据
├── Makefile                # run / build / package / test / vet / cross / clean
├── main.go                 # 入口：先处理提权子进程参数，再启动界面
├── assets/
│   ├── assets.go           # go:embed 图标
│   ├── logo.svg            # 主标志（盾牌 + 锚）
│   ├── logo-small.svg      # 小尺寸简化标志（仅锚）
│   ├── icon.png            # 应用图标（512x512）
│   └── icons/              # 16~512 多尺寸图标（Linux 主题用）
├── packaging/
│   └── linux/
│       └── io.github.yinrg.trustanchor.desktop
├── tools/
│   └── genicon/            # 图标生成器（make icon）
├── internal/
│   ├── truststore/
│   │   ├── rootCA.crt      # 内嵌根证书（PEM，业务凭证）
│   │   ├── truststore.go   # 证书解析、CA 校验、状态汇总
│   │   ├── install_*.go    # 各平台信任库写入与刷新
│   │   ├── elevate*.go     # 提权安装与结果回传
│   │   └── cli.go          # 提权后子进程入口（--install-ca 等）
│   ├── ui/
│   │   ├── app.go          # 生命周期、安装/移除、更新交互
│   │   ├── view.go         # 主界面布局、状态渲染、详情/关于弹窗
│   │   └── tray.go         # 系统托盘
│   ├── updater/
│   │   └── updater.go      # 版本检测与自更新
│   └── version/
│       └── version.go      # 版本变量，构建时注入
└── docs/
    └── solution.md         # 技术方案记录
```

## 环境要求

- Go **1.26+**（依赖 `go-selfupdate` 的最低要求）
- CGO 必须开启（Fyne 需要 OpenGL 与系统图形库）
- 各平台编译依赖：

### Ubuntu / Debian

```bash
sudo apt install -y build-essential pkg-config \
  libgl1-mesa-dev xorg-dev libxkbcommon-dev libwayland-dev
```

### Fedora

```bash
sudo dnf install -y gcc pkgconf-pkg-config libX11-devel libXcursor-devel \
  libXrandr-devel libXinerama-devel libXi-devel mesa-libGL-devel libxkbcommon-devel
```

### Arch Linux

```bash
sudo pacman -S --needed base-devel pkgconf libx11 libxcursor libxrandr \
  libxinerama libxi mesa libxkbcommon
```

### macOS

```bash
xcode-select --install
```

### Windows

需要 MinGW-w64 的 gcc（CGO 必需），推荐 MSYS2：

```bash
pacman -S --needed mingw-w64-x86_64-gcc
```

并确保 `gcc` 在 `PATH` 中。

## 运行

```bash
make run
# 或
go run .
```

## 构建

```bash
make build
```

产物：`build/trustanchor`（Windows 为 `build/trustanchor.exe`）。

版本号默认取 `git describe --tags --abbrev=0`，没有 tag 时为 `0.0.0-dev`。也可以手动指定：

```bash
make build VERSION=v1.2.3 COMMIT=abc1234
```

## 打包

```bash
make package
```

`fyne package` 会读取 `FyneApp.toml` 中的名称、图标与版本，产物自动移动到 `build/`：

| 平台 | 产物 |
| --- | --- |
| Linux | `Trust Anchor.tar.xz` |
| macOS | `Trust Anchor.app` |
| Windows | `Trust Anchor.exe` |

> `make package` 依赖 `go run fyne.io/tools/cmd/fyne@latest`，首次运行需要联网下载工具。

## 任务栏 / 程序图标

图标在三个平台的来源不同，务必区分：

| 平台 | 任务栏 / Dock 图标来源 | 做法 |
| --- | --- | --- |
| Linux（Wayland） | 窗口 `app_id` 匹配 `~/.local/share/applications/<id>.desktop`，图标取自 hicolor 主题 | `make install` |
| Linux（X11） | 窗口 `WM_CLASS` 匹配 `.desktop` 的 `StartupWMClass`，否则用 `_NET_WM_ICON` | `make install` |
| Windows | 可执行文件内嵌的 icon 资源（不是运行时 `SetIcon`） | `make package`（Windows 本机）或 `make cross-windows`（Linux 交叉） |
| macOS | `.app` 包内的 `AppIcon.icns` | `make package`（裸二进制没有 Dock 图标） |

Linux 安装（当前用户，无需 root）：

```bash
make install          # 安装到 ~/.local/bin + ~/.local/share
make uninstall        # 卸载
```

安装内容：

- `~/.local/bin/trustanchor`
- `~/.local/share/applications/io.github.yinrg.trustanchor.desktop`
- `~/.local/share/icons/hicolor/{16,24,32,48,64,128,256,512}/apps/io.github.yinrg.trustanchor.png`

对应关系：Wayland 下 `app_id` 为 `app.NewWithID("io.github.yinrg.trustanchor")`，X11 下 `WM_CLASS` 为窗口标题 `Trust Anchor`，两者都已在 `.desktop` 中声明。

修改图标后重新生成多尺寸资源：

```bash
make icon
```

图标生成器是纯 Go 实现（`tools/genicon`）：`assets/logo.svg` 是主标志，`assets/logo-small.svg` 用于小于 48px 的尺寸，避免任务栏小图标糊成一团。渲染时先按 512×512 原生出图，再按面积平均缩放到各尺寸，保证描边粗细一致。

窗口与托盘图标使用 64×64 的 `assets/icons/64x64.png`（`assets.WindowIcon()`）。X11 对 `_NET_WM_ICON` 的请求体积有上限，512×512 的 ARGB 数据写不进去，而 GLFW 会静默忽略这个错误——所以窗口图标不能直接用 512 那张。512×512 的 `assets/icon.png` 只用于打包和桌面主题。

### 交叉编译

Windows amd64 可以直接在 Linux 上用 MinGW-w64 交叉编译：

```bash
sudo apt install -y mingw-w64
make cross-windows     # → build/trustanchor.exe
```

产物已包含应用图标（16/32/48/64/256 多尺寸）、DPI 感知清单（per-monitor v2）、版本资源，并链接为 GUI 子系统（不会弹出控制台窗口）。

- apt 提供的 MinGW 只支持 Windows **amd64 / 386**，不含 arm64；arm64 需在 Windows 上构建或使用 llvm-mingw。
- 也可以使用 [fyne-cross](https://github.com/fyne-io/fyne-cross)（需要 Docker 权限）：`make cross`，可同时构建 Windows 与 Linux 目标。
- **macOS 目标必须在 macOS 上构建**，无法在 Linux/Windows 上交叉生成可用的 `.app`。

## 自动更新

### 启用

1. 仓库地址已配置为 `yinrg/trustanchor`，位置在 `internal/updater/updater.go`：

   ```go
   var RepositorySlug = "yinrg/trustanchor"
   ```

   fork 或改名后需同步修改；置空则关闭更新功能。

2. 发布 Release 时使用语义化版本 tag，例如 `v0.2.0`。
3. 上传带平台标识的资产（`go-selfupdate` 依据文件名中的 OS/架构匹配）：

   | 平台 | 资产命名示例 |
   | --- | --- |
   | Windows amd64 | `trustanchor_0.2.0_windows_amd64.zip` |
   | Windows arm64 | `trustanchor_0.2.0_windows_arm64.zip` |
   | Linux amd64 | `trustanchor_0.2.0_linux_amd64.tar.gz` |
   | Linux arm64 | `trustanchor_0.2.0_linux_arm64.tar.gz` |
   | macOS amd64 | `trustanchor_0.2.0_darwin_amd64.zip` |
   | macOS arm64 | `trustanchor_0.2.0_darwin_arm64.zip` |

   zip / tar.gz 内需包含可执行文件。可选上传 `checksums.txt`，`go-selfupdate` 会用于校验。

### 行为

- 启动后 1.5 秒自动检查（可在界面勾选「启动时检查更新」关闭，偏好持久化）。
- 菜单「文件 → 检查更新…」或托盘菜单可手动触发。
- 发现新版本时弹窗展示版本号与 Release Notes，可跳转下载页。
- Windows / Linux 点击「立即更新」会下载并替换当前可执行文件，重启后生效。
- **macOS 不执行原地替换**：替换 `.app` 内的可执行文件会破坏代码签名与公证，因此只提供下载入口。
- 未配置 `RepositorySlug` 时更新功能静默关闭，手动检查会提示配置方法。

### 已知限制

- Windows 安装在 `Program Files` 等受保护目录时自更新会失败，需要写权限；此时建议引导用户重新下载。
- Linux 使用系统包管理器安装时应关闭自更新，AppImage / 单文件部署最合适。
- 自动更新只替换可执行文件，不处理快捷方式、注册表或系统服务。

## 版本注入

`Makefile` 通过 `-ldflags -X` 注入以下变量：

```go
// internal/version/version.go
Version = "0.0.0-dev"
Commit  = "none"
Date    = "unknown"
```

直接用 `go build` 时保持默认值。

## 平台注意事项

- **Linux 托盘**：依赖桌面环境的 StatusNotifier / AppIndicator 支持。GNOME 默认无托盘，需要安装扩展；无托盘环境下窗口与菜单仍正常工作。
- **Windows**：必须使用 MinGW-w64 gcc 编译 CGO 部分，安装包不依赖 WebView 运行时。
- **macOS**：正式分发需要 Apple Developer ID 签名与公证；自更新已禁用。
- **字体**：Fyne 内置默认字体，缺字时回退系统字体；如需完全控制中文字体，可通过自定义主题替换。

## 签名与分发

### Windows

```bash
signtool sign /fd SHA256 /tr http://timestamp.digicert.com /td SHA256 "build/Trust Anchor.exe"
```

### macOS

```bash
codesign --deep --force --options runtime \
  --sign "Developer ID Application: YourName (TEAMID)" "build/Trust Anchor.app"

xcrun notarytool submit "Trust Anchor.zip" --keychain-profile "notary" --wait
xcrun stapler staple "build/Trust Anchor.app"
```

## 常见问题

| 现象 | 原因 / 处理 |
| --- | --- |
| `pkg-config: command not found` | 缺少 `pkg-config`，按上面的平台命令安装 |
| `fatal error: GL/gl.h: No such file` | 缺少 `libgl1-mesa-dev`（或对应发行版的 mesa 开发包） |
| `fatal error: X11/Xlib.h: No such file` | 缺少 `xorg-dev` / `libX11-devel` |
| 构建报 CGO 相关错误 | 确认 `CGO_ENABLED=1` 且安装了 gcc |
| 首次编译很慢 | Fyne 的 CGO 依赖首次全量编译需要几分钟，之后走缓存 |
| 窗口能开但托盘没反应 | Linux 桌面环境不支持托盘，属预期行为 |

## 根证书

根证书以 PEM 形式内嵌进二进制：`internal/truststore/rootCA.crt`。启动时会校验它必须是自签、且带 `basicConstraints CA:TRUE` 的证书，校验不过则界面提示替换后重新构建。

替换为自有证书并重新构建：

```bash
cp your-root-ca.crt internal/truststore/rootCA.crt   # 必须是 PEM，不要用 DER/.cer
make build
```

安装落点：

| 平台 | 位置 |
| --- | --- |
| Windows | `LocalMachine\Root`（本机所有用户生效，需 UAC） |
| macOS | 系统钥匙串（需管理员授权） |
| Linux | `/usr/local/share/ca-certificates/trustanchor-root-ca.crt` 或发行版对应的 anchors 目录，随后执行 `update-ca-certificates` / `update-ca-trust` / `trust extract-compat` |

> Firefox 使用独立证书库，不受系统信任库影响，需要单独导入或启用企业根证书策略。

