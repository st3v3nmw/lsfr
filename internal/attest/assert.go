package attest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// eventually checks that the condition becomes true within the given period.
func eventually(ctx context.Context, condition func() bool, timeout, pollInterval time.Duration) bool {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return false
		case <-time.After(pollInterval):
			if condition() {
				return true
			}
		}
	}

	return false
}

// consistently checks that the condition is always true for the given period.
func consistently(ctx context.Context, condition func() bool, timeout, pollInterval time.Duration) bool {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return false
		case <-time.After(pollInterval):
			if !condition() {
				return false
			}
		}
	}

	return true
}

// Assert defines the interface for executing and validating test assertions.
// Implementations handle domain-specific operations like HTTP requests or CLI commands.
type Assert interface {
	// Assert executes the operation and validates the result.
	Assert(help string)
	// execute executes the operation once and returns whether it meets expectations.
	execute() bool
	// check validates the result and panics with formatted error message on failure.
	check()
	// formatHelp formats help text with proper indentation for error messages.
	formatHelp() string
}

var _ Assert = (*HTTPAssert)(nil)
var _ Assert = (*CLIAssert)(nil)

// AssertBase provides common assertion functionality.
type AssertBase struct {
	help string

	config *Config
}

func (a *AssertBase) formatHelp() string {
	return "\n\n  " + strings.ReplaceAll(a.help, "\n", "\n  ")
}

// HTTPAssert provides assertions for HTTP response validation.
type HTTPAssert struct {
	AssertBase

	promise        *HTTPPromise
	responseBody   string
	responseStatus int

	statusMatchers []Matcher[int]
	bodyMatchers   []Matcher[string]
	jsonMatchers   []JSONFieldMatcher
}

// Status adds expected HTTP response status code matchers.
// All matchers must pass.
func (a *HTTPAssert) Status(matchers ...Matcher[int]) *HTTPAssert {
	a.statusMatchers = append(a.statusMatchers, matchers...)
	return a
}

// Body adds expected HTTP response body matchers.
// All matchers must pass.
func (a *HTTPAssert) Body(matchers ...Matcher[string]) *HTTPAssert {
	a.bodyMatchers = append(a.bodyMatchers, matchers...)
	return a
}

// JSON adds expected matchers for a JSON field at the given gjson path.
// All matchers must pass.
func (a *HTTPAssert) JSON(path string, matchers ...Matcher[string]) *HTTPAssert {
	for _, matcher := range matchers {
		a.jsonMatchers = append(a.jsonMatchers, JSONFieldMatcher{
			Path:    path,
			Matcher: matcher,
		})
	}

	return a
}

func (a *HTTPAssert) Assert(help string) {
	a.help = help

	p := a.promise
	switch p.timing {
	case TimingEventually:
		eventually(p.ctx, a.execute, p.timeout, a.config.RetryPollInterval)
	case TimingConsistently:
		consistently(p.ctx, a.execute, p.timeout, a.config.RetryPollInterval)
	default:
		a.execute()
	}

	a.check()
}

func (a *HTTPAssert) execute() bool {
	client := &http.Client{Timeout: a.config.ExecuteTimeout}
	p := a.promise

	req, err := http.NewRequestWithContext(p.ctx, p.method, p.url, bytes.NewReader(p.body))
	if err != nil {
		panic(fmt.Sprintf("An error occurred: %v", err))
	}

	for key, value := range p.headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		panic(fmt.Sprintf("An error occurred: %v", err))
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(fmt.Sprintf("An error occurred: %v", err))
	}

	a.responseBody = string(responseBody)
	a.responseStatus = resp.StatusCode

	return checkAll(a.responseStatus, a.statusMatchers, nil) &&
		checkAll(a.responseBody, a.bodyMatchers, nil) &&
		checkAllJSON(a.responseBody, a.jsonMatchers, nil)
}

