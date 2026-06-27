package config

import "fmt"

func Validate(cfg File) error {
	listNames := make(map[string]struct{}, len(cfg.Lists))
	for _, list := range cfg.Lists {
		if list.Name == "" {
			return fmt.Errorf("list name must not be empty")
		}
		if _, exists := listNames[list.Name]; exists {
			return fmt.Errorf("duplicate list %q", list.Name)
		}
		listNames[list.Name] = struct{}{}
	}

	groupNames := make(map[string]struct{}, len(cfg.Groups))
	for _, group := range cfg.Groups {
		if group.Name == "" {
			return fmt.Errorf("group name must not be empty")
		}
		if _, exists := groupNames[group.Name]; exists {
			return fmt.Errorf("duplicate group %q", group.Name)
		}
		groupNames[group.Name] = struct{}{}

		seenApps := map[string]struct{}{}
		for _, app := range group.Apps {
			if app.App.Name == "" {
				return fmt.Errorf("group %q contains empty app", group.Name)
			}
			if _, exists := seenApps[app.App.Name]; exists {
				return fmt.Errorf("group %q contains duplicate app %q", group.Name, app.App.Name)
			}
			seenApps[app.App.Name] = struct{}{}
		}
	}

	for _, list := range cfg.Lists {
		for _, groupName := range list.Groups {
			if _, exists := groupNames[groupName]; !exists {
				return fmt.Errorf("list %q references unknown group %q", list.Name, groupName)
			}
		}
	}

	return nil
}
