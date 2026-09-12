package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/UsingCoding/bmx/internal/model"
)

const coreGroup = "core"

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
		Lists: []List{{Name: "macos", Groups: []string{coreGroup, "desktop"}}},
		Groups: []Group{
			{Name: coreGroup, Apps: []AppEntry{{App: docker}}},
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
		Groups: []Group{{Name: coreGroup}},
	}

	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error")
	}
}

func osWriteFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o600)
}

func TestAppendAppPreservesInlineConfigBytes(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "bmxfile.toml")
	input := "# header\n[[groups]]\nname  = 'core' # keep\ncustom = { enabled = true }\napps = [ 'brew:git' ] # keep"
	want := "# header\n[[groups]]\nname  = 'core' # keep\ncustom = { enabled = true }\napps = [ 'brew:git', \"brew:lazygit\" ] # keep"
	writeConfigBytes(t, path, []byte(input), 0o640)

	appendConfigApp(t, path, "brew:lazygit")

	assertConfigBytesAndApp(t, path, want, "brew:lazygit")
}

func TestAppendAppPreservesMultilineLayout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "two spaces with trailing comma",
			input: "[[groups]]\nname = \"core\"\napps = [\n  \"brew:git\",\n]\n",
			want:  "[[groups]]\nname = \"core\"\napps = [\n  \"brew:git\",\n  \"brew:lazygit\",\n]\n",
		},
		{
			name:  "four spaces without trailing comma",
			input: "[[groups]]\nname = \"core\"\napps = [\n    \"brew:git\"\n]\n",
			want:  "[[groups]]\nname = \"core\"\napps = [\n    \"brew:git\",\n    \"brew:lazygit\"\n]\n",
		},
		{
			name:  "tabs",
			input: "[[groups]]\nname = \"core\"\napps = [\n\t\"brew:git\",\n]\n",
			want:  "[[groups]]\nname = \"core\"\napps = [\n\t\"brew:git\",\n\t\"brew:lazygit\",\n]\n",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "bmxfile.toml")
			writeConfigBytes(t, path, []byte(test.input), 0o600)
			appendConfigApp(t, path, "brew:lazygit")
			assertConfigBytesAndApp(t, path, test.want, "brew:lazygit")
		})
	}
}

func TestAppendAppPreservesCRLFObjectsAndComments(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "bmxfile.toml")
	input := "[[groups]]\r\nname = \"core\"\r\napps = [\r\n    { name = \"brew:git\" }, # object comment\r\n    # keep before the new value\r\n]\r\n"
	want := "[[groups]]\r\nname = \"core\"\r\napps = [\r\n    { name = \"brew:git\" }, # object comment\r\n    # keep before the new value\r\n    \"brew:lazygit\",\r\n]\r\n"
	writeConfigBytes(t, path, []byte(input), 0o640)

	appendConfigApp(t, path, "brew:lazygit")

	assertConfigBytesAndApp(t, path, want, "brew:lazygit")
}

func TestAppendAppAddsToEmptyAndMissingArrays(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty inline",
			input: "[[groups]]\nname = \"core\"\napps = [ ]\n",
			want:  "[[groups]]\nname = \"core\"\napps = [ \"brew:lazygit\"]\n",
		},
		{
			name:  "empty multiline with comment indentation",
			input: "[[groups]]\nname = \"core\"\napps = [\n    # retained\n]\n",
			want:  "[[groups]]\nname = \"core\"\napps = [\n    # retained\n    \"brew:lazygit\"\n]\n",
		},
		{
			name:  "missing apps",
			input: "[[groups]]\nname = \"core\"\ncustom = true\n\n# next section\n[[groups]]\nname = \"other\"\napps = []\n",
			want:  "[[groups]]\nname = \"core\"\ncustom = true\napps = [\"brew:lazygit\"]\n\n# next section\n[[groups]]\nname = \"other\"\napps = []\n",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "bmxfile.toml")
			writeConfigBytes(t, path, []byte(test.input), 0o600)
			appendConfigApp(t, path, "brew:lazygit")
			assertConfigBytesAndApp(t, path, test.want, "brew:lazygit")
		})
	}
}

func TestAppendAppFailurePreservesFileAndMode(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "bmxfile.toml")
	input := "[[groups]]\nname = \"core\"\napps = [\n  \"brew:git\", ]\n"
	writeConfigBytes(t, path, []byte(input), 0o640)

	app, err := model.NewApp("brew:lazygit")
	if err != nil {
		t.Fatal(err)
	}
	if err := AppendApp(path, coreGroup, app); err == nil {
		t.Fatal("AppendApp() error = nil")
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != input {
		t.Fatalf("config changed on failed append\nwant:\n%s\ngot:\n%s", input, got)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("mode = %o, want 640", info.Mode().Perm())
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".bmx-config-") {
			t.Fatalf("unexpected temporary config file %q", entry.Name())
		}
	}
}
func TestAppendAppRejectsInvalidConfigWithoutMutation(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "bmxfile.toml")
	input := "[[groups]]\nname = \"core\"\napps = [\"brew:git\"\n"
	writeConfigBytes(t, path, []byte(input), 0o640)

	app, err := model.NewApp("brew:lazygit")
	if err != nil {
		t.Fatal(err)
	}
	if err := AppendApp(path, coreGroup, app); err == nil {
		t.Fatal("AppendApp() error = nil")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != input {
		t.Fatalf("config changed on parse failure\nwant:\n%s\ngot:\n%s", input, got)
	}
}

