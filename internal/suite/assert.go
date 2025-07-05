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
		msg := fmt.Sprintf("Expected response: %q\n  Actual response: %q", content, a.body)

		// Add contextual guidance
		if a.body == "" {
			msg += "\n\n   Your server returned an empty response."
			msg += "\n   Check that your handler is writing the response body."
		} else if content == "" {
			msg += "\n\n   Your server returned unexpected content."
			msg += "\n   Expected no response body."
		} else if content == "key cannot be empty\n" {
			msg += "\n\n   Your server should validate that keys are not empty."
			msg += "\n   Add validation to return this error message for empty keys."
		} else if content == "value cannot be empty\n" {
			msg += "\n\n   Your server should validate that values are not empty."
			msg += "\n   Add validation to return this error message for empty values."
		} else if content == "key not found\n" {
			msg += "\n\n   Your server should return this message when a key doesn't exist."
			msg += "\n   Check your key lookup logic and error handling."
		} else if content == "method not allowed\n" {
			msg += "\n\n   Your server should reject unsupported HTTP methods."
			msg += "\n   Add logic to return 405 Method Not Allowed for unsupported methods."
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
			msg += "\n\n   Could not connect to server."
			msg += "\n   Check that your run.sh script starts a server on the expected port."
		} else if code == 400 && a.statusCode == 200 {
			msg += "\n\n   Your server accepted invalid input when it should reject it."
			msg += "\n   Add validation to return 400 Bad Request for invalid requests."
		} else if code == 200 && a.statusCode == 500 {
			msg += "\n\n   Your server is returning an internal error."
			msg += "\n   Check server logs for the specific error message."
		} else if code == 404 && a.statusCode == 200 {
			msg += "\n\n   Your server returned data for a non-existent resource."
			msg += "\n   Add logic to check if the resource exists and return 404 if not found."
		} else if code == 404 && a.statusCode == 500 {
			msg += "\n\n   Your server crashed when handling a non-existent resource."
			msg += "\n   Add proper error handling to return 404 for missing resources."
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
