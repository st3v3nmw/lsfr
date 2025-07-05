package main

import (
	"context"
	"log"
	"os"

	"github.com/st3v3nmw/lsfr/internal/cli"
	commands "github.com/urfave/cli/v3"
)

func main() {
	cmd := &commands.Command{
		Name:  "lsfr",
		Usage: "Build complex systems from scratch",
		Commands: []*commands.Command{
			{
				Name:      "new",
				Usage:     "Start a new challenge",
				ArgsUsage: "<challenge> [path]",
				Action:    cli.NewChallenge,
			},
			{
				Name:      "test",
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
				Name:   "status",
				Usage:  "Show current progress",
				Action: cli.ShowStatus,
			},
			{
				Name:   "list",
				Usage:  "List available challenges",
				Action: cli.ListChallenges,
			},
		},
	}

	log.SetFlags(0)
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
