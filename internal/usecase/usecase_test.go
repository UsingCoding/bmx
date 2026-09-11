package usecase

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/UsingCoding/bmx/internal/backend"
	"github.com/UsingCoding/bmx/internal/config"
	"github.com/UsingCoding/bmx/internal/model"
	"github.com/UsingCoding/bmx/internal/state"
)

const (
	coreGroup = "core"
	macosList = "macos"
)

type fakeManager struct {
	installs   []model.App
	uninstalls []model.App
	installErr error
	removeErr  error
}

func (f *fakeManager) Install(_ context.Context, app model.App) error {
	f.installs = append(f.installs, app)
	return f.installErr
}

func (f *fakeManager) Uninstall(_ context.Context, app model.App) error {
	f.uninstalls = append(f.uninstalls, app)
	return f.removeErr
}

func TestInitCreatesTemplateAndAdvice(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "bmxfile.toml")
	out := &bytes.Buffer{}

	if err := Init(context.Background(), InitInput{ConfigPath: configPath, Out: out}); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	if !strings.Contains(text, "# BMX desired-state config.") {
		t.Fatalf("template missing header comment:\n%s", text)
	}
	if !strings.Contains(text, "brew:age") {
		t.Fatalf("template missing starter package:\n%s", text)
	}
	if !strings.Contains(text, "brew-cask:gimp") {
		t.Fatalf("template missing brew-cask example:\n%s", text)
	}
	if !strings.Contains(out.String(), "bmx add brew:age") {
		t.Fatalf("output missing next-step advice: %s", out.String())
	}
}

func TestInitDoesNotOverwriteExistingConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "bmxfile.toml")
	if err := os.WriteFile(configPath, []byte("existing\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := Init(context.Background(), InitInput{ConfigPath: configPath, Out: &bytes.Buffer{}})
	if err == nil {
		t.Fatal("expected init error")
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "existing\n" {
		t.Fatalf("init overwrote config: %q", string(content))
	}
}

func TestConvergeAppliesPlanAndWritesState(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "bmxfile.toml")
	statePath := filepath.Join(dir, "bmxfile.state.toml")

	cfg := config.File{
		Lists:  []config.List{{Name: macosList, Groups: []string{coreGroup}}},
		Groups: []config.Group{{Name: coreGroup, Apps: []config.AppEntry{{App: mustApp(t, "brew:docker")}}}},
	}
	if err := config.Write(configPath, cfg); err != nil {
		t.Fatal(err)
	}

	mgr := &fakeManager{}
	out := &bytes.Buffer{}
	err := Converge(context.Background(), ConvergeInput{
		PlanInput: PlanInput{ConfigPath: configPath, StatePath: statePath, ListName: macosList},
		Managers:  backend.Registry{"brew": mgr},
		In:        strings.NewReader("y\n"),
		Out:       out,
	})
	if err != nil {
		t.Fatalf("Converge() error = %v", err)
	}
	if len(mgr.installs) != 1 || mgr.installs[0].Name != "brew:docker" {
		t.Fatalf("unexpected installs: %+v", mgr.installs)
	}

	st, err := state.Load(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if st.ActiveList != macosList {
		t.Fatalf("unexpected active list: %q", st.ActiveList)
	}
	if got := st.InstalledApps(); len(got) != 1 || got[0].Name != "brew:docker" {
		t.Fatalf("unexpected persisted apps: %+v", got)
	}
}

func TestConvergeSupportsBrewCaskThroughBrewManager(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "bmxfile.toml")
	statePath := filepath.Join(dir, "bmxfile.state.toml")

	cfg := config.File{
		Lists:  []config.List{{Name: macosList, Groups: []string{"gui"}}},
		Groups: []config.Group{{Name: "gui", Apps: []config.AppEntry{{App: mustApp(t, "brew-cask:gimp")}}}},
	}
	if err := config.Write(configPath, cfg); err != nil {
		t.Fatal(err)
	}

	mgr := &fakeManager{}
	err := Converge(context.Background(), ConvergeInput{
		PlanInput: PlanInput{ConfigPath: configPath, StatePath: statePath, ListName: macosList},
		Managers:  backend.Registry{"brew-cask": mgr},
		In:        strings.NewReader("y\n"),
		Out:       &bytes.Buffer{},
	})
	if err != nil {
		t.Fatalf("Converge() error = %v", err)
	}
	if len(mgr.installs) != 1 || mgr.installs[0].Name != "brew-cask:gimp" {
		t.Fatalf("unexpected cask installs: %+v", mgr.installs)
	}
}

func TestConvergeDoesNotWriteStateOnFailure(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "bmxfile.toml")
	statePath := filepath.Join(dir, "bmxfile.state.toml")

	cfg := config.File{
		Lists:  []config.List{{Name: macosList, Groups: []string{coreGroup}}},
		Groups: []config.Group{{Name: coreGroup, Apps: []config.AppEntry{{App: mustApp(t, "brew:docker")}}}},
	}
	if err := config.Write(configPath, cfg); err != nil {
		t.Fatal(err)
	}

	original := state.FromApps(macosList, []model.App{mustApp(t, "brew:lazygit")})
	if err := state.Write(statePath, original); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}

	mgr := &fakeManager{installErr: errors.New("boom")}
	err = Converge(context.Background(), ConvergeInput{
		PlanInput: PlanInput{ConfigPath: configPath, StatePath: statePath, ListName: macosList},
		Managers:  backend.Registry{"brew": mgr},
		In:        strings.NewReader("y\n"),
		Out:       &bytes.Buffer{},
	})
	if err == nil {
		t.Fatal("expected converge error")
	}

	after, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("state file changed on failure\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestAddDuplicateDoesNotModifyConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "bmxfile.toml")
	cfg := config.File{
		Lists:  []config.List{{Name: macosList, Groups: []string{coreGroup}}},
		Groups: []config.Group{{Name: coreGroup, Apps: []config.AppEntry{{App: mustApp(t, "brew:codex")}}}},
	}
	if err := config.Write(configPath, cfg); err != nil {
		t.Fatal(err)
	}

	out := &bytes.Buffer{}
	if err := Add(context.Background(), AddInput{
		ConfigPath: configPath,
		AppName:    "brew:codex",
		GroupName:  coreGroup,
		In:         strings.NewReader(""),
		Out:        out,
	}); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	reloaded, err := config.Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cfg, reloaded) {
		t.Fatalf("config changed on duplicate\nwant: %#v\ngot: %#v", cfg, reloaded)
	}
}

func mustApp(t *testing.T, name string) model.App {
	t.Helper()
	app, err := model.NewApp(name)
	if err != nil {
		t.Fatal(err)
	}
	return app
}
