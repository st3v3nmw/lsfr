package suite

import "fmt"

type ErrAssert struct {
	err error
}

func (a *ErrAssert) NoError() *ErrAssert {
	if a.err != nil {
		panic(fmt.Sprintf("an error occurred: %q", a.err))
	}

	return a
}

func (a *ErrAssert) Error(message string) *ErrAssert {
	if a.err == nil {
		panic(fmt.Sprintf("expected err %q, none raised", message))
	}

	if a.err.Error() != message {
		panic(fmt.Sprintf("expected err %q, got %q", message, a.err))
	}

	return a
}

type HTTPAssert struct {
	ErrAssert
	body       string
	statusCode int
}

func (a *HTTPAssert) Got() *HTTPAssert {
	a.NoError()

	return a
}

func (a *HTTPAssert) Body(content string) *HTTPAssert {
	if a.body != content {
		panic(fmt.Sprintf("expected body %q, got %q", content, a.body))
	}

	return a
}

func (a *HTTPAssert) Status(code int) *HTTPAssert {
	if a.statusCode != code {
		panic(fmt.Sprintf("expected status code %d, got %d", code, a.statusCode))
	}

	return a
}

type CLIAssert struct {
	ErrAssert
	output   string
	exitCode int
}

func (a *CLIAssert) Got() *CLIAssert {
	a.NoError()

	return a
}

func (a *CLIAssert) Output(text string) *CLIAssert {
	if a.output != text {
		panic(fmt.Sprintf("expected output %q, got %q", text, a.output))
	}

	return a
}

func (a *CLIAssert) Exit(code int) *CLIAssert {
	if a.exitCode != code {
		panic(fmt.Sprintf("expected exit code %d, got %d", code, a.exitCode))
	}

	return a
}
