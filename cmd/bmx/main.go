package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	appcli "github.com/UsingCoding/bmx/internal/cli"
)

func main() {
	ctx := subscribeForKillSignals(context.Background())
	if err := runApp(ctx, os.Args); err != nil {
		log.Fatal(err)
	}
}

func runApp(ctx context.Context, args []string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	return appcli.New(version, commit).Run(ctx, args)
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
