package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

func Load(path string) (File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return File{}, fmt.Errorf("read config %s: %w", path, err)
	}

	var cfg File
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return File{}, fmt.Errorf("decode config %s: %w", path, err)
	}

	if err := Validate(cfg); err != nil {
		return File{}, err
	}

	return cfg, nil
}
