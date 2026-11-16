package truststore

import (
	"encoding/json"
	"fmt"
	"os"
)

const (
	cmdInstallCA   = "--install-ca"
	cmdUninstallCA = "--uninstall-ca"
	cmdCAStatus    = "--ca-status"
	resultFlag     = "--result"
)

type cliResult struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
	State State  `json:"state"`
}

func RunCLI(args []string) (bool, int) {
	if len(args) == 0 {
		return false, 0
	}
	switch args[0] {
	case cmdInstallCA, cmdUninstallCA, cmdCAStatus:
	default:
		return false, 0
	}

	var resultPath string
	for i := 1; i < len(args); i++ {
		if args[i] == resultFlag && i+1 < len(args) {
			resultPath = args[i+1]
			i++
		}
	}

	var (
		st  State
		err error
	)
	switch args[0] {
	case cmdInstallCA:
		st, err = Install()
	case cmdUninstallCA:
		st, err = Uninstall()
	default:
		st, err = Status()
	}

	if resultPath != "" {
		writeResult(resultPath, cliResult{OK: err == nil, Error: msgOf(err), State: st})
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return true, 1
	}
	if resultPath == "" {
		printState(st)
	}
	return true, 0
}

func writeResult(path string, res cliResult) {
	data, err := json.Marshal(res)
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o644)
}

func printState(st State) {
	fmt.Printf("主体: %s\n", st.Subject)
	fmt.Printf("SHA-256 指纹: %s\n", st.Fingerprint)
	fmt.Printf("有效期: %s 至 %s\n", st.NotBefore.Format("2006-01-02"), st.NotAfter.Format("2006-01-02"))
	if st.Installed {
		fmt.Printf("状态: 已安装到 %s\n", st.Location)
		return
	}
	fmt.Printf("状态: 未安装（目标位置 %s）\n", st.Location)
}

func msgOf(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
