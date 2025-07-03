package suite

import (
	"context"
	"fmt"
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

	do := NewDo(verbose)
	defer do.Done()

	failed := false
	if s.setupFn != nil {
		if err := s.setupFn(do); err != nil {
			// TODO: Print this error
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
				if r := recover(); r != nil {
					failed = true
					fmt.Printf(" %s %s\n", crossMark, test.Name)
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
		fmt.Printf("\nRun %s to advance to stage %d", yellow("'lsfr next'"), +1)
	} else {
		fmt.Printf("%d/%d tests passed", passed, total)
	}

	duration := time.Since(start).Round(time.Millisecond)
	fmt.Printf(" (took %s)\n", duration)
}
