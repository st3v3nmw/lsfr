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
	setupFn func(*Do)
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
func (s *Suite) Setup(fn func(*Do)) *Suite {
	s.setupFn = fn
	return s
}

// Test adds a test case to the suite
func (s *Suite) Test(name string, fn func(*Do)) *Suite {
	s.tests = append(s.tests, TestFunc{Name: name, Fn: fn})

	return s
}

// Run executes the test suite and returns results
func (s *Suite) Run(ctx context.Context, name string) bool {
	fmt.Printf("Running %s\n\n", name)

	do := NewDo(ctx)
	defer do.Done()

	// Run setup function if defined
	var failed bool
	if s.setupFn != nil {
		func() {
			defer func() {
				if err := recover(); err != nil {
					failed = true

					fmt.Printf("%s %s\n", crossMark, "SETUP")
					fmt.Printf("\n%s\n", err)
				}
			}()

			s.setupFn(do)
		}()
	}

	// Run each test, stopping on first failure or cancellation
	for _, test := range s.tests {
		if failed {
			break
		}

		select {
		case <-ctx.Done():
			return false
		default:
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
		}()

		fmt.Printf("%s %s\n", checkMark, test.Name)
	}

	if failed {
		fmt.Printf("\n%s %s\n", bold("FAILED"), crossMark)
	} else {
		fmt.Printf("\n%s %s\n", bold("PASSED"), checkMark)
	}

	return !failed
}
