package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/UsingCoding/bmx/internal/backend"
	brewbackend "github.com/UsingCoding/bmx/internal/backend/brew"
	"github.com/UsingCoding/bmx/internal/paths"
	"github.com/UsingCoding/bmx/internal/usecase"
)

func New(version, commit string) *cli.Command {
	return &cli.Command{
		Name:                  "bmx",
		Version:               version,
		HideVersion:           true,
		Usage:                 "Declarative local package manager helper",
		EnableShellCompletion: true,
		Commands: []*cli.Command{
			versionCommand(version, commit),
			initCommand(),
			convergeCommand(),
			listCommand(),
			addCommand(),
			removeCommand(),
		},
	}
}

func commonPathFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "config", Usage: "Path to bmxfile.toml"},
		&cli.StringFlag{Name: "state", Usage: "Path to bmxfile.state.toml"},
	}
}

func configFlag() cli.Flag {
	return &cli.StringFlag{Name: "config", Usage: "Path to bmxfile.toml"}
}

func initCommand() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "Create a starter bmxfile.toml",
		Flags: []cli.Flag{configFlag()},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			resolved, err := paths.Resolve(paths.Options{ConfigPath: cmd.String("config")})
			if err != nil {
				return err
			}

			return usecase.Init(ctx, usecase.InitInput{
				ConfigPath: resolved.ConfigPath,
				Out:        os.Stdout,
			})
		},
	}
}

func convergeCommand() *cli.Command {
	flags := append(commonPathFlags(), &cli.StringFlag{Name: "list", Usage: "List profile to converge"})
	return &cli.Command{
		Name:  "converge",
		Usage: "Converge installed apps to desired state",
		Flags: flags,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			resolved, err := paths.Resolve(paths.Options{
				ConfigPath: cmd.String("config"),
				StatePath:  cmd.String("state"),
			})
			if err != nil {
				return err
			}

			return usecase.Converge(ctx, usecase.ConvergeInput{
				PlanInput: usecase.PlanInput{
					ConfigPath: resolved.ConfigPath,
					StatePath:  resolved.StatePath,
					ListName:   cmd.String("list"),
				},
				Managers: defaultRegistry(),
				In:       os.Stdin,
				Out:      os.Stdout,
			})
		},
	}
}

func listCommand() *cli.Command {
	flags := append(commonPathFlags(), &cli.StringFlag{Name: "list", Usage: "List profile to show"})
	return &cli.Command{
		Name:      "ls",
		Usage:     "List config entries",
		ArgsUsage: "[list-name]",
		Flags:     flags,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			resolved, err := paths.Resolve(paths.Options{ConfigPath: cmd.String("config"), StatePath: cmd.String("state")})
			if err != nil {
				return err
			}

			listName := cmd.String("list")
			if listName == "" && cmd.Args().Len() > 0 {
				listName = cmd.Args().First()
			}

			return usecase.List(ctx, usecase.ListInput{
				ConfigPath: resolved.ConfigPath,
				ListName:   listName,
				Out:        os.Stdout,
			})
		},
	}
}

func addCommand() *cli.Command {
	flags := append(commonPathFlags(), &cli.StringFlag{Name: "group", Usage: "Group to receive the app"})
	return &cli.Command{
		Name:      "add",
		Usage:     "Add package to desired config",
		ArgsUsage: "<manager:package>",
		Flags:     flags,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() != 1 {
				return fmt.Errorf("expected exactly one package argument")
			}

			resolved, err := paths.Resolve(paths.Options{ConfigPath: cmd.String("config"), StatePath: cmd.String("state")})
			if err != nil {
				return err
			}

			return usecase.Add(ctx, usecase.AddInput{
				ConfigPath: resolved.ConfigPath,
				AppName:    cmd.Args().First(),
				GroupName:  cmd.String("group"),
				In:         os.Stdin,
				Out:        os.Stdout,
			})
		},
	}
}

func removeCommand() *cli.Command {
	flags := append(commonPathFlags(), &cli.StringFlag{Name: "group", Usage: "Group containing the app"})
	return &cli.Command{
		Name:      "rm",
		Aliases:   []string{"remove"},
		Usage:     "Remove package from desired config",
		ArgsUsage: "<manager:package>",
		Flags:     flags,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() != 1 {
				return fmt.Errorf("expected exactly one package argument")
			}

			resolved, err := paths.Resolve(paths.Options{ConfigPath: cmd.String("config"), StatePath: cmd.String("state")})
			if err != nil {
				return err
			}

			return usecase.Remove(ctx, usecase.RemoveInput{
				ConfigPath: resolved.ConfigPath,
				AppName:    cmd.Args().First(),
				GroupName:  cmd.String("group"),
				In:         os.Stdin,
				Out:        os.Stdout,
			})
		},
	}
}

func defaultRegistry() backend.Registry {
	brewManager := brewbackend.New()
	return backend.Registry{
		"brew":      brewManager,
		"brew-cask": brewManager,
	}
}

func versionCommand(version, commit string) *cli.Command {
	return &cli.Command{
		Name:    "version",
		Usage:   "Show bmx version",
		Aliases: []string{"v"},
		Action: func(context.Context, *cli.Command) error {
			payload, err := json.Marshal(struct {
				Version string `json:"version"`
				Commit  string `json:"commit"`
			}{Version: version, Commit: commit})
			if err != nil {
				return err
			}
			fmt.Println(string(payload))
			return nil
		},
	}
}
