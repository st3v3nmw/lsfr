package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	_ "github.com/st3v3nmw/lsfr/challenges"
	"github.com/st3v3nmw/lsfr/internal/config"
	"github.com/st3v3nmw/lsfr/internal/registry"
	commands "github.com/urfave/cli/v3"
)

func NewChallenge(ctx context.Context, cmd *commands.Command) error {
	args := cmd.Args().Slice()
	if len(args) == 0 {
		return fmt.Errorf("challenge name is required\nUsage: lsfr new <challenge> [path]")
	}

	challengeKey := args[0]
	var targetPath string

	if len(args) > 1 {
		targetPath = args[1]
	} else {
		targetPath = "."
	}

	// Validate that the challenge exists
	challenge, err := registry.GetChallenge(challengeKey)
	if err != nil {
		return fmt.Errorf("unknown challenge: %s", challengeKey)
	}

	// Create directory if specified
	if targetPath != "." {
		if err := os.MkdirAll(targetPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", targetPath, err)
		}
	}

	// Create run.sh
	runShPath := filepath.Join(targetPath, "run.sh")
	runShContent := `#!/bin/bash

# This script runs your implementation
# lsfr will execute this script to start your program
# "$@" passes any command-line arguments from lsfr to your program

echo "Replace this line with the command that runs your implementation"
# Examples:
#   go run ./cmd/server "$@"
#   python main.py "$@"
#   ./my-program "$@"
`
	if err := os.WriteFile(runShPath, []byte(runShContent), 0755); err != nil {
		return fmt.Errorf("failed to create run.sh: %w", err)
	}

	// Create README.md
	readmePath := filepath.Join(targetPath, "README.md")
	if err := os.WriteFile(readmePath, []byte(challenge.README), 0644); err != nil {
		return fmt.Errorf("failed to create README.md: %w", err)
	}

	// Create lsfr.yaml
	cfg := &config.Config{
		Challenge: challengeKey,
		Stage:     1,
	}
	configPath := filepath.Join(targetPath, "lsfr.yaml")
	if err := config.SaveTo(cfg, configPath); err != nil {
		return fmt.Errorf("failed to create lsfr.yaml: %w", err)
	}

	// Output success message
	if targetPath == "." {
		fmt.Println("Created challenge in current directory")
	} else {
		fmt.Printf("Created challenge in directory: %s\n", targetPath)
	}

	fmt.Println("  run.sh       - Your implementation entry point")
	fmt.Println("  README.md    - Challenge overview and requirements")
	fmt.Println("  lsfr.yaml    - Tracks your progress")
	fmt.Println()

	if targetPath == "." {
		if challenge.Len() > 0 {
			stage, _ := challenge.GetStage(1)
			fmt.Printf("Implement %s stage, then run 'lsfr test'.\n", stage.Name)
		} else {
			fmt.Println("Run 'lsfr test' to get started.")
		}
	} else {
		if challenge.Len() > 0 {
			stage, _ := challenge.GetStage(1)
			fmt.Printf("cd %s and implement %s stage, then run 'lsfr test'\n", targetPath, stage.Name)
		} else {
			fmt.Printf("cd %s and run 'lsfr test' to get started.\n", targetPath)
		}
	}

	return nil
}

func TestChallenge(ctx context.Context, cmd *commands.Command) error {
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
