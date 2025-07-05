package suite

import (
	"context"
	"fmt"

	"github.com/fatih/color"
)

var (
	green     = color.New(color.FgGreen).SprintFunc()
	red       = color.New(color.FgRed).SprintFunc()
	yellow    = color.New(color.FgYellow).SprintFunc()
	bold      = color.New(color.Bold).SprintFunc()
	checkMark = green("✓")
	crossMark = red("✗")
)

// Suite represents a test suite with setup and test functions
type Suite struct {
	setupFn func(*Do) error
	tests   []TestFunc
}

// TestFunc represents a single test case with name and function
type TestFunc struct {
	Name string
	Fn   func(*Do)
}

// New creates a new empty test suite
func New() *Suite {
	return &Suite{tests: make([]TestFunc, 0)}
}

// Setup adds a setup function that runs before all tests
func (s *Suite) Setup(fn func(*Do) error) *Suite {
	s.setupFn = fn

	return s
}

// Test adds a test case to the suite
func (s *Suite) Test(name string, fn func(*Do)) *Suite {
	s.tests = append(s.tests, TestFunc{Name: name, Fn: fn})

	return s
}

// Run executes the test suite and displays results
func (s *Suite) Run(ctx context.Context, challengeKey, stageKey, stageName string) bool {
	fmt.Printf("Running %s: %s\n\n", stageKey, stageName)

	do := NewDo()
	defer do.Done()

	if s.setupFn != nil {
		if err := s.setupFn(do); err != nil {
			fmt.Printf("%s Setup failed\n", crossMark)
			fmt.Printf("   %s\n", err)
		}
	}

	// Run tests sequentially, stopping on first failure
	passed := 0
	failed := false
	for _, test := range s.tests {
		if failed {
			break
		}

		func() {
			defer func() {
				if err := recover(); err != nil {
					failed = true

					fmt.Printf("%s %s\n", crossMark, test.Name)
					fmt.Printf("\n%s\n", err)
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
	} else {
		fmt.Printf("%s %s\n", bold("FAILED"), crossMark)
	}

	return passed == total
}
