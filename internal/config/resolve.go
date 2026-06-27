package config

import (
	"fmt"

	"github.com/UsingCoding/bmx/internal/model"
)

func ResolveList(cfg File, listName string) ([]model.App, error) {
	list, ok := FindList(cfg, listName)
	if !ok {
		return nil, fmt.Errorf("unknown list %q", listName)
	}

	groupsByName := make(map[string]Group, len(cfg.Groups))
	for _, group := range cfg.Groups {
		groupsByName[group.Name] = group
	}

	appsByName := make(map[string]model.App)
	ordered := make([]model.App, 0)
	for _, groupName := range list.Groups {
		group := groupsByName[groupName]
		for _, entry := range group.Apps {
			if _, ok := appsByName[entry.App.Name]; ok {
				continue
			}
			appsByName[entry.App.Name] = entry.App
			ordered = append(ordered, entry.App)
		}
	}

	return ordered, nil
}

func FindList(cfg File, name string) (List, bool) {
	for _, list := range cfg.Lists {
		if list.Name == name {
			return list, true
		}
	}

	return List{}, false
}

func FindGroup(cfg File, name string) (Group, int, bool) {
	for idx, group := range cfg.Groups {
		if group.Name == name {
			return group, idx, true
		}
	}

	return Group{}, -1, false
}
