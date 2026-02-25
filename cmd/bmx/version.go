package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/urfave/cli/v3"
)

func versionCMD() *cli.Command {
	return &cli.Command{
		Name:    "version",
		Usage:   "Show bmx version",
		Aliases: []string{"v"},
		Action:  executeVersion,
	}
}

func executeVersion(context.Context, *cli.Command) error {
	v := struct {
		Version string `json:"version"`
		Commit  string `json:"commit"`
	}{
		Version: version,
		Commit:  commit,
	}
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	fmt.Println(string(data))

	return nil
}
