package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

func Write(path string, cfg File) error {
	if err := Validate(cfg); err != nil {
		return err
	}

	var buf bytes.Buffer
	for idx, list := range cfg.Lists {
		if idx > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString("[[lists]]\n")
		buf.WriteString("name = ")
		buf.WriteString(strconv.Quote(list.Name))
		buf.WriteByte('\n')
		buf.WriteString("groups = [")
		for i, group := range list.Groups {
			if i > 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(strconv.Quote(group))
		}
		buf.WriteString("]\n")
	}

	if len(cfg.Lists) > 0 && len(cfg.Groups) > 0 {
		buf.WriteByte('\n')
	}

	for idx, group := range cfg.Groups {
		if idx > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString("[[groups]]\n")
		buf.WriteString("name = ")
		buf.WriteString(strconv.Quote(group.Name))
		buf.WriteByte('\n')
		buf.WriteString("apps = [\n")
		for _, app := range group.Apps {
			buf.WriteString("  ")
			buf.WriteString(strconv.Quote(app.App.Name))
			buf.WriteString(",\n")
		}
		buf.WriteString("]\n")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		return fmt.Errorf("write config %s: %w", path, err)
	}

	return nil
}
