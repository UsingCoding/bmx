package usecase

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/manifoldco/promptui"

	"github.com/UsingCoding/bmx/internal/config"
)

type ListInput struct {
	ConfigPath string
	ListName   string
	Out        io.Writer
	UseStyles  bool
}

func List(ctx context.Context, input ListInput) error {
	_ = ctx
	cfg, err := config.Load(input.ConfigPath)
	if err != nil {
		return err
	}

	if input.ListName != "" {
		apps, err := config.ResolveList(cfg, input.ListName)
		if err != nil {
			return err
		}
		fmt.Fprintf(input.Out, "%s %s\n", listHeading(input.UseStyles, "List"), listName(input.UseStyles, input.ListName))
		for _, app := range apps {
			fmt.Fprintf(input.Out, "- %s\n", appName(input.UseStyles, app.Name))
		}
		return nil
	}

	fmt.Fprintln(input.Out, listHeading(input.UseStyles, "Lists:"))
	for _, list := range cfg.Lists {
		fmt.Fprintf(input.Out, "- %s: %s\n", listName(input.UseStyles, list.Name), groupNames(input.UseStyles, list.Groups))
	}

	fmt.Fprintln(input.Out, groupHeading(input.UseStyles, "Groups:"))
	for _, group := range cfg.Groups {
		fmt.Fprintf(input.Out, "- %s\n", listName(input.UseStyles, group.Name))
		for _, app := range group.Apps {
			fmt.Fprintf(input.Out, "  - %s\n", appName(input.UseStyles, app.App.Name))
		}
	}

	return nil
}

func listHeading(useStyles bool, value string) string {
	if !useStyles {
		return value
	}
	return promptui.Styler(promptui.FGCyan)(value)
}

func groupHeading(useStyles bool, value string) string {
	if !useStyles {
		return value
	}
	return promptui.Styler(promptui.FGMagenta)(value)
}

func listName(useStyles bool, value string) string {
	if !useStyles {
		return value
	}
	return promptui.Styler(promptui.FGItalic)(value)
}

func appName(useStyles bool, value string) string {
	if !useStyles {
		return value
	}
	return promptui.Styler(promptui.FGBold)(value)
}

func groupNames(useStyles bool, groups []string) string {
	if !useStyles {
		return strings.Join(groups, ", ")
	}

	styled := make([]string, len(groups))
	for idx, group := range groups {
		styled[idx] = listName(true, group)
	}
	return strings.Join(styled, ", ")
}
