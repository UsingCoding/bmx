package usecase

import (
	"context"
	"fmt"
	"io"

	"github.com/UsingCoding/bmx/internal/config"
	"github.com/UsingCoding/bmx/internal/model"
)

type RemoveInput struct {
	ConfigPath string
	AppName    string
	GroupName  string
	In         io.Reader
	Out        io.Writer
}

func Remove(ctx context.Context, input RemoveInput) error {
	_ = ctx
	app, err := model.NewApp(input.AppName)
	if err != nil {
		return err
	}
	cfg, err := config.Load(input.ConfigPath)
	if err != nil {
		return err
	}
	groupIndex, err := chooseGroup(cfg, input.GroupName, input.In, input.Out)
	if err != nil {
		return err
	}
	group := cfg.Groups[groupIndex]
	found := false
	for _, entry := range group.Apps {
		if entry.App.Name == app.Name {
			found = true
			break
		}
	}
	if !found {
		_, err := fmt.Fprintf(input.Out, "%s does not exist in group %s\n", app.Name, group.Name)
		return err
	}
	if err := config.RemoveApp(input.ConfigPath, group.Name, app); err != nil {
		return err
	}
	_, err = fmt.Fprintf(input.Out, "Removed %s from group %s. Run `bmx converge` next.\n", app.Name, group.Name)
	return err
}
