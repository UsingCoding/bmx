package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/UsingCoding/bmx/internal/model"
)

func TestLoadParsesListsGroupsAndObjectApps(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "bmxfile.toml")
	content := `[[lists]]
name = "macos"
groups = ["core", "gui"]

[[groups]]
name = "core"
apps = ["brew:docker", "brew:lazygit"]

[[groups]]
name = "gui"
apps = [
  { name = "brew-cask:gimp" },
]
`
	if err := osWriteFile(path, content); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(cfg.Lists) != 1 {
		t.Fatalf("expected 1 list, got %d", len(cfg.Lists))
	}
	if len(cfg.Groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(cfg.Groups))
	}
	if got := cfg.Groups[1].Apps[0].App; got.Name != "brew-cask:gimp" || got.Manager != "brew-cask" {
		t.Fatalf("unexpected parsed object app: %+v", got)
	}
}

func TestLoadRejectsLegacyCaskFlag(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "bmxfile.toml")
	content := `[[lists]]
name = "macos"
groups = ["gui"]

[[groups]]
name = "gui"
apps = [
  { name = "brew:gimp", cask = true },
]
`
	if err := osWriteFile(path, content); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "brew-cask") {
		t.Fatalf("expected legacy cask flag error, got %v", err)
	}
}

func TestResolveListDeduplicatesApps(t *testing.T) {
	t.Parallel()

	docker, err := model.NewApp("brew:docker")
	if err != nil {
		t.Fatal(err)
	}
	gimp, err := model.NewApp("brew-cask:gimp")
	if err != nil {
		t.Fatal(err)
	}

	cfg := File{
		Lists: []List{{Name: "macos", Groups: []string{"core", "desktop"}}},
		Groups: []Group{
			{Name: "core", Apps: []AppEntry{{App: docker}}},
			{Name: "desktop", Apps: []AppEntry{{App: docker}, {App: gimp}}},
		},
	}

	resolved, err := ResolveList(cfg, "macos")
	if err != nil {
		t.Fatalf("ResolveList() error = %v", err)
	}
	if len(resolved) != 2 {
		t.Fatalf("expected 2 apps after dedupe, got %d", len(resolved))
	}
	if resolved[0].Name != "brew:docker" || resolved[1].Name != "brew-cask:gimp" {
		t.Fatalf("unexpected resolve order: %+v", resolved)
	}
}

func TestValidateRejectsUnknownGroup(t *testing.T) {
	t.Parallel()

	cfg := File{
		Lists:  []List{{Name: "macos", Groups: []string{"missing"}}},
		Groups: []Group{{Name: "core"}},
	}

	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error")
	}
}

func osWriteFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o600)
}
