package usecase

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/UsingCoding/bmx/internal/config"
)

type ListInput struct {
	ConfigPath string
	ListName   string
	Out        io.Writer
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
		fmt.Fprintf(input.Out, "List %s\n", input.ListName)
		for _, app := range apps {
			fmt.Fprintf(input.Out, "- %s\n", app.Name)
		}
		return nil
	}

	fmt.Fprintln(input.Out, "Lists:")
	for _, list := range cfg.Lists {
		fmt.Fprintf(input.Out, "- %s: %s\n", list.Name, strings.Join(list.Groups, ", "))
	}

	fmt.Fprintln(input.Out, "Groups:")
	for _, group := range cfg.Groups {
		fmt.Fprintf(input.Out, "- %s\n", group.Name)
		for _, app := range group.Apps {
			fmt.Fprintf(input.Out, "  - %s\n", app.App.Name)
		}
	}

	return nil
}
