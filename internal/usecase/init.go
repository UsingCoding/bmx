package usecase

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type InitInput struct {
	ConfigPath string
	Out        io.Writer
}

const defaultConfigTemplate = `# BMX desired-state config.
#
# This file declares what packages should be installed.
# Run "bmx converge" to apply changes after editing it.
#
# Model:
#   - A list is an install profile, for example "linux" or "macos".
#   - A list references one or more groups.
#   - A group contains app entries.
#   - App names use package-manager prefixes, for example "brew:docker" or "brew-cask:gimp".
#
# Example lists:
[[lists]]
name = "macos"
groups = [
  "core.unix",
  "core.macos",
  "core.macos.gui",
]

[[lists]]
name = "linux"
groups = ["core.unix"]

# Example package groups:
[[groups]]
name = "core.unix"
apps = [
  "brew:age",
  "brew:lazygit",
]

[[groups]]
name = "core.macos"
apps = [
  "brew:mactop",
]

[[groups]]
name = "core.macos.gui"
apps = [
  "brew-cask:gimp",
]
`

func Init(ctx context.Context, input InitInput) error {
	_ = ctx

	if _, err := os.Stat(input.ConfigPath); err == nil {
		return fmt.Errorf("config already exists: %s", input.ConfigPath)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat config %s: %w", input.ConfigPath, err)
	}

	if err := os.MkdirAll(filepath.Dir(input.ConfigPath), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	if err := os.WriteFile(input.ConfigPath, []byte(defaultConfigTemplate), 0o600); err != nil {
		return fmt.Errorf("write config %s: %w", input.ConfigPath, err)
	}

	if _, err := fmt.Fprintf(input.Out, "Created %s\nNext: run `bmx add brew:age` or edit the file, then run `bmx converge`.\n", input.ConfigPath); err != nil {
		return err
	}

	return nil
}
