package usecase

import (
	"bytes"
	"context"
	"errors"
	"io"
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
	brewManager = "brew"
	coreGroup   = "core"
	dockerApp   = "brew:docker"
	lazygitApp  = "brew:lazygit"
	macosList   = "macos"
	guiGroup    = "gui"
)

type fakeManager struct {
	checks     []model.App
	installs   []model.App
	uninstalls []model.App
	installed  bool
	statusErr  error
	installErr error
	removeErr  error
}

func (f *fakeManager) Installed(_ context.Context, app model.App) (bool, error) {
	f.checks = append(f.checks, app)
	return f.installed, f.statusErr
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
		Groups: []config.Group{{Name: coreGroup, Apps: []config.AppEntry{{App: mustApp(t, dockerApp)}}}},
	}
	if err := config.Write(configPath, cfg); err != nil {
		t.Fatal(err)
	}

	mgr := &fakeManager{}
	out := &bytes.Buffer{}
	err := Converge(context.Background(), ConvergeInput{
		PlanInput: PlanInput{ConfigPath: configPath, StatePath: statePath, ListName: macosList},
		Managers:  backend.Registry{brewManager: mgr},
		In:        strings.NewReader("y\n"),
		Out:       out,
	})
	if err != nil {
		t.Fatalf("Converge() error = %v", err)
	}
	if len(mgr.installs) != 1 || mgr.installs[0].Name != dockerApp {
		t.Fatalf("unexpected installs: %+v", mgr.installs)
	}

	st, err := state.Load(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if st.ActiveList != macosList {
		t.Fatalf("unexpected active list: %q", st.ActiveList)
	}
	if got := st.InstalledApps(); len(got) != 1 || got[0].Name != dockerApp {
		t.Fatalf("unexpected persisted apps: %+v", got)
	}
}

func TestConvergeSkipsAlreadyInstalledPackageAndWritesState(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "bmxfile.toml")
	statePath := filepath.Join(dir, "bmxfile.state.toml")
	cfg := config.File{
		Lists:  []config.List{{Name: macosList, Groups: []string{coreGroup}}},
		Groups: []config.Group{{Name: coreGroup, Apps: []config.AppEntry{{App: mustApp(t, dockerApp)}}}},
	}
	if err := config.Write(configPath, cfg); err != nil {
		t.Fatal(err)
	}

	mgr := &fakeManager{installed: true}
	out := &bytes.Buffer{}
	err := Converge(context.Background(), ConvergeInput{
		PlanInput: PlanInput{ConfigPath: configPath, StatePath: statePath, ListName: macosList},
		Managers:  backend.Registry{brewManager: mgr},
		In:        strings.NewReader("y\n"),
		Out:       out,
	})
	if err != nil {
		t.Fatalf("Converge() error = %v", err)
	}
	if len(mgr.checks) != 1 || mgr.checks[0].Name != dockerApp {
		t.Fatalf("unexpected checks: %+v", mgr.checks)
	}
	if len(mgr.installs) != 0 {
		t.Fatalf("unexpected installs: %+v", mgr.installs)
	}
	if strings.Contains(out.String(), "Installing brew:docker") {
		t.Fatalf("output included install for an installed app: %s", out.String())
	}
	if !strings.Contains(out.String(), "Converge complete.") {
		t.Fatalf("output missing completion: %s", out.String())
	}

	st, err := state.Load(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if got := st.InstalledApps(); len(got) != 1 || got[0].Name != dockerApp {
		t.Fatalf("unexpected persisted apps: %+v", got)
	}
}

func TestConvergeSupportsBrewCaskThroughBrewManager(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "bmxfile.toml")
	statePath := filepath.Join(dir, "bmxfile.state.toml")

	cfg := config.File{
		Lists:  []config.List{{Name: macosList, Groups: []string{guiGroup}}},
		Groups: []config.Group{{Name: guiGroup, Apps: []config.AppEntry{{App: mustApp(t, "brew-cask:gimp")}}}},
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
		Groups: []config.Group{{Name: coreGroup, Apps: []config.AppEntry{{App: mustApp(t, dockerApp)}}}},
	}
	if err := config.Write(configPath, cfg); err != nil {
		t.Fatal(err)
	}

	original := state.FromApps(macosList, []model.App{mustApp(t, lazygitApp)})
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
		Managers:  backend.Registry{brewManager: mgr},
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

