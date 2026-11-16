package updater

import (
	"context"
	"errors"
	"testing"
)

func TestEnabled(t *testing.T) {
	original := RepositorySlug
	defer func() { RepositorySlug = original }()

	RepositorySlug = ""
	if Enabled() {
		t.Fatal("empty slug should disable the updater")
	}

	RepositorySlug = "owner/repo"
	if !Enabled() {
		t.Fatal("non-empty slug should enable the updater")
	}
}

func TestCheckDisabled(t *testing.T) {
	original := RepositorySlug
	defer func() { RepositorySlug = original }()
	RepositorySlug = ""

	if _, err := Check(context.Background()); !errors.Is(err, ErrDisabled) {
		t.Fatalf("Check() error = %v, want ErrDisabled", err)
	}
}

func TestCanSelfUpdate(t *testing.T) {
	cases := map[string]bool{
		"windows": true,
		"linux":   true,
		"darwin":  false,
	}
	for goos, want := range cases {
		if got := canSelfUpdate(goos); got != want {
			t.Errorf("canSelfUpdate(%q) = %v, want %v", goos, got, want)
		}
	}
}
