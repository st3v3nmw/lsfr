package suite

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

const pollInterval = 100 * time.Millisecond

// Eventually checks that the condition becomes true within the given period
func Eventually(ctx context.Context, executor func() bool, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return false
		case <-time.After(pollInterval):
			if executor() {
				return true
			}
		}
	}

	return false
}

// Consistently checks that the condition is always true for the given period
func Consistently(ctx context.Context, executor func() bool, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return false
		case <-time.After(pollInterval):
			if !executor() {
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
	// immediately executes the operation once and returns whether it meets expectations
	immediately() bool
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
}

func (a *AssertBase) formatHelp() string {
	if a.help != "" {
		return "\n\n  " + strings.ReplaceAll(a.help, "\n", "\n  ")
	}

	return ""
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
		Eventually(p.ctx, a.immediately, p.timeout)
	case TimingConsistently:
		Consistently(p.ctx, a.immediately, p.timeout)
	default:
		a.immediately()
	}

	a.check()
}

func (a *HTTPAssert) immediately() bool {
	client := &http.Client{Timeout: 30 * time.Second}
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

	return a.responseStatus == a.expectedStatus && a.responseBody == a.expectedBody
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
		Eventually(p.ctx, a.immediately, p.timeout)
	case TimingConsistently:
		Consistently(p.ctx, a.immediately, p.timeout)
	default:
		a.immediately()
	}

	a.check()
}

func (a *CLIAssert) immediately() bool {
	p := a.promise
	cmd := exec.CommandContext(p.ctx, p.command, p.args...)

	stdout, err := cmd.Output()
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			a.output = string(exitError.Stderr)
			a.exitCode = exitError.ExitCode()
		} else {
			panic(err.Error())
		}
	} else {
		a.output = string(stdout)
	}

	return a.exitCode == a.expectedExitCode && a.output == a.expectedOutput
}

func (a *CLIAssert) check() {
	p := a.promise

	if a.exitCode != a.expectedExitCode {
		msg := fmt.Sprintf("%s\n  Expected exit code %d, got %d%s",
			p.command,
			a.expectedExitCode, a.exitCode,
			a.formatHelp())
		panic(msg)
	}

	if a.output != a.expectedOutput {
		msg := fmt.Sprintf("%s\n  Expected output: %q\n  Actual output: %q%s",
			p.command,
			a.expectedOutput, a.output,
			a.formatHelp())
		panic(msg)
	}
}
