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

// eventually checks that the condition becomes true within the given period
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

// consistently checks that the condition is always true for the given period
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
	// Assert executes the operation and validates the result
	Assert(help string)
	// execute executes the operation once and returns whether it meets expectations
	execute() bool
	// check validates the result and panics with formatted error message on failure
	check()
	// formatHelp formats help text with proper indentation for error messages
	formatHelp() string
}

// Compile-time type checks
var _ Assert = (*HTTPAssert)(nil)
var _ Assert = (*CLIAssert)(nil)

// AssertBase provides common assertion functionality
type AssertBase struct {
	help string

	config *Config
}

func (a *AssertBase) formatHelp() string {
	return "\n\n  " + strings.ReplaceAll(a.help, "\n", "\n  ")
}

// HTTPAssert provides assertions for HTTP response validation
type HTTPAssert struct {
	AssertBase

	promise        *HTTPPromise
	responseBody   string
	responseStatus int

	expectedStatus int
	expectedBody   string
}

// Status sets the expected HTTP response status code
func (a *HTTPAssert) Status(code int) *HTTPAssert {
	a.expectedStatus = code
	return a
}

// Body sets the expected HTTP response body content
func (a *HTTPAssert) Body(content string) *HTTPAssert {
	a.expectedBody = content
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

	return a.responseStatus == a.expectedStatus &&
		a.responseBody == a.expectedBody
}

func (a *HTTPAssert) check() {
	p := a.promise

	if a.responseStatus != a.expectedStatus {
		msg := fmt.Sprintf("%s %s\n  Expected %d %s, got %d %s%s",
			p.method, p.url,
			a.expectedStatus, http.StatusText(a.expectedStatus),
			a.responseStatus, http.StatusText(a.responseStatus),
			a.formatHelp())
		panic(msg)
	}

	if a.responseBody != a.expectedBody {
		msg := fmt.Sprintf("%s %s\n  Expected response: %q\n  Actual response: %q%s",
			p.method, p.url,
			a.expectedBody, a.responseBody,
			a.formatHelp())
		panic(msg)
	}
}

// CLIAssert provides CLI command output and exit code assertions
type CLIAssert struct {
	AssertBase

	promise  *CLIPromise
	output   string
	exitCode int

	expectedOutput   string
	expectedExitCode int
}

// Exit sets the expected exit code
func (a *CLIAssert) Exit(code int) *CLIAssert {
	a.expectedExitCode = code
	return a
}

// Output sets the expected command output
func (a *CLIAssert) Output(text string) *CLIAssert {
	a.expectedOutput = text
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

	return a.exitCode == a.expectedExitCode &&
		a.output == a.expectedOutput
}

func (a *CLIAssert) check() {
	p := a.promise

	if a.exitCode != a.expectedExitCode {
		msg := fmt.Sprintf("%s %s\n  Expected exit code %d, got %d%s",
			p.command, strings.Join(p.args, " "),
			a.expectedExitCode, a.exitCode,
			a.formatHelp())
		panic(msg)
	}

	if a.output != a.expectedOutput {
		msg := fmt.Sprintf("%s %s\n  Expected output: %q\n  Actual output: %q%s",
			p.command, strings.Join(p.args, " "),
			a.expectedOutput, a.output,
			a.formatHelp())
		panic(msg)
	}
}