func TestConvergeDoesNotWriteStateOnInstalledCheckFailure(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "bmxfile.toml")
	statePath := filepath.Join(dir, "bmxfile.state.toml")
	cfg := config.File{
		Lists:  []config.List{{Name: macosList, Groups: []string{coreGroup}}},
		Groups: []config.Group{{Name: coreGroup, Apps: []config.AppEntry{{App: mustApp(t, dockerApp)}}}},
	}
	if err := config.Write(configPath, cfg); err != nil {
		t.Fatal(err)
	}

	original := state.FromApps(macosList, []model.App{mustApp(t, lazygitApp)})
	if err := state.Write(statePath, original); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}

	statusErr := errors.New("status check failed")
	mgr := &fakeManager{statusErr: statusErr}
	err = Converge(context.Background(), ConvergeInput{
		PlanInput: PlanInput{ConfigPath: configPath, StatePath: statePath, ListName: macosList},
		Managers:  backend.Registry{brewManager: mgr},
		In:        strings.NewReader("y\n"),
		Out:       &bytes.Buffer{},
	})
	if !errors.Is(err, statusErr) {
		t.Fatalf("Converge() error = %v, want wrapping %v", err, statusErr)
	}
	if len(mgr.installs) != 0 {
		t.Fatalf("unexpected installs: %+v", mgr.installs)
	}

	after, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("state file changed on status check failure\nbefore:\n%s\nafter:\n%s", before, after)
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

//nolint:dupl // Add and Remove exercise distinct public contracts.
func TestAddPreservesCommentedConfig(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "bmxfile.toml")
	input := "# user-owned header\n[[groups]]\nname  = 'core'\ncustom = { enabled = true }\napps = [\n    'brew:git', # retain\n]\n"
	want := "# user-owned header\n[[groups]]\nname  = 'core'\ncustom = { enabled = true }\napps = [\n    'brew:git', # retain\n    \"brew:lazygit\",\n]\n"
	if err := os.WriteFile(path, []byte(input), 0o600); err != nil {
		t.Fatal(err)
	}

	out := &bytes.Buffer{}
	if err := Add(context.Background(), AddInput{
		ConfigPath: path,
		AppName:    lazygitApp,
		GroupName:  coreGroup,
		In:         strings.NewReader(""),
		Out:        out,
	}); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if got := out.String(); got != "Added brew:lazygit to group core. Run `bmx converge` next.\n" {
		t.Fatalf("output = %q", got)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("config bytes differ\nwant:\n%s\ngot:\n%s", want, got)
	}
}

//nolint:dupl // Add and Remove exercise distinct public contracts.
func TestRemovePreservesCommentedConfig(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "bmxfile.toml")
	want := "# user-owned header\n[[groups]]\nname  = 'core'\ncustom = { enabled = true }\napps = [\n    'brew:git', # retain\n]\n"
	input := "# user-owned header\n[[groups]]\nname  = 'core'\ncustom = { enabled = true }\napps = [\n    'brew:git', # retain\n    \"brew:lazygit\",\n]\n"
	if err := os.WriteFile(path, []byte(input), 0o600); err != nil {
		t.Fatal(err)
	}
	out := &bytes.Buffer{}
	if err := Remove(context.Background(), RemoveInput{
		ConfigPath: path,
		AppName:    lazygitApp,
		GroupName:  coreGroup,
		In:         strings.NewReader(""),
		Out:        out,
	}); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if got := out.String(); got != "Removed brew:lazygit from group core. Run `bmx converge` next.\n" {
		t.Fatalf("output = %q", got)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("config bytes differ\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestRemoveMissingAppDoesNotModifyConfig(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "bmxfile.toml")
	input := "[[groups]]\nname = \"core\"\napps = [\"brew:git\"]\n"
	if err := os.WriteFile(path, []byte(input), 0o600); err != nil {
		t.Fatal(err)
	}
	out := &bytes.Buffer{}
	if err := Remove(context.Background(), RemoveInput{
		ConfigPath: path,
		AppName:    lazygitApp,
		GroupName:  coreGroup,
		In:         strings.NewReader(""),
		Out:        out,
	}); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if got := out.String(); got != "brew:lazygit does not exist in group core\n" {
		t.Fatalf("output = %q", got)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != input {
		t.Fatalf("config changed\nwant:\n%s\ngot:\n%s", input, got)
	}
}

func TestConvergeDeclinesWithoutMutatingState(t *testing.T) {
	t.Parallel()

	for name, input := range map[string]io.Reader{
		"negative": strings.NewReader("n\n"),
		"EOF":      strings.NewReader(""),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			configPath := filepath.Join(dir, "bmxfile.toml")
			statePath := filepath.Join(dir, "bmxfile.state.toml")
			cfg := config.File{
				Lists:  []config.List{{Name: macosList, Groups: []string{coreGroup}}},
				Groups: []config.Group{{Name: coreGroup, Apps: []config.AppEntry{{App: mustApp(t, dockerApp)}}}},
			}
			if err := config.Write(configPath, cfg); err != nil {
				t.Fatal(err)
			}

			mgr := &fakeManager{}
			out := &bytes.Buffer{}
			if err := Converge(context.Background(), ConvergeInput{
				PlanInput: PlanInput{ConfigPath: configPath, StatePath: statePath, ListName: macosList},
				Managers:  backend.Registry{brewManager: mgr},
				In:        input,
				Out:       out,
			}); err != nil {
				t.Fatalf("Converge() error = %v", err)
			}
			if got := out.String(); !strings.Contains(got, "Aborted.\n") {
				t.Fatalf("output = %q, want Aborted.", got)
			}
			if len(mgr.checks) != 0 || len(mgr.installs) != 0 || len(mgr.uninstalls) != 0 {
				t.Fatalf("manager mutated: %+v", mgr)
			}
			if _, err := os.Stat(statePath); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("state file exists or could not be checked: %v", err)
			}
		})
	}
}

