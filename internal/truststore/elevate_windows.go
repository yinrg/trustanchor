//go:build windows

package truststore

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/windows"
)

func IsElevated() bool {
	return windows.GetCurrentProcessToken().IsElevated()
}

func elevate(args []string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("定位可执行文件失败: %w", err)
	}
	verb, err := windows.UTF16PtrFromString("runas")
	if err != nil {
		return err
	}
	file, err := windows.UTF16PtrFromString(exe)
	if err != nil {
		return err
	}
	argLine, err := windows.UTF16PtrFromString(quoteArgs(args))
	if err != nil {
		return err
	}
	if err := windows.ShellExecute(0, verb, file, argLine, nil, windows.SW_SHOWNORMAL); err != nil {
		if errors.Is(err, windows.ERROR_CANCELLED) {
			return ErrCancelled
		}
		return fmt.Errorf("请求管理员权限失败: %w", err)
	}
	return nil
}

func quoteArgs(args []string) string {
	quoted := make([]string, len(args))
	for i, a := range args {
		if strings.ContainsAny(a, " \t") {
			quoted[i] = `"` + a + `"`
			continue
		}
		quoted[i] = a
	}
	return strings.Join(quoted, " ")
}
