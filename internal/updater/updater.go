package updater

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/Masterminds/semver/v3"
	"github.com/creativeprojects/go-selfupdate"

	"github.com/yinrg/trustanchor/internal/version"
)

var RepositorySlug = "yinrg/trustanchor"

var (
	ErrDisabled     = errors.New("更新功能未启用")
	ErrManualUpdate = errors.New("当前平台需要手动下载安装包")
)

type Update struct {
	Version string
	Notes   string
	URL     string
	release *selfupdate.Release
}

func Enabled() bool {
	return RepositorySlug != ""
}

func Check(ctx context.Context) (*Update, error) {
	if !Enabled() {
		return nil, ErrDisabled
	}

	current, err := semver.NewVersion(version.Version)
	if err != nil {
		current = semver.MustParse("0.0.0")
	}

	latest, found, err := selfupdate.DefaultUpdater().DetectLatest(ctx, selfupdate.ParseSlug(RepositorySlug))
	if err != nil {
		return nil, fmt.Errorf("获取最新版本失败: %w", err)
	}
	if !found {
		return nil, nil
	}

	latestVersion, err := semver.NewVersion(latest.Version())
	if err != nil {
		return nil, fmt.Errorf("解析版本号 %q 失败: %w", latest.Version(), err)
	}
	if !latestVersion.GreaterThan(current) {
		return nil, nil
	}

	return &Update{
		Version: latest.Version(),
		Notes:   latest.ReleaseNotes,
		URL:     latest.URL,
		release: latest,
	}, nil
}

func (u *Update) CanSelfUpdate() bool {
	return canSelfUpdate(runtime.GOOS)
}

func (u *Update) Install(ctx context.Context) error {
	if !u.CanSelfUpdate() {
		return ErrManualUpdate
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("定位当前可执行文件失败: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}

	if err := selfupdate.DefaultUpdater().UpdateTo(ctx, u.release, exe); err != nil {
		return fmt.Errorf("安装更新失败: %w", err)
	}
	return nil
}

func canSelfUpdate(goos string) bool {
	return goos != "darwin"
}
