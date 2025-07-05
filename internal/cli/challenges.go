package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

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
		Stages: config.Stages{
			Current:   challenge.StageOrder[0],
			Completed: []string{},
		},
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
		firstStageKey := challenge.StageOrder[0]
		fmt.Printf("Implement %s stage, then run 'lsfr test'.\n", firstStageKey)
	} else {
		firstStageKey := challenge.StageOrder[0]
		fmt.Printf("cd %s and implement %s stage, then run 'lsfr test'\n", targetPath, firstStageKey)
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
	var stageKey string

	args := cmd.Args().Slice()
	switch cmd.NArg() {
	case 0:
		// Use current stage from config
		challengeKey = cfg.Challenge
		stageKey = cfg.Stages.Current
	case 1:
		// lsfr test <stage>
		challengeKey = cfg.Challenge
		stageKey = args[0]
	default:
		return fmt.Errorf("too many arguments\nUsage: lsfr test [stage]")
	}

	// Validate
	challenge, err := registry.GetChallenge(challengeKey)
	if err != nil {
		return fmt.Errorf("unknown challenge: %s", challengeKey)
	}

	stage, err := challenge.GetStage(stageKey)
	if err != nil {
		msg := "\nAvailable stages:\n"
		for _, stage := range challenge.StageOrder {
			msg += fmt.Sprintf("- %s\n", stage)
		}
		return fmt.Errorf("%w\n%s", err, msg)
	}

	// Run tests
	suite := stage.Fn()
	suite.Run(ctx, stageKey, stage.Name, stage.Summary)

	return nil
}
