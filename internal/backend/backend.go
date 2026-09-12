package backend

import (
	"context"
	"fmt"

	"github.com/UsingCoding/bmx/internal/model"
)

type Manager interface {
	Installed(ctx context.Context, app model.App) (bool, error)
	Install(ctx context.Context, app model.App) error
	Uninstall(ctx context.Context, app model.App) error
}

type Registry map[string]Manager

func (r Registry) For(app model.App) (Manager, error) {
	manager, ok := r[app.Manager]
	if !ok {
		return nil, fmt.Errorf("unsupported package manager %q", app.Manager)
	}

	return manager, nil
}
