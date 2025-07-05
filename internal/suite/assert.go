package suite

import (
	"fmt"
	"strings"
)

// Assert interface that all assertion types must implement
// T is the concrete assertion type (HTTPAssert, CLIAssert, etc.)
type Assert[T any] interface {
	// Error handling
	NoError() *ErrAssert
	Error(message string) *ErrAssert
	Got() T

	// Help
	setHelp(message string)
	formatHelp() string
	WithHelp(message string) T
}

var _ Assert[*HTTPAssert] = (*HTTPAssert)(nil)
var _ Assert[*CLIAssert] = (*CLIAssert)(nil)

// ErrAssert provides basic error assertion functionality
type ErrAssert struct {
	err error
}

// NoError asserts that no error occurred
func (a *ErrAssert) NoError() *ErrAssert {
	if a.err != nil {
		panic(fmt.Sprintf("An error occurred: %q", a.err))
	}

	return a
}

// Error asserts that an error occurred with the expected message
func (a *ErrAssert) Error(message string) *ErrAssert {
	if a.err == nil {
		panic(fmt.Sprintf("Expected err %q, none raised", message))
	}

	if a.err.Error() != message {
		panic(fmt.Sprintf("Expected err %q, got %q", message, a.err))
	}

	return a
}

// Help provides shared help text functionality
type Help struct {
	help string
}

// setHelp formats and stores help text with proper indentation
func (h *Help) setHelp(message string) {
	lines := strings.Split(message, "\n")
	h.help = strings.Join(lines, "\n  ")
}

// formatHelp returns formatted help text for error messages
func (h *Help) formatHelp() string {
	if h.help != "" {
		return "\n\n  " + h.help
	}

	return ""
}

// HTTPAssert provides HTTP response assertions with contextual error messages
type HTTPAssert struct {
	ErrAssert
	Help

	// Request fields
	requestMethod string
	requestURL    string
	requestBody   string

	// Response fields
	responseBody   string
	responseStatus int
}

// WithHelp adds contextual guidance for when this assertion fails
func (a *HTTPAssert) WithHelp(message string) *HTTPAssert {
	a.setHelp(message)

	return a
}

// Got checks for errors and returns the assertion for further chaining
func (a *HTTPAssert) Got() *HTTPAssert {
	a.NoError()

	return a
}

// Body asserts the HTTP response body matches the expected content
func (a *HTTPAssert) Body(content string) *HTTPAssert {
	if a.responseBody != content {
		msg := fmt.Sprintf("%s %s", a.requestMethod, a.requestURL)
		if a.requestBody != "" {
			msg += fmt.Sprintf(" \"%s\"", a.requestBody)
		}
		msg += fmt.Sprintf("\n  Expected response: %q\n  Actual response: %q", content, a.responseBody)

		msg += a.formatHelp()

		panic(msg)
	}

	return a
}

// Status asserts the HTTP response status code matches the expected value
func (a *HTTPAssert) Status(code int) *HTTPAssert {
	if a.responseStatus != code {
		msg := fmt.Sprintf("%s %s", a.requestMethod, a.requestURL)
		if a.requestBody != "" {
			msg += fmt.Sprintf(" \"%s\"", a.requestBody)
		}
		msg += fmt.Sprintf("\n  Expected %d, got %d", code, a.responseStatus)

		msg += a.formatHelp()

		panic(msg)
	}

	return a
}

// CLIAssert provides CLI command output and exit code assertions
type CLIAssert struct {
	ErrAssert
	Help

	output   string
	exitCode int
}

// WithHelp adds contextual guidance for when this assertion fails
func (a *CLIAssert) WithHelp(message string) *CLIAssert {
	a.setHelp(message)

	return a
}

// Got checks for errors and returns the assertion for further chaining
func (a *CLIAssert) Got() *CLIAssert {
	a.NoError()

	return a
}

// Output asserts the command output matches the expected text
func (a *CLIAssert) Output(text string) *CLIAssert {
	if a.output != text {
		msg := fmt.Sprintf("Expected output %q, got %q", text, a.output)
		msg += a.formatHelp()

		panic(msg)
	}

	return a
}

// Exit asserts the command exit code matches the expected value
func (a *CLIAssert) Exit(code int) *CLIAssert {
	if a.exitCode != code {
		msg := fmt.Sprintf("Expected exit code %d, got %d", code, a.exitCode)
		msg += a.formatHelp()

		panic(msg)
	}

	return a
}
