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
				Usage:     "Test your implementation",
				ArgsUsage: "[stage]",
				Action:    cli.TestChallenge,
			},
			{
				Name:  "next",
				Usage: "Advance to the next stage",
				// Action: nextStage,
			},
			{
				Name:  "list",
				Usage: "Show available challenges",
				// Action: listChallenges,
			},
			{
				Name:      "info",
				Usage:     "Show challenge details",
				ArgsUsage: "<challenge>",
				// Action:    showInfo,
			},
			{
				Name:  "status",
				Usage: "Show current progress",
				// Action: showStatus,
			},
		},
	}

	log.SetFlags(0)
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
