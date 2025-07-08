package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	_ "github.com/st3v3nmw/lsfr/challenges"
	"github.com/st3v3nmw/lsfr/internal/config"
	"github.com/st3v3nmw/lsfr/internal/registry"
	commands "github.com/urfave/cli/v3"
)

var (
	green     = color.New(color.FgGreen).SprintFunc()
	red       = color.New(color.FgRed).SprintFunc()
	yellow    = color.New(color.FgYellow).SprintFunc()
	bold      = color.New(color.Bold).SprintFunc()
	checkMark = green("✓")
	crossMark = red("✗")
)

// createChallengeFiles creates the initial project files for a new challenge
func createChallengeFiles(challenge *registry.Challenge, targetPath string) error {
	// run.sh
	scriptPath := filepath.Join(targetPath, "run.sh")
	scriptTemplate := `#!/bin/bash

# This script builds and runs your implementation.
# lsfr will execute this script to start your program.
# "$@" passes any command-line arguments from lsfr to your program.

echo "Replace this line with the command that runs your implementation."
# Examples:
#   go run ./cmd/server "$@"
#   python main.py "$@"
#   ./my-program "$@"
`

	if err := os.WriteFile(scriptPath, []byte(scriptTemplate), 0755); err != nil {
		return fmt.Errorf("Failed to create run.sh: %w", err)
	}

	// README.md
	readmePath := filepath.Join(targetPath, "README.md")
	if err := os.WriteFile(readmePath, []byte(challenge.README()), 0644); err != nil {
		return fmt.Errorf("Failed to create README.md: %w", err)
	}

	// lsfr.yaml
	cfg := &config.Config{
		Challenge: challenge.Key,
		Stages: config.Stages{
			Current:   challenge.StageOrder[0],
			Completed: []string{},
		},
	}
	configPath := filepath.Join(targetPath, "lsfr.yaml")
	if err := config.SaveTo(cfg, configPath); err != nil {
		return fmt.Errorf("Failed to create lsfr.yaml: %w", err)
	}

	return nil
}

// NewChallenge creates a new challenge in the specified directory
func NewChallenge(ctx context.Context, cmd *commands.Command) error {
	// Get Challenge
	args := cmd.Args().Slice()
	if len(args) == 0 {
		return fmt.Errorf("Challenge name is required.\nUsage: lsfr new <challenge> [path]")
	}

	challengeKey := args[0]
	challenge, err := registry.GetChallenge(challengeKey)
	if err != nil {
		return err
	}

	// Create Directory
	var targetPath string
	if len(args) > 1 {
		targetPath = args[1]
		if err := os.MkdirAll(targetPath, 0755); err != nil {
			return fmt.Errorf("Failed to create directory %s: %w", targetPath, err)
		}
	} else {
		targetPath = "."
	}

	if err := createChallengeFiles(challenge, targetPath); err != nil {
		return err
	}

	if targetPath == "." {
		fmt.Println("Created challenge in current directory.")
	} else {
		fmt.Printf("Created challenge in directory: ./%s\n", targetPath)
	}

	fmt.Println("  run.sh       - Your implementation entry point")
	fmt.Println("  README.md    - Challenge overview and requirements")
	fmt.Printf("  lsfr.yaml    - Tracks your progress\n\n")

	firstStageKey := challenge.StageOrder[0]
	if targetPath == "." {
		fmt.Printf("Implement %s stage, then run 'lsfr test'.\n", firstStageKey)
	} else {
		fmt.Printf("cd %s and implement %s stage, then run 'lsfr test'.\n", targetPath, firstStageKey)
	}

	return nil
}

func isStageCompleted(stageKey string, completedStages []string) bool {
	for _, completed := range completedStages {
		if completed == stageKey {
			return true
		}
	}

	return false
}

