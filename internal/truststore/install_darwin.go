//go:build darwin

package truststore

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	keychainPath  = "/Library/Keychains/System.keychain"
	storeLocation = "/Library/Keychains/System.keychain（系统钥匙串）"
)

func queryStatus(cert *x509.Certificate, der []byte) (State, error) {
	args := []string{"find-certificate", "-a"}
	if cn := cert.Subject.CommonName; cn != "" {
		args = append(args, "-c", cn)
	}
	args = append(args, "-p", keychainPath)

	out, err := exec.Command("/usr/bin/security", args...).Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return State{Location: storeLocation}, nil
		}
		return State{}, fmt.Errorf("读取系统钥匙串失败: %w", err)
	}

	rest := out
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			return State{Location: storeLocation}, nil
		}
		if bytes.Equal(block.Bytes, der) {
			return State{Installed: true, Location: storeLocation}, nil
		}
	}
}

func installCert(der []byte) error {
	path, cleanup, err := tempPEMFile(der)
	if err != nil {
		return err
	}
	defer cleanup()

	out, err := exec.Command("/usr/bin/security",
		"add-trusted-cert", "-d", "-r", "trustRoot", "-p", "ssl", "-k", keychainPath, path).CombinedOutput()
	if err != nil {
		return fmt.Errorf("写入系统钥匙串失败: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func uninstallCert(der []byte) error {
	out, err := exec.Command("/usr/bin/security",
		"delete-certificate", "-Z", fingerprintHex(der), keychainPath).CombinedOutput()
	if err != nil {
		return fmt.Errorf("删除根证书失败: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func elevate(args []string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("定位可执行文件失败: %w", err)
	}
	line := shellJoin(append([]string{exe}, args...))
	script := fmt.Sprintf("do shell script %s with administrator privileges", appleScriptString(line))

	out, err := exec.Command("/usr/bin/osascript", "-e", script).CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "-128") {
			return ErrCancelled
		}
		return fmt.Errorf("请求管理员授权失败: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func appleScriptString(s string) string {
	escaped := strings.ReplaceAll(s, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	return `"` + escaped + `"`
}
