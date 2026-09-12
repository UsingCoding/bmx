package usecase

import (
	"context"
	"fmt"
	"io"

	"github.com/UsingCoding/bmx/internal/config"
	"github.com/UsingCoding/bmx/internal/model"
)

type AddInput struct {
	ConfigPath string
	AppName    string
	GroupName  string
	In         io.Reader
	Out        io.Writer
}

func Add(ctx context.Context, input AddInput) error {
	_ = ctx
	app, err := model.NewApp(input.AppName)
	if err != nil {
		return err
	}

	cfg, err := config.Load(input.ConfigPath)
	if err != nil {
		return err
	}

	for _, group := range cfg.Groups {
		for _, entry := range group.Apps {
			if entry.App.Name == app.Name {
				_, err := fmt.Fprintf(input.Out, "%s already exists in group %s\n", app.Name, group.Name)
				return err
			}
		}
	}

	groupIndex, err := chooseGroup(cfg, input.GroupName, input.In, input.Out)
	if err != nil {
		return err
	}

	if err := config.AppendApp(input.ConfigPath, cfg.Groups[groupIndex].Name, app); err != nil {
		return err
	}

	_, err = fmt.Fprintf(input.Out, "Added %s to group %s. Run `bmx converge` next.\n", app.Name, cfg.Groups[groupIndex].Name)
	return err
}

func chooseGroup(cfg config.File, explicit string, in io.Reader, out io.Writer) (int, error) {
	if explicit != "" {
		_, idx, ok := config.FindGroup(cfg, explicit)
		if !ok {
			return -1, fmt.Errorf("unknown group %q", explicit)
		}
		return idx, nil
	}

	if len(cfg.Groups) == 1 {
		return 0, nil
	}

	items := make([]string, len(cfg.Groups))
	for idx, group := range cfg.Groups {
		items[idx] = group.Name
	}

	return selectPrompt("Select group", items, in, out)
}
