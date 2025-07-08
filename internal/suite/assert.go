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

// Eventually checks that the condition becomes true within the given period
func Eventually(ctx context.Context, executor func() bool, timeout time.Duration) bool {
	interval := 100 * time.Millisecond
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return false
		case <-time.After(interval):
			if executor() {
				return true
			}
		}
	}

	return false
}

// Consistently checks that the condition is always true for the given period
func Consistently(ctx context.Context, executor func() bool, timeout time.Duration) bool {
	interval := 100 * time.Millisecond
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return false
		case <-time.After(interval):
			if !executor() {
				return false
			}
		}
	}

	return true
}

// DomainAssert represents assertion behavior for domain-specific operations
type DomainAssert interface {
	// Core
	Assert(help string)
	immediately() bool

	// Helpers
	formatHelp() string
}

// Compile-time type checks
var _ DomainAssert = (*HTTPAssert)(nil)
var _ DomainAssert = (*CLIAssert)(nil)

type BaseAssert struct {
	help string
}

func (a *BaseAssert) formatHelp() string {
	if a.help != "" {
		return "\n\n  " + strings.ReplaceAll(a.help, "\n", "\n  ")
	}

	return ""
}

type HTTPAssert struct {
	BaseAssert

	promise        *HTTPPromise
	responseBody   string
	responseStatus int

	expectedStatus int
	expectedBody   string
}

// Body sets the expected HTTP response body content
func (a *HTTPAssert) Body(content string) *HTTPAssert {
	a.expectedBody = content
	return a
}

// Status sets the expected HTTP response status code
func (a *HTTPAssert) Status(code int) *HTTPAssert {
	a.expectedStatus = code
	return a
}

// Execute and assert results
func (a *HTTPAssert) Assert(help string) {
	a.help = help

	// Execute deferred operation
	p := a.promise
	switch p.timing {
	case TimingEventually:
		Eventually(p.ctx, a.immediately, 5*time.Second)
	case TimingConsistently:
		Consistently(p.ctx, a.immediately, 5*time.Second)
	default:
		a.immediately()
	}

	// Assert
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

// CLIAssert provides CLI command output and exit code assertions
type CLIAssert struct {
	BaseAssert

	promise  *CLIPromise
	output   string
	exitCode int

	expectedOutput   string
	expectedExitCode int
}

// Output sets the expected command output
func (a *CLIAssert) Output(text string) *CLIAssert {
	a.expectedOutput = text
	return a
}

// Exit sets the expected exit code
func (a *CLIAssert) Exit(code int) *CLIAssert {
	a.expectedExitCode = code
	return a
}

// Execute and assert results
func (a *CLIAssert) Assert(help string) {
	a.help = help

	// Execute deferred operation
	p := a.promise
	switch p.timing {
	case TimingEventually:
		Eventually(p.ctx, a.immediately, 5*time.Second)
	case TimingConsistently:
		Consistently(p.ctx, a.immediately, 5*time.Second)
	default:
		a.immediately()
	}

	// Assert
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
