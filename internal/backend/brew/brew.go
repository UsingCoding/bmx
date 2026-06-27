package brew

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/UsingCoding/bmx/internal/model"
)

type Manager struct{}

func New() *Manager {
	return &Manager{}
}

func (m *Manager) Install(ctx context.Context, app model.App) error {
	args := brewArgs("install", app)
	cmd := exec.CommandContext(ctx, "brew", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("brew install %s: %w: %s", app.Name, err, string(output))
	}

	return nil
}

func (m *Manager) Uninstall(ctx context.Context, app model.App) error {
	args := brewArgs("uninstall", app)
	cmd := exec.CommandContext(ctx, "brew", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("brew uninstall %s: %w: %s", app.Name, err, string(output))
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