// validateEnvironment checks if run.sh exists and loads the config
func validateEnvironment() (*config.Config, error) {
	if _, err := os.Stat("run.sh"); os.IsNotExist(err) {
		return nil, fmt.Errorf("run.sh not found\nCreate an executable run.sh script that starts your implementation.")
	}

	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

// runStageTests runs tests for a specific stage and returns success/failure
func runStageTests(ctx context.Context, challengeKey, stageKey string) (bool, error) {
	challenge, err := registry.GetChallenge(challengeKey)
	if err != nil {
		return false, err
	}

	stage, err := challenge.GetStage(stageKey)
	if err != nil {
		msg := "\nAvailable stages:\n"
		for _, stage := range challenge.StageOrder {
			msg += fmt.Sprintf("- %s\n", stage)
		}

		return false, fmt.Errorf("%w\n%s", err, msg)
	}

	suite := stage.Fn()
	passed := suite.Run(ctx, fmt.Sprintf("%s: %s", stageKey, stage.Name))
	return passed, nil
}

// TestStage runs tests for the current or specified stage
func TestStage(ctx context.Context, cmd *commands.Command) error {
	cfg, err := validateEnvironment()
	if err != nil {
		return err
	}

	var challengeKey string
	var stageKey string

	switch cmd.NArg() {
	case 0:
		// Use current stage from config
		challengeKey = cfg.Challenge
		stageKey = cfg.Stages.Current
	case 1:
		// lsfr test <stage>
		challengeKey = cfg.Challenge
		stageKey = cmd.Args().Slice()[0]
	default:
		return fmt.Errorf("Too many arguments.\nUsage: lsfr test [stage]")
	}

	passed, err := runStageTests(ctx, challengeKey, stageKey)
	if passed {
		fmt.Printf("\nRun %s to advance to the next stage.\n", yellow("'lsfr next'"))
	} else {
		guideURL := fmt.Sprintf("https://lsfr.io/c/%s/%s", challengeKey, stageKey)
		fmt.Printf("\nRead the guide: \033]8;;%s\033\\lsfr.io/c/%s/%s\033]8;;\033\\\n", guideURL, challengeKey, stageKey)
	}

	return err
}

// NextStage advances to the next stage after verifying current stage is complete
func NextStage(ctx context.Context, cmd *commands.Command) error {
	// Get Challenge
	cfg, err := validateEnvironment()
	if err != nil {
		return err
	}

	challenge, err := registry.GetChallenge(cfg.Challenge)
	if err != nil {
		return err
	}

	// Check if stage is completed
	currentIndex := challenge.StageIndex(cfg.Stages.Current)
	if currentIndex == -1 {
		return fmt.Errorf("Current stage '%s' not found in challenge", cfg.Stages.Current)
	}

	isCurrentCompleted := isStageCompleted(cfg.Stages.Current, cfg.Stages.Completed)
	if !isCurrentCompleted {
		passed, err := runStageTests(ctx, cfg.Challenge, cfg.Stages.Current)
		if err != nil {
			return err
		}

		if !passed {
			return fmt.Errorf("\nComplete %s before advancing.", cfg.Stages.Current)
		}

		cfg.Stages.Completed = append(cfg.Stages.Completed, cfg.Stages.Current)
	}

	// Check if already at final stage
	if currentIndex == challenge.Len()-1 {
		if !isCurrentCompleted {
			fmt.Print("\n")
		}
		fmt.Printf("You've completed all stages for %s! 🎉\n\n", cfg.Challenge)
		fmt.Printf("Share your work: tag your repo with 'lsfr-go' (or your language).\n\n")
		fmt.Println("Consider trying another challenge at \033]8;;https://lsfr.io/challenges\033\\lsfr.io/challenges\033]8;;\033\\")

		return config.Save(cfg)
	}

	// Advance to next stage
	nextStageKey := challenge.StageOrder[currentIndex+1]
	cfg.Stages.Current = nextStageKey
	if err := config.Save(cfg); err != nil {
		return err
	}

	nextStage, err := challenge.GetStage(nextStageKey)
	if err != nil {
		return err
	}

	fmt.Printf("Advanced to %s: %s\n\n", nextStageKey, nextStage.Name)
	guideURL := fmt.Sprintf("https://lsfr.io/c/%s/%s", cfg.Challenge, nextStageKey)
	fmt.Printf("Read the guide: \033]8;;%s\033\\lsfr.io/c/%s/%s\033]8;;\033\\\n\n", guideURL, cfg.Challenge, nextStageKey)
	fmt.Println("Run 'lsfr test' when ready.")

	return nil
}

// ShowStatus displays the current challenge progress and next steps
func ShowStatus(ctx context.Context, cmd *commands.Command) error {
	// Summary
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	challenge, err := registry.GetChallenge(cfg.Challenge)
	if err != nil {
		return err
	}

	fmt.Printf("%s\n\n%s\n\n", challenge.Name, challenge.Summary)

	// Progress
	fmt.Println("Progress:")
	for _, stageKey := range challenge.StageOrder {
		stage, err := challenge.GetStage(stageKey)
		if err != nil {
			continue
		}

		isCompleted := isStageCompleted(stageKey, cfg.Stages.Completed)
		if isCompleted {
			fmt.Printf("✓ %-18s - %s\n", stageKey, stage.Name)
		} else if stageKey == cfg.Stages.Current {
			fmt.Printf("→ %-18s - %s\n", stageKey, stage.Name)
		} else {
			fmt.Printf("  %-18s - %s\n", stageKey, stage.Name)
		}
	}

	// Next steps
	guideURL := fmt.Sprintf("https://lsfr.io/c/%s/%s", cfg.Challenge, cfg.Stages.Current)
	fmt.Printf("\nRead the guide: \033]8;;%s\033\\lsfr.io/c/%s/%s\033]8;;\033\\\n\n", guideURL, cfg.Challenge, cfg.Stages.Current)
	fmt.Printf("Implement %s, then run 'lsfr test'.\n", cfg.Stages.Current)

	return nil
}

// ListChallenges displays all available challenges
func ListChallenges(ctx context.Context, cmd *commands.Command) error {
	fmt.Printf("Available challenges:\n\n")

	challenges := registry.GetAllChallenges()
	for key, challenge := range challenges {
		fmt.Printf("  %-20s - %s (%d stages)\n", key, challenge.Name, challenge.Len())
	}

	fmt.Printf("\nStart with: lsfr new <challenge-name>\n")

	return nil
}
