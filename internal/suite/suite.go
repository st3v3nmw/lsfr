package suite

import (
	"context"
	"fmt"
	"time"
)

type Suite struct {
	name    string
	setupFn func(*Do) error
	tests   []TestFunc
}

func New(name string) *Suite {
	return &Suite{
		name:  name,
		tests: make([]TestFunc, 0),
	}
}

func (s *Suite) Setup(fn func(*Do) error) *Suite {
	s.setupFn = fn
	return s
}

func (s *Suite) Test(name string, fn func(*Do)) *Suite {
	s.tests = append(s.tests, TestFunc{Name: name, Fn: fn})
	return s
}

func (s *Suite) Run(ctx context.Context, binary string) Report {
	result := Report{}
	start := time.Now()

	do := NewDo(binary)
	defer do.Done()

	if s.setupFn != nil {
		if err := s.setupFn(do); err != nil {
			result.Failed = len(s.tests)
			result.Errors = append(result.Errors, TestError{
				TestName: "Setup",
				Message:  err.Error(),
			})

			result.Duration = time.Since(start)
			return result
		}
	}

	for _, test := range s.tests {
		func() {
			defer func() {
				if r := recover(); r != nil {
					result.Failed++
					result.Errors = append(result.Errors, TestError{
						TestName: test.Name,
						Message:  fmt.Sprintf("panic: %v", r),
					})
				}
			}()

			test.Fn(do)
			result.Passed++
		}()
	}

	result.Duration = time.Since(start)
	return result
}

type Report struct {
	Passed   int
	Failed   int
	Errors   []TestError
	Duration time.Duration
}

type TestFunc struct {
	Name string
	Fn   func(*Do)
}

type TestError struct {
	TestName string
	Message  string
}
