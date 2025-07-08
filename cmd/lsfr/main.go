package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/st3v3nmw/lsfr/internal/cli"
	commands "github.com/urfave/cli/v3"
)

func main() {
	log.SetFlags(0)

	cmd := &commands.Command{
		Name:  "lsfr",
		Usage: "Build complex systems from scratch",
		Commands: []*commands.Command{
			{
				Name:      "new",
				Aliases:   []string{"n"},
				Usage:     "Start a new challenge",
				ArgsUsage: "<challenge> [path]",
				Action:    cli.NewChallenge,
			},
			{
				Name:      "test",
				Aliases:   []string{"t"},
				Usage:     "Test current or specific stage",
				ArgsUsage: "[stage]",
				Action:    cli.TestStage,
			},
			{
				Name:   "next",
				Usage:  "Advance to the next stage",
				Action: cli.NextStage,
			},
			{
				Name:    "status",
				Aliases: []string{"s"},
				Usage:   "Show current progress",
				Action:  cli.ShowStatus,
			},
			{
				Name:    "list",
				Aliases: []string{"l", "ls"},
				Usage:   "List available challenges",
				Action:  cli.ListChallenges,
			},
		},
	}

	// Graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		cancel()
	}()

	// Run
	if err := cmd.Run(ctx, os.Args); err != nil {
		if ctx.Err() == context.Canceled {
			os.Exit(0)
		}

		log.Fatal(err)
	}
}
