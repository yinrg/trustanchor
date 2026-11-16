package version

import (
	"fmt"
	"runtime"
)

var (
	Version = "0.0.0-dev"
	Commit  = "none"
	Date    = "unknown"
)

func Platform() string {
	return runtime.GOOS + "/" + runtime.GOARCH
}

func Summary() string {
	return fmt.Sprintf("%s (%s, %s)", Version, Commit, Date)
}
