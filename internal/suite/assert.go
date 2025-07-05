package suite

import "fmt"

type ErrAssert struct {
	err error
}

func (a *ErrAssert) NoError() *ErrAssert {
	if a.err != nil {
		panic(fmt.Sprintf("An error occurred: %q", a.err))
	}

	return a
}

func (a *ErrAssert) Error(message string) *ErrAssert {
	if a.err == nil {
		panic(fmt.Sprintf("Expected err %q, none raised", message))
	}

	if a.err.Error() != message {
		panic(fmt.Sprintf("Expected err %q, got %q", message, a.err))
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
		msg := fmt.Sprintf("Expected body %q, got %q", content, a.body)
		
		// Add guidance for common body mismatches
		if a.body == "" {
			msg += "\n   Your server returned an empty response"
			msg += "\n   Check that your handler is writing the response body"
		} else if content == "" {
			msg += "\n   Your server returned unexpected content"
			msg += "\n   Expected no response body"
		}
		
		panic(msg)
	}

	return a
}

func (a *HTTPAssert) Status(code int) *HTTPAssert {
	if a.statusCode != code {
		msg := fmt.Sprintf("Expected status %d, got %d", code, a.statusCode)
		
		// Add contextual guidance
		if a.statusCode == 0 {
			msg += "\n   Could not connect to server"
			msg += "\n   Check that your run.sh script starts a server on the expected port"
		} else if code == 400 && a.statusCode == 200 {
			msg += "\n   Your server should validate input and return 400 for invalid requests"
		} else if code == 200 && a.statusCode == 500 {
			msg += "\n   Your server is returning an internal error"
			msg += "\n   Check server logs for the specific error"
		} else if code == 404 && a.statusCode == 200 {
			msg += "\n   Your server should return 404 for non-existent resources"
		}
		
		panic(msg)
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
		msg := fmt.Sprintf("Expected output %q, got %q", text, a.output)
		
		// Add guidance for CLI output mismatches
		if a.output == "" {
			msg += "\n   Your command produced no output"
			msg += "\n   Check that your script is printing to stdout"
		}
		
		panic(msg)
	}

	return a
}

func (a *CLIAssert) Exit(code int) *CLIAssert {
	if a.exitCode != code {
		msg := fmt.Sprintf("Expected exit code %d, got %d", code, a.exitCode)
		
		// Add guidance for exit code mismatches
		if code == 0 && a.exitCode != 0 {
			msg += "\n   Your command failed unexpectedly"
			msg += "\n   Check the error output for details"
		} else if code != 0 && a.exitCode == 0 {
			msg += "\n   Your command should fail in this case"
			msg += "\n   Check that your error handling is working correctly"
		}
		
		panic(msg)
	}

	return a
}
