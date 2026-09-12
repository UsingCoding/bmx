package cli

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/UsingCoding/bmx/internal/config"
	"github.com/UsingCoding/bmx/internal/model"
	"github.com/UsingCoding/bmx/internal/state"
)

const coreGroup = "core"

func TestDefaultRegistrySupportsBrewAndBrewCask(t *testing.T) {
	t.Parallel()

	registry := defaultRegistry(io.Discard)
	brewManager, ok := registry["brew"]
	if !ok {
		t.Fatal("brew manager missing")
	}

	brewCaskManager, ok := registry["brew-cask"]
	if !ok {
		t.Fatal("brew-cask manager missing")
	}

	if brewManager != brewCaskManager {
		t.Fatal("brew and brew-cask should share one backend manager")
	}
}

func TestUseStylesDisablesRegularFile(t *testing.T) {
	t.Parallel()

	out, err := os.CreateTemp(t.TempDir(), "output")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := out.Close(); err != nil {
			t.Error(err)
		}
	})

	if useStyles(out) {
		t.Fatal("useStyles() = true for regular file")
	}
}

func TestAddConvergeFlag(t *testing.T) {
	app, err := model.NewApp("brew:age")
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	configPath := filepath.Join(dir, "bmxfile.toml")
	statePath := filepath.Join(dir, "bmxfile.state.toml")
	if err := config.Write(configPath, config.File{
		Lists:  []config.List{{Name: "macos", Groups: []string{coreGroup}}},
		Groups: []config.Group{{Name: coreGroup}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := state.Write(statePath, state.FromApps("macos", []model.App{app})); err != nil {
		t.Fatal(err)
	}

	outputPath, restoreStdout := captureStdout(t)
	err = New("test", "test").Run(context.Background(), []string{
		"bmx", "add", app.Name, "--converge", "--config", configPath, "--state", statePath, "--group", coreGroup,
	})
	restoreStdout()
	if err != nil {
		t.Fatalf("add --converge error = %v", err)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Groups[0].Apps; len(got) != 1 || got[0].App != app {
		t.Fatalf("apps after add = %+v, want [%+v]", got, app)
	}
	assertNoChangesOutput(t, outputPath)
}

func TestRemoveConvergeFlag(t *testing.T) {
	app, err := model.NewApp("brew:age")
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	configPath := filepath.Join(dir, "bmxfile.toml")
	statePath := filepath.Join(dir, "bmxfile.state.toml")
	if err := config.Write(configPath, config.File{
		Lists:  []config.List{{Name: "macos", Groups: []string{coreGroup}}},
		Groups: []config.Group{{Name: coreGroup, Apps: []config.AppEntry{{App: app}}}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := state.Write(statePath, state.FromApps("macos", nil)); err != nil {
		t.Fatal(err)
	}

	outputPath, restoreStdout := captureStdout(t)
	err = New("test", "test").Run(context.Background(), []string{
		"bmx", "rm", app.Name, "-cv", "--config", configPath, "--state", statePath, "--group", coreGroup,
	})
	restoreStdout()
	if err != nil {
		t.Fatalf("rm -cv error = %v", err)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Groups[0].Apps; len(got) != 0 {
		t.Fatalf("apps after remove = %+v, want none", got)
	}
	assertNoChangesOutput(t, outputPath)
}

func captureStdout(t *testing.T) (outputPath string, restore func()) {
	t.Helper()

	output, err := os.CreateTemp(t.TempDir(), "stdout")
	if err != nil {
		t.Fatal(err)
	}
	original := os.Stdout
	os.Stdout = output
	restored := false
	closed := false
	t.Cleanup(func() {
		if !restored {
			os.Stdout = original
		}
		if !closed {
			if err := output.Close(); err != nil {
				t.Error(err)
			}
		}
	})

	return output.Name(), func() {
		os.Stdout = original
		restored = true
		if err := output.Close(); err != nil {
			t.Fatal(err)
		}
		closed = true
	}
}

func assertNoChangesOutput(t *testing.T, outputPath string) {
	t.Helper()

	output, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(output), "No changes.") {
		t.Fatalf("output missing converge result:\n%s", output)
	}
}
