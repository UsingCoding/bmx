package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	zeroslog "github.com/samber/slog-zerolog"
	"github.com/urfave/cli/v3"
)

const (
	appID = "bmx"
)

var (
	version = "dev"
	commit  = "none"
)

func main() {
	ctx := context.Background()

	ctx = subscribeForKillSignals(ctx)

	err := runApp(ctx, os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func runApp(ctx context.Context, args []string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	c := &cli.Command{
		Name:    appID,
		Version: version,
		// do not use built-in version flag
		HideVersion:           true,
		Usage:                 "Small CLI to sync brew installed packages",
		EnableShellCompletion: true,
		Commands: []*cli.Command{
			versionCMD(),
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
			},
		},
	}

	return c.Run(ctx, args)
}

func subscribeForKillSignals(ctx context.Context) context.Context {
	ctx, cancel := context.WithCancel(ctx)

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		defer cancel()
		select {
		case <-ctx.Done():
			signal.Stop(ch)
		case <-ch:
		}
	}()

	return ctx
}

func logger(cmd *cli.Command) *slog.Logger {
	level := zerolog.InfoLevel
	leveler := slog.LevelInfo
	if cmd.Bool("verbose") {
		level = zerolog.DebugLevel
		leveler = slog.LevelDebug
	}

	w := zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.DateTime,
	}

	zerologL := zerolog.New(w).Level(level)

	opts := zeroslog.Option{
		Logger: &zerologL,
		Level:  leveler,
	}
	handler := opts.NewZerologHandler()
	return slog.New(handler)
}
