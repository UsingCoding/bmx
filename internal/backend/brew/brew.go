package brew

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"

	"github.com/UsingCoding/bmx/internal/backend"
	"github.com/UsingCoding/bmx/internal/model"
)

type Manager struct {
	in     io.Reader
	out    io.Writer
	errOut io.Writer
}

func New(in io.Reader, out, errOut io.Writer) *Manager {
	return &Manager{in: in, out: out, errOut: errOut}
}

func (m *Manager) Installed(ctx context.Context, app model.App) (bool, error) {
	var kind string
	switch app.Manager {
	case "brew":
		kind = "--formula"
	case "brew-cask":
		kind = "--cask"
	default:
		return false, fmt.Errorf("unsupported Homebrew app manager %q", app.Manager)
	}

	if err := backend.TraceCommand(m.errOut, "brew", "list", kind, app.Package); err != nil {
		return false, fmt.Errorf("trace brew list %s %s: %w", kind, app.Package, err)
	}

	err := exec.CommandContext(ctx, "brew", "list", kind, app.Package).Run()
	if err == nil {
		return true, nil
	}
	if ctx.Err() != nil {
		return false, fmt.Errorf("brew list %s %s: %w", kind, app.Package, ctx.Err())
	}

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return false, fmt.Errorf("brew list %s %s: %w", kind, app.Package, err)
	}

	return false, nil
}

func (m *Manager) Install(ctx context.Context, app model.App) error {
	args := brewArgs("install", app)
	if err := backend.TraceCommand(m.errOut, "brew", args...); err != nil {
		return fmt.Errorf("trace brew install %s: %w", app.Name, err)
	}

	cmd := exec.CommandContext(ctx, "brew", args...)
	cmd.Stdin = m.in
	cmd.Stdout = m.out
	cmd.Stderr = m.errOut
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("brew install %s: %w", app.Name, err)
	}

	return nil
}

func (m *Manager) Uninstall(ctx context.Context, app model.App) error {
	args := brewArgs("uninstall", app)
	if err := backend.TraceCommand(m.errOut, "brew", args...); err != nil {
		return fmt.Errorf("trace brew uninstall %s: %w", app.Name, err)
	}
	cmd := exec.CommandContext(ctx, "brew", args...)

	cmd.Stdin = m.in
	cmd.Stdout = m.out
	cmd.Stderr = m.errOut
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("brew uninstall %s: %w", app.Name, err)
	}

	return nil
}

func brewArgs(action string, app model.App) []string {
	args := []string{action}
	if app.Manager == "brew-cask" {
		args = append(args, "--cask")
	}
	args = append(args, app.Package)
	return args
}
