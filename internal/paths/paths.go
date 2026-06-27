package paths

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	EnvConfigPath = "BMX_CONFIG"
	EnvStatePath  = "BMX_STATE"

	defaultConfigFile = "bmxfile.toml"
	defaultStateFile  = "bmxfile.state.toml"
)

type Options struct {
	ConfigPath string
	StatePath  string
}

type Resolved struct {
	ConfigPath string
	StatePath  string
}

func Resolve(opts Options) (Resolved, error) {
	configPath, err := resolveOne(opts.ConfigPath, EnvConfigPath, defaultConfigFile)
	if err != nil {
		return Resolved{}, fmt.Errorf("resolve config path: %w", err)
	}

	statePath, err := resolveOne(opts.StatePath, EnvStatePath, defaultStateFile)
	if err != nil {
		return Resolved{}, fmt.Errorf("resolve state path: %w", err)
	}

	return Resolved{ConfigPath: configPath, StatePath: statePath}, nil
}

func resolveOne(explicit, envName, fallback string) (string, error) {
	if explicit != "" {
		return filepath.Abs(explicit)
	}

	if fromEnv := os.Getenv(envName); fromEnv != "" {
		return filepath.Abs(fromEnv)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, fallback), nil
}