func (a *HTTPAssert) check() {
	p := a.promise

	checkAll(a.responseStatus, a.statusMatchers, func(m Matcher[int], actual int) {
		msg := fmt.Sprintf("%s %s\n  Expected status: %s\n  Actual status: %d %s%s",
			p.method, p.url, m.Expected(), actual,
			http.StatusText(actual), a.formatHelp())
		panic(msg)
	})

	checkAll(a.responseBody, a.bodyMatchers, func(m Matcher[string], actual string) {
		msg := fmt.Sprintf("%s %s\n  Expected response: %s\n  Actual response: %q%s",
			p.method, p.url, m.Expected(), actual, a.formatHelp())
		panic(msg)
	})

	checkAllJSON(a.responseBody, a.jsonMatchers, func(m JSONFieldMatcher, actual any) {
		msg := fmt.Sprintf("%s %s\n  Expected JSON field %q: %s\n  Actual value: %v%s",
			p.method, p.url, m.Path, m.Matcher.Expected(), actual, a.formatHelp())
		panic(msg)
	})
}

// CLIAssert provides CLI command output and exit code assertions.
type CLIAssert struct {
	AssertBase

	promise  *CLIPromise
	output   string
	exitCode int

	exitMatchers   []Matcher[int]
	outputMatchers []Matcher[string]
}

// ExitCode adds expected exit code matchers.
// All matchers must pass.
func (a *CLIAssert) ExitCode(matchers ...Matcher[int]) *CLIAssert {
	a.exitMatchers = append(a.exitMatchers, matchers...)
	return a
}

// Output adds expected command output matchers.
// All matchers must pass.
func (a *CLIAssert) Output(matchers ...Matcher[string]) *CLIAssert {
	a.outputMatchers = append(a.outputMatchers, matchers...)
	return a
}

func (a *CLIAssert) Assert(help string) {
	a.help = help

	p := a.promise
	switch p.timing {
	case TimingEventually:
		eventually(p.ctx, a.execute, p.timeout, a.config.RetryPollInterval)
	case TimingConsistently:
		consistently(p.ctx, a.execute, p.timeout, a.config.RetryPollInterval)
	default:
		a.execute()
	}

	a.check()
}

func (a *CLIAssert) execute() bool {
	p := a.promise

	ctx, cancel := context.WithTimeout(p.ctx, a.config.ExecuteTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, p.command, p.args...)

	stdout, err := cmd.Output()
	if err != nil {
		var exitError *exec.ExitError
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			a.output = fmt.Sprintf("%s timed out after %s", p.command, a.config.ExecuteTimeout)
			a.exitCode = -1
		} else if errors.Is(ctx.Err(), context.Canceled) {
			a.output = fmt.Sprintf("%s was cancelled", p.command)
			a.exitCode = -1
		} else if errors.As(err, &exitError) {
			a.output = string(exitError.Stderr)
			a.exitCode = exitError.ExitCode()
		} else {
			panic(err.Error())
		}
	} else {
		a.output = string(stdout)
		a.exitCode = 0
	}

	return checkAll(a.exitCode, a.exitMatchers, nil) &&
		checkAll(a.output, a.outputMatchers, nil)
}

func (a *CLIAssert) check() {
	p := a.promise

	checkAll(a.exitCode, a.exitMatchers, func(m Matcher[int], actual int) {
		msg := fmt.Sprintf("%s %s\n  Expected exit code: %s\n  Actual exit code: %d%s",
			p.command, strings.Join(p.args, " "), m.Expected(), actual,
			a.formatHelp())
		panic(msg)
	})

	checkAll(a.output, a.outputMatchers, func(m Matcher[string], actual string) {
		msg := fmt.Sprintf("%s %s\n  Expected output: %s\n  Actual output: %q%s",
			p.command, strings.Join(p.args, " "), m.Expected(), actual,
			a.formatHelp())
		panic(msg)
	})
}
