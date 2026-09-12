package brew

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/UsingCoding/bmx/internal/model"
)

func TestManagerInstalled(t *testing.T) {
	app := model.App{Name: "brew:jq", Manager: "brew", Package: "jq"}
	caskApp := model.App{Name: "brew-cask:jq", Manager: "brew-cask", Package: "jq"}

	t.Run("formula installed", func(t *testing.T) {
		logPath := setupFakeBrew(t)
		t.Setenv("BMX_FORMULA_EXIT", "0")

		installed, err := New().Installed(context.Background(), app)
		if err != nil {
			t.Fatalf("Installed() error = %v", err)
		}
		if !installed {
			t.Fatal("Installed() installed = false, want true")
		}
		if got := readBrewLog(t, logPath); got != "list --formula jq" {
			t.Fatalf("brew commands = %q, want %q", got, "list --formula jq")
		}
	})

	t.Run("cask installed", func(t *testing.T) {
		logPath := setupFakeBrew(t)
		t.Setenv("BMX_CASK_EXIT", "0")

		installed, err := New().Installed(context.Background(), caskApp)
		if err != nil {
			t.Fatalf("Installed() error = %v", err)
		}
		if !installed {
			t.Fatal("Installed() installed = false, want true")
		}
		if got := readBrewLog(t, logPath); got != "list --cask jq" {
			t.Fatalf("brew commands = %q, want %q", got, "list --cask jq")
		}
	})

	t.Run("not installed", func(t *testing.T) {
		logPath := setupFakeBrew(t)
		t.Setenv("BMX_FORMULA_EXIT", "1")

		installed, err := New().Installed(context.Background(), app)
		if err != nil {
			t.Fatalf("Installed() error = %v", err)
		}
		if installed {
			t.Fatal("Installed() installed = true, want false")
		}
		if got := readBrewLog(t, logPath); got != "list --formula jq" {
			t.Fatalf("brew commands = %q, want %q", got, "list --formula jq")
		}
	})

	t.Run("missing executable", func(t *testing.T) {
		t.Setenv("PATH", t.TempDir())

		installed, err := New().Installed(context.Background(), app)
		if err == nil || !errors.Is(err, exec.ErrNotFound) {
			t.Fatalf("Installed() error = %v, want wrapping exec.ErrNotFound", err)
		}
		if installed {
			t.Fatal("Installed() installed = true, want false")
		}
	})

	t.Run("canceled context", func(t *testing.T) {
		setupFakeBrew(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		installed, err := New().Installed(ctx, app)
		if err == nil || !errors.Is(err, context.Canceled) {
			t.Fatalf("Installed() error = %v, want wrapping context.Canceled", err)
		}
		if installed {
			t.Fatal("Installed() installed = true, want false")
		}
	})
}

func setupFakeBrew(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	logPath := filepath.Join(dir, "brew.log")
	scriptPath := filepath.Join(dir, "brew")
	script := `#!/bin/sh
echo "$*" >> "$BMX_BREW_LOG"
case "$2" in
  --formula) exit "${BMX_FORMULA_EXIT:-1}" ;;
  --cask) exit "${BMX_CASK_EXIT:-1}" ;;
esac
exit 1
`
	//nolint:gosec // The test script must be executable.
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BMX_BREW_LOG", logPath)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	return logPath
}

func readBrewLog(t *testing.T, path string) string {
	t.Helper()

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return strings.TrimSuffix(string(contents), "\n")
}