func TestAddSelectsSecondGroup(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "bmxfile.toml")
	cfg := config.File{
		Groups: []config.Group{{Name: coreGroup}, {Name: guiGroup}},
	}
	if err := config.Write(path, cfg); err != nil {
		t.Fatal(err)
	}

	if err := Add(context.Background(), AddInput{
		ConfigPath: path,
		AppName:    lazygitApp,
		In:         strings.NewReader("j\n"),
		Out:        &bytes.Buffer{},
	}); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	reloaded, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded.Groups[0].Apps) != 0 {
		t.Fatalf("first group apps = %+v, want empty", reloaded.Groups[0].Apps)
	}
	if got := reloaded.Groups[1].Apps; len(got) != 1 || got[0].App.Name != lazygitApp {
		t.Fatalf("second group apps = %+v, want %s", got, lazygitApp)
	}
}
func TestListRenderingStyles(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "bmxfile.toml")
	cfg := config.File{
		Lists: []config.List{
			{Name: macosList, Groups: []string{coreGroup, guiGroup}},
			{Name: "core-only", Groups: []string{coreGroup}},
		},
		Groups: []config.Group{
			{Name: coreGroup, Apps: []config.AppEntry{{App: mustApp(t, "brew:age")}, {App: mustApp(t, "brew:jq")}}},
			{Name: guiGroup, Apps: []config.AppEntry{{App: mustApp(t, "brew-cask:gimp")}, {App: mustApp(t, "brew:jq")}}},
		},
	}
	if err := config.Write(path, cfg); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		listName  string
		useStyles bool
		want      string
	}{
		{
			name: "full plain",
			want: "Lists:\n- macos: core, gui\n- core-only: core\nGroups:\n- core\n  - brew:age\n  - brew:jq\n- gui\n  - brew-cask:gimp\n  - brew:jq\n",
		},
		{
			name:      "full styled",
			useStyles: true,
			want:      "\x1b[36mLists:\x1b[0m\n- \x1b[3mmacos\x1b[0m: \x1b[3mcore\x1b[0m, \x1b[3mgui\x1b[0m\n- \x1b[3mcore-only\x1b[0m: \x1b[3mcore\x1b[0m\n\x1b[35mGroups:\x1b[0m\n- \x1b[3mcore\x1b[0m\n  - \x1b[1mbrew:age\x1b[0m\n  - \x1b[1mbrew:jq\x1b[0m\n- \x1b[3mgui\x1b[0m\n  - \x1b[1mbrew-cask:gimp\x1b[0m\n  - \x1b[1mbrew:jq\x1b[0m\n",
		},
		{
			name:     "selected plain",
			listName: macosList,
			want:     "List macos\n- brew:age\n- brew:jq\n- brew-cask:gimp\n",
		},
		{
			name:      "selected styled",
			listName:  macosList,
			useStyles: true,
			want:      "\x1b[36mList\x1b[0m \x1b[3mmacos\x1b[0m\n- \x1b[1mbrew:age\x1b[0m\n- \x1b[1mbrew:jq\x1b[0m\n- \x1b[1mbrew-cask:gimp\x1b[0m\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer
			if err := List(context.Background(), ListInput{
				ConfigPath: path,
				ListName:   test.listName,
				Out:        &out,
				UseStyles:  test.useStyles,
			}); err != nil {
				t.Fatalf("List() error = %v", err)
			}
			if got := out.String(); got != test.want {
				t.Fatalf("List() output = %q, want %q", got, test.want)
			}
			if !test.useStyles && strings.Contains(out.String(), "\x1b") {
				t.Fatalf("plain output contains ANSI escape: %q", out.String())
			}
		})
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
