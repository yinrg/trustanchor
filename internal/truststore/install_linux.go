//go:build linux

package truststore

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const anchorName = "trustanchor-root-ca.crt"

type layout struct {
	anchorDir string
	refresh   []string
}

func detectLayout() (layout, error) {
	if tool := toolPath("/usr/bin/update-ca-trust", "/usr/sbin/update-ca-trust", "update-ca-trust"); tool != "" &&
		dirExists("/etc/pki/ca-trust/source/anchors") {
		return layout{"/etc/pki/ca-trust/source/anchors", []string{tool, "extract"}}, nil
	}
	if tool := toolPath("/usr/bin/trust", "trust"); tool != "" &&
		dirExists("/etc/ca-certificates/trust-source/anchors") {
		return layout{"/etc/ca-certificates/trust-source/anchors", []string{tool, "extract-compat"}}, nil
	}
	if tool := toolPath("/usr/sbin/update-ca-certificates", "/usr/bin/update-ca-certificates", "update-ca-certificates"); tool != "" {
		if dirExists("/etc/pki/trust/anchors") && !fileExists("/etc/debian_version") {
			return layout{"/etc/pki/trust/anchors", []string{tool}}, nil
		}
		return layout{"/usr/local/share/ca-certificates", []string{tool}}, nil
	}
	return layout{}, errors.New("未找到受支持的信任库工具（update-ca-certificates / update-ca-trust / trust）")
}

func (l layout) anchorPath() string {
	return filepath.Join(l.anchorDir, anchorName)
}

func queryStatus(_ *x509.Certificate, der []byte) (State, error) {
	lay, err := detectLayout()
	if err != nil {
		return State{}, err
	}
	path := lay.anchorPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return State{Location: lay.anchorDir}, nil
		}
		return State{}, err
	}
	block, _ := pem.Decode(data)
	if block == nil || !bytes.Equal(block.Bytes, der) {
		return State{Location: lay.anchorDir}, nil
	}
	return State{Installed: true, Location: path}, nil
}

func installCert(der []byte) error {
	lay, err := detectLayout()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(lay.anchorDir, 0o755); err != nil {
		return fmt.Errorf("创建 %s 失败: %w", lay.anchorDir, err)
	}
	path := lay.anchorPath()
	if err := os.WriteFile(path, pemBytes(der), 0o644); err != nil {
		return fmt.Errorf("写入 %s 失败: %w", path, err)
	}
	return refresh(lay)
}

func uninstallCert(der []byte) error {
	lay, err := detectLayout()
	if err != nil {
		return err
	}
	if err := os.Remove(lay.anchorPath()); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrNotInstalled
		}
		return fmt.Errorf("删除 %s 失败: %w", lay.anchorPath(), err)
	}
	return refresh(lay)
}

func refresh(lay layout) error {
	out, err := exec.Command(lay.refresh[0], lay.refresh[1:]...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("刷新系统证书库失败: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func elevate(args []string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("定位可执行文件失败: %w", err)
	}
	if _, err := exec.LookPath("pkexec"); err != nil {
		return &ManualCommandError{Command: "sudo " + shellJoin([]string{exe, args[0]})}
	}
	out, err := exec.Command("pkexec", append([]string{exe}, args...)...).CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && (exitErr.ExitCode() == 126 || exitErr.ExitCode() == 127) {
			return ErrCancelled
		}
		return fmt.Errorf("管理员授权失败: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func toolPath(candidates ...string) string {
	for _, c := range candidates {
		if strings.ContainsRune(c, os.PathSeparator) {
			if fi, err := os.Stat(c); err == nil && !fi.IsDir() {
				return c
			}
			continue
		}
		if p, err := exec.LookPath(c); err == nil {
			return p
		}
	}
	return ""
}

func dirExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}

func fileExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && !fi.IsDir()
}
