package suite

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
)

var (
	green          = color.New(color.FgGreen).SprintFunc()
	red            = color.New(color.FgRed).SprintFunc()
	yellow         = color.New(color.FgYellow).SprintFunc()
	bold           = color.New(color.Bold).SprintFunc()
	checkMark      = green("✓")
	crossMark      = red("✗")
	skipMark       = yellow("○")
	celebrateEmoji = "🎉"
)

type Suite struct {
	name    string
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

func (s *Suite) Run(ctx context.Context, verbose bool) {
	start := time.Now()

	if verbose {
		fmt.Printf("Starting your implementation...\n")
		fmt.Printf("  Executing: %s\n", "./run.sh")
	}

	do := NewDo(verbose)
	defer do.Done()

	failed := false
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
			
			failed = true
		}
	}

	passed := 0
	for _, test := range s.tests {
		if failed {
			fmt.Printf(" %s %s [skipped]\n", skipMark, test.Name)
			continue
		}

		func() {
			defer func() {
				if err := recover(); err != nil {
					failed = true
					fmt.Printf(" %s %s\n", crossMark, test.Name)
					
					// Format error message with proper indentation
					errorMsg := fmt.Sprintf("%s", err)
					lines := strings.Split(errorMsg, "\n")
					for _, line := range lines {
						if line != "" {
							fmt.Printf("   %s\n", line)
						}
					}
				}
			}()

			test.Fn(do)

			passed++
			fmt.Printf(" %s %s\n", checkMark, test.Name)
		}()
	}

	fmt.Println()

	total := len(s.tests)
	if passed == total {
		fmt.Printf("%s %s\n", bold("PASSED"), checkMark)
		fmt.Printf("\nRun %s to advance to the next stage", yellow("'lsfr next'"))
	} else {
		fmt.Printf("\n%s %d/%d tests passed\n", bold("FAILED"), passed, total)
		
		// Add guidance based on failure patterns
		if passed == 0 {
			fmt.Printf("\nYour implementation might not be running correctly.\n")
			fmt.Printf("Try running %s directly to see any error messages.\n", yellow("./run.sh"))
		}
	}

	duration := time.Since(start).Round(time.Millisecond)
	fmt.Printf(" (took %s)\n", duration)
}
