package suite

import (
	"context"
	"fmt"
	"strings"

	"github.com/fatih/color"
)

var (
	green          = color.New(color.FgGreen).SprintFunc()
	red            = color.New(color.FgRed).SprintFunc()
	yellow         = color.New(color.FgYellow).SprintFunc()
	bold           = color.New(color.Bold).SprintFunc()
	checkMark      = green("✓")
	crossMark      = red("✗")
	celebrateEmoji = "🎉"
)

type Suite struct {
	setupFn func(*Do) error
	tests   []TestFunc
}

type TestFunc struct {
	Name string
	Fn   func(*Do)
}

func New() *Suite {
	return &Suite{tests: make([]TestFunc, 0)}
}

func (s *Suite) Setup(fn func(*Do) error) *Suite {
	s.setupFn = fn
	return s
}

func (s *Suite) Test(name string, fn func(*Do)) *Suite {
	s.tests = append(s.tests, TestFunc{Name: name, Fn: fn})
	return s
}

func (s *Suite) Run(ctx context.Context, stageKey, stageName, stageSummary string) {
	fmt.Printf("Running %s: %s\n\n", stageKey, stageName)

	do := NewDo()
	defer do.Done()

	if s.setupFn != nil {
		if err := s.setupFn(do); err != nil {
			fmt.Printf("%s Setup failed\n", crossMark)
			fmt.Printf("   %s\n", err)

			// Add actionable guidance based on error type
			if strings.Contains(err.Error(), "permission denied") {
				fmt.Printf("   Try: %s\n", yellow("chmod +x run.sh"))
			} else if strings.Contains(err.Error(), "no such file") {
				fmt.Printf("   Create an executable run.sh script that runs your implementation\n")
			} else if strings.Contains(err.Error(), "connection refused") {
				fmt.Printf("   Possible issues:\n")
				fmt.Printf("   - Server not starting on the expected port\n")
				fmt.Printf("   - Server crashing during startup\n")
				fmt.Printf("   Debug with: %s and check for error messages\n", yellow("./run.sh"))
			}
		}
	}

	passed := 0
	failed := false
	for _, test := range s.tests {
		if failed {
			break
		}

		func() {
			defer func() {
				if err := recover(); err != nil {
					fmt.Printf("%s %s\n", crossMark, test.Name)
					fmt.Println()

					// Format error message with proper indentation
					errorMsg := fmt.Sprintf("%s", err)
					lines := strings.Split(errorMsg, "\n")
					for _, line := range lines {
						if line != "" {
							fmt.Printf("  %s\n", line)
						}
					}

					failed = true
				}
			}()

			test.Fn(do)

			passed++
			fmt.Printf("%s %s\n", checkMark, test.Name)
		}()
	}

	fmt.Println()

	total := len(s.tests)
	if passed == total {
		fmt.Printf("%s %s\n", bold("PASSED"), checkMark)
		fmt.Printf("\n%s\n", stageSummary)
		fmt.Printf("\nRun %s to advance to the next stage.\n", yellow("'lsfr next'"))
	} else {
		fmt.Printf("%s %s\n", bold("FAILED"), crossMark)
	}
}
