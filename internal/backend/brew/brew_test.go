package brew

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/UsingCoding/bmx/internal/model"
)

const (
	formulaManager  = "brew"
	formulaName     = "brew:jq"
	caskManager     = "brew-cask"
	uninstallAction = "uninstall"
)

func TestManagerInstalled(t *testing.T) {
	app := model.App{Name: formulaName, Manager: formulaManager, Package: "jq"}
	caskApp := model.App{Name: "brew-cask:jq", Manager: "brew-cask", Package: "jq"}

	t.Run("formula installed", func(t *testing.T) {
		logPath := setupFakeBrew(t)
		t.Setenv("BMX_FORMULA_EXIT", "0")

		installed, err := New(nil, io.Discard, io.Discard).Installed(context.Background(), app)
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

		installed, err := New(nil, io.Discard, io.Discard).Installed(context.Background(), caskApp)
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

		installed, err := New(nil, io.Discard, io.Discard).Installed(context.Background(), app)
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

		installed, err := New(nil, io.Discard, io.Discard).Installed(context.Background(), app)
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

		installed, err := New(nil, io.Discard, io.Discard).Installed(ctx, app)
		if err == nil || !errors.Is(err, context.Canceled) {
			t.Fatalf("Installed() error = %v, want wrapping context.Canceled", err)
		}
		if installed {
			t.Fatal("Installed() installed = true, want false")
		}
	})
}

func TestManagerTracesCommands(t *testing.T) {
	setupFakeBrew(t)
	t.Setenv("BMX_FORMULA_EXIT", "0")
	t.Setenv("BMX_CASK_EXIT", "0")
	var trace bytes.Buffer
	manager := New(nil, io.Discard, &trace)
	formula := model.App{Name: formulaName, Manager: formulaManager, Package: "jq"}
	cask := model.App{Name: "brew-cask:jq", Manager: "brew-cask", Package: "jq"}

	for _, app := range []model.App{formula, cask} {
		if _, err := manager.Installed(context.Background(), app); err != nil {
			t.Fatalf("Installed(%s) error = %v", app.Name, err)
		}
		if err := manager.Install(context.Background(), app); err != nil {
			t.Fatalf("Install(%s) error = %v", app.Name, err)
		}
		if err := manager.Uninstall(context.Background(), app); err != nil {
			t.Fatalf("Uninstall(%s) error = %v", app.Name, err)
		}
	}

	want := "$ brew list --formula jq\n$ brew install jq\n$ brew uninstall jq\n$ brew list --cask jq\n$ brew install --cask jq\n$ brew uninstall --cask jq\n"
	if got := trace.String(); got != want {
		t.Fatalf("trace = %q, want %q", got, want)
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("trace write failed")
}

func TestManagerDoesNotRunWhenTraceFails(t *testing.T) {
	logPath := setupFakeBrew(t)
	app := model.App{Name: formulaName, Manager: formulaManager, Package: "jq"}

	err := New(nil, io.Discard, failingWriter{}).Install(context.Background(), app)
	if err == nil || !strings.Contains(err.Error(), "trace write failed") {
		t.Fatalf("Install() error = %v, want trace write failure", err)
	}
	if _, err := os.Stat(logPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("brew was invoked or log could not be checked: %v", err)
	}
}

func TestManagerStreamsMutationIO(t *testing.T) {
	for _, action := range []string{"install", uninstallAction} {
		for _, kind := range []string{formulaManager, caskManager} {
			t.Run(action+"/"+kind, func(t *testing.T) {
				logPath := setupFakeBrew(t)
				t.Setenv("BMX_MUTATION_INTERACTIVE", "1")
				inR, inW := mutationPipe(t)
				outR, outW := mutationPipe(t)
				errR, errW := mutationPipe(t)
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				t.Cleanup(cancel)
				manager := New(inR, outW, errW)
				app := model.App{Name: kind + ":jq", Manager: kind, Package: "jq"}
				run := manager.Install
				if action == uninstallAction {
					run = manager.Uninstall
				}
				done := make(chan error, 1)
				go func() { done <- run(ctx, app) }()
				readMutationOutput(t, outR, "brew-progress\n")
				args := action + " jq"
				if kind == caskManager {
					args = action + " --cask jq"
				}
				wantErr := "$ brew " + args + "\nbrew-diagnostic\n"
				readMutationOutput(t, errR, wantErr)
				if _, err := inW.WriteString("continue\n"); err != nil {
					t.Fatal(err)
				}
				select {
				case err := <-done:
					if err != nil {
						t.Fatalf("mutation: %v", err)
					}
				case <-ctx.Done():
					t.Fatal("mutation did not complete after input")
				}
				if got := readBrewLog(t, logPath); got != args {
					t.Fatalf("arguments = %q, want %q", got, args)
				}
			})
		}
	}
}

func TestManagerMutationFailurePreservesDiagnostics(t *testing.T) {
	for _, action := range []string{"install", uninstallAction} {
		t.Run(action, func(t *testing.T) {
			setupFakeBrew(t)
			t.Setenv("BMX_MUTATION_INTERACTIVE", "1")
			t.Setenv("BMX_MUTATION_EXIT", "7")
			var stdout, stderr bytes.Buffer
			manager := New(strings.NewReader("continue\n"), &stdout, &stderr)
			run := manager.Install
			if action == uninstallAction {
				run = manager.Uninstall
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			t.Cleanup(cancel)
			err := run(ctx, model.App{Name: formulaName, Manager: formulaManager, Package: "jq"})
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) || exitErr.ExitCode() != 7 {
				t.Fatalf("error = %v, want wrapped exit status 7", err)
			}
			if !strings.Contains(err.Error(), "brew "+action+" "+formulaName) ||
				strings.Contains(err.Error(), "brew-diagnostic") {
				t.Fatalf("unexpected error context: %v", err)
			}
			if stdout.String() != "brew-progress\n" {
				t.Fatalf("stdout = %q", stdout.String())
			}
			if want := "$ brew " + action + " jq\nbrew-diagnostic\n"; stderr.String() != want {
				t.Fatalf("stderr = %q, want %q", stderr.String(), want)
			}
		})
	}
}

func readMutationOutput(t *testing.T, reader io.Reader, want string) {
	t.Helper()
	got := make([]byte, len(want))
	if _, err := io.ReadFull(reader, got); err != nil {
		t.Fatalf("output before input: %v", err)
	}
	if string(got) != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func mutationPipe(t *testing.T) (reader, writer *os.File) {
	t.Helper()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = reader.Close()
		_ = writer.Close()
	})
	deadline := time.Now().Add(5 * time.Second)
	if err := reader.SetReadDeadline(deadline); err != nil {
		t.Fatal(err)
	}
	if err := writer.SetWriteDeadline(deadline); err != nil {
		t.Fatal(err)
	}
	return reader, writer
}

func setupFakeBrew(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	logPath := filepath.Join(dir, "brew.log")
	scriptPath := filepath.Join(dir, "brew")
	script := `#!/bin/sh
echo "$*" >> "$BMX_BREW_LOG"
case "$1" in
  list)
    case "$2" in
      --formula) exit "${BMX_FORMULA_EXIT:-1}" ;;
      --cask) exit "${BMX_CASK_EXIT:-1}" ;;
    esac
    ;;
  install|uninstall)
    if test "${BMX_MUTATION_INTERACTIVE:-0}" = 1; then
      printf 'brew-progress\n'
      printf 'brew-diagnostic\n' >&2
      IFS= read -r answer || exit 9
      test "$answer" = continue || exit 8
    fi
    exit "${BMX_MUTATION_EXIT:-0}"
    ;;
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
