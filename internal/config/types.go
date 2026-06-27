package config

import (
	"fmt"

	"github.com/UsingCoding/bmx/internal/model"
)

type File struct {
	Lists  []List  `toml:"lists"`
	Groups []Group `toml:"groups"`
}

type List struct {
	Name   string   `toml:"name"`
	Groups []string `toml:"groups"`
}

type Group struct {
	Name string     `toml:"name"`
	Apps []AppEntry `toml:"apps"`
}

type AppEntry struct {
	App model.App
}

func (a *AppEntry) UnmarshalTOML(value any) error {
	switch typed := value.(type) {
	case string:
		app, err := model.NewApp(typed)
		if err != nil {
			return err
		}
		a.App = app
		return nil
	case map[string]any:
		nameValue, ok := typed["name"]
		if !ok {
			return fmt.Errorf("app object missing name")
		}

		name, ok := nameValue.(string)
		if !ok {
			return fmt.Errorf("app object name must be a string")
		}

		if _, hasCask := typed["cask"]; hasCask {
			return fmt.Errorf("app object cask is no longer supported; use brew-cask:<package>")
		}

		app, err := model.NewApp(name)
		if err != nil {
			return err
		}
		a.App = app
		return nil
	default:
		return fmt.Errorf("unsupported app entry type %T", value)
	}
}
