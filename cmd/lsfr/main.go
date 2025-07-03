package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	_ "github.com/st3v3nmw/lsfr/challenges"
	"github.com/st3v3nmw/lsfr/internal/config"
	"github.com/st3v3nmw/lsfr/internal/registry"
	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:  "lsfr",
		Usage: "Build complex systems from scratch",
		Commands: []*cli.Command{
			{
				Name:      "new",
				Usage:     "Start a new challenge",
				ArgsUsage: "<challenge>",
				// Action:    newChallenge,
			},
			{
				Name:      "test",
				Usage:     "Test your implementation",
				ArgsUsage: "[challenge] <stage>",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "verbose",
						Usage:   "Show detailed test output",
						Aliases: []string{"v"},
						Value:   false,
					},
				},
				Action: testChallenge,
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

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func testChallenge(ctx context.Context, cmd *cli.Command) error {
	// Check run.sh exists
	if _, err := os.Stat("run.sh"); os.IsNotExist(err) {
		return fmt.Errorf("run.sh not found\nCreate an executable run.sh script that starts your implementation")
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	var challengeKey string
	var stageNum int

	args := cmd.Args().Slice()
	switch cmd.NArg() {
	case 0:
		// Use current stage from config
		challengeKey = cfg.Challenge
		stageNum = cfg.Stage
	case 1:
		// lsfr test <stage>
		challengeKey = cfg.Challenge
		var err error
		stageNum, err = strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid stage number: %s", args[0])
		}
	case 2:
		// lsfr test <challenge> <stage>
		challengeKey = args[0]
		var err error
		stageNum, err = strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("invalid stage number: %s", args[1])
		}
	default:
		return fmt.Errorf("too many arguments\nUsage: lsfr test [challenge] <stage>")
	}

	// Validate
	challenge, err := registry.GetChallenge(challengeKey)
	if err != nil {
		return fmt.Errorf("unknown challenge: %s", challengeKey)
	}

	numStages := challenge.Len()
	if stageNum < 1 || stageNum > numStages {
		return fmt.Errorf("stage %d does not exist for %s\nAvailable stages: 1-%d",
			stageNum, challenge.Name, numStages)
	}

	stage, err := challenge.GetStage(stageNum)
	if err != nil {
		return err
	}

	// Run tests
	fmt.Printf("Running stage %d: %s\n\n", stageNum, stage.Name)

	fn := stage.Fn()
	fn.Run(ctx, cmd.Bool("verbose"))

	return nil
}
