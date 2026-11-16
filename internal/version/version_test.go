package version

import (
	"strings"
	"testing"
)

func TestPlatform(t *testing.T) {
	platform := Platform()
	if !strings.Contains(platform, "/") {
		t.Fatalf("Platform() = %q, want os/arch", platform)
	}
}

func TestSummary(t *testing.T) {
	summary := Summary()
	if !strings.Contains(summary, Version) {
		t.Fatalf("Summary() = %q, want it to contain version %q", summary, Version)
	}
}
