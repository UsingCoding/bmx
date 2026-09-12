package backend

import (
	"context"
	"fmt"
	"io"
	"strings"

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

func TraceCommand(out io.Writer, name string, args ...string) error {
	var display strings.Builder
	display.WriteString("$ ")
	display.WriteString(traceArgument(name))
	for _, arg := range args {
		display.WriteByte(' ')
		display.WriteString(traceArgument(arg))
	}
	display.WriteByte('\n')

	_, err := io.WriteString(out, display.String())
	return err
}

func traceArgument(arg string) string {
	if arg != "" && strings.IndexFunc(arg, func(r rune) bool {
		return (r < 'a' || r > 'z') &&
			(r < 'A' || r > 'Z') &&
			(r < '0' || r > '9') &&
			!strings.ContainsRune("_@%+=:,./-", r)
	}) == -1 {
		return arg
	}

	return "'" + strings.ReplaceAll(arg, "'", `'"'"'`) + "'"
}
