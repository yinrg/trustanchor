package truststore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type ManualCommandError struct{ Command string }

func (e *ManualCommandError) Error() string {
	return "无法自动提权，请以管理员身份执行：" + e.Command
}

func EnsureInstalled() (State, error) {
	if st, err := Status(); err == nil && st.Installed {
		return st, nil
	}
	return runElevated(cmdInstallCA)
}

func EnsureUninstalled() (State, error) {
	if st, err := Status(); err == nil && !st.Installed {
		return st, nil
	}
	return runElevated(cmdUninstallCA)
}

func runElevated(cmd string) (State, error) {
	dir, err := os.MkdirTemp("", "trustanchor-ca-")
	if err != nil {
		return State{}, fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(dir)

	resultPath := filepath.Join(dir, "result.json")
	f, err := os.Create(resultPath)
	if err != nil {
		return State{}, err
	}
	f.Close()

	if err := elevate([]string{cmd, resultFlag, resultPath}); err != nil {
		return State{}, err
	}
	if err := waitResult(resultPath); err != nil {
		return State{}, err
	}

	data, err := os.ReadFile(resultPath)
	if err != nil {
		return State{}, fmt.Errorf("读取提权结果失败: %w", err)
	}
	var res cliResult
	if err := json.Unmarshal(data, &res); err != nil {
		return State{}, fmt.Errorf("解析提权结果失败: %w", err)
	}
	if !res.OK {
		return State{}, errors.New(res.Error)
	}
	return res.State, nil
}

func waitResult(path string) error {
	deadline := time.Now().Add(2 * time.Minute)
	for {
		if fi, err := os.Stat(path); err == nil && fi.Size() > 0 {
			return nil
		}
		if time.Now().After(deadline) {
			return errors.New("等待管理员授权超时")
		}
		time.Sleep(50 * time.Millisecond)
	}
}
