package model

import (
	"fmt"
	"strings"
)

type App struct {
	Name    string
	Manager string
	Package string
}

func NewApp(name string) (App, error) {
	manager, pkg, err := SplitName(name)
	if err != nil {
		return App{}, err
	}

	return App{
		Name:    name,
		Manager: manager,
		Package: pkg,
	}, nil
}

func SplitName(name string) (manager, pkg string, err error) {
	parts := strings.SplitN(strings.TrimSpace(name), ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid app name %q: expected <manager>:<package>", name)
	}

	return parts[0], parts[1], nil
}

func (a App) Key() string {
	return a.Name
}