func TestRemoveAppPreservesInlineLayout(t *testing.T) {
	t.Parallel()

	const apps = "[[groups]]\nname = \"core\"\napps = [ \"brew:a\", 'brew:b', \"brew:c\" ]\n"
	tests := []struct {
		name   string
		input  string
		remove string
		want   string
	}{
		{"first", apps, "brew:a", "[[groups]]\nname = \"core\"\napps = [ 'brew:b', \"brew:c\" ]\n"},
		{"middle", apps, "brew:b", "[[groups]]\nname = \"core\"\napps = [ \"brew:a\", \"brew:c\" ]\n"},
		{"last", apps, "brew:c", "[[groups]]\nname = \"core\"\napps = [ \"brew:a\", 'brew:b' ]\n"},
		{"only", "[[groups]]\nname = \"core\"\napps = [ \"brew:a\" ]\n", "brew:a", "[[groups]]\nname = \"core\"\napps = [  ]\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "bmxfile.toml")
			writeConfigBytes(t, path, []byte(test.input), 0o600)
			removeConfigApp(t, path, test.remove)
			assertConfigBytes(t, path, test.want)
		})
	}
}

func TestRemoveAppPreservesMultilineCommentsAndCRLFObject(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "bmxfile.toml")
	input := "[[groups]]\r\nname = \"core\"\r\napps = [\r\n    { name = \"brew:a\" }, # keep\r\n    \"brew:b\",\r\n    # before closing\r\n]\r\n"
	want := "[[groups]]\r\nname = \"core\"\r\napps = [\r\n    # keep\r\n    \"brew:b\",\r\n    # before closing\r\n]\r\n"
	writeConfigBytes(t, path, []byte(input), 0o640)
	removeConfigApp(t, path, "brew:a")
	assertConfigBytes(t, path, want)
}
func TestRemoveAppPreservesMultilineWithoutTrailingComma(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "bmxfile.toml")
	input := "[[groups]]\nname = \"core\"\napps = [\n  \"brew:a\",\n  \"brew:b\"\n]\n"
	want := "[[groups]]\nname = \"core\"\napps = [\n  \"brew:a\",\n]\n"
	writeConfigBytes(t, path, []byte(input), 0o600)
	removeConfigApp(t, path, "brew:b")
	assertConfigBytes(t, path, want)
}

func TestRemoveAppFailurePreservesFileAndMode(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "bmxfile.toml")
	input := "[[groups]]\nname = \"core\"\napps = [\"brew:a\"]\n"
	writeConfigBytes(t, path, []byte(input), 0o640)
	app, err := model.NewApp("brew:missing")
	if err != nil {
		t.Fatal(err)
	}
	if err := RemoveApp(path, coreGroup, app); err == nil {
		t.Fatal("RemoveApp() error = nil")
	}
	assertConfigBytes(t, path, input)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("mode = %o, want 640", info.Mode().Perm())
	}
}

func removeConfigApp(t *testing.T, path, name string) {
	t.Helper()
	app, err := model.NewApp(name)
	if err != nil {
		t.Fatal(err)
	}
	if err := RemoveApp(path, coreGroup, app); err != nil {
		t.Fatalf("RemoveApp() error = %v", err)
	}
}

func assertConfigBytes(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("config bytes differ\nwant:\n%s\ngot:\n%s", want, got)
	}
	if _, err := Load(path); err != nil {
		t.Fatal(err)
	}
}

func appendConfigApp(t *testing.T, path, name string) {
	t.Helper()

	app, err := model.NewApp(name)
	if err != nil {
		t.Fatal(err)
	}
	if err := AppendApp(path, coreGroup, app); err != nil {
		t.Fatalf("AppendApp() error = %v", err)
	}
}

func assertConfigBytesAndApp(t *testing.T, path, want, appName string) {
	t.Helper()

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("config bytes differ\nwant:\n%s\ngot:\n%s", want, got)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	group, _, ok := FindGroup(cfg, coreGroup)
	if !ok {
		t.Fatalf("group %q not found", coreGroup)
	}
	if len(group.Apps) == 0 || group.Apps[len(group.Apps)-1].App.Name != appName {
		t.Fatalf("last app = %#v, want %q", group.Apps, appName)
	}
}

func writeConfigBytes(t *testing.T, path string, data []byte, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, data, mode); err != nil {
		t.Fatal(err)
	}
}
