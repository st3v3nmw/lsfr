# Testing Framework Design

`internal/suite`

## Philosophy

**Test behavior, not implementation** - Tests validate external interfaces and observable behavior, giving flexibility in implementation while ensuring correctness. Focus on what the system does, not how it does it.

**Expressive, fluent test design** - Checks read like natural language through method chaining, making them easy to write, understand, and maintain. Chained methods should read left-to-right.

**Build incrementally** - Each test builds on previous ones, establishing foundations before testing advanced features. Tests run sequentially and stop on first failure which makes debugging straightforward.

**Fail with context and guidance** - When tests fail, provide detailed information about what went wrong and actionable advice for fixing it. Test output shows exactly what the system did versus what was expected, with enough context to troubleshoot without guesswork.

## Assertions

Since we're testing behavior, we'll go with [BDD-style](https://en.wikipedia.org/wiki/Behavior-driven_development) checks:

```go
HTTP(service, method, path, args...).Returns().Status(X).Body(Y).Assert(helpMessage)
```

The builder pattern allows us to write very readable checks without being stuck in error handling hell, think:

```go
req, err := HTTP(service, method, path, ...args)
if err != nil {
    ...
}

res, err := req.Returns()
if err != nil {
    ...
}

assert.Equal(res.Status, X, helpMessage)
assert.Equal(res.Body, Y, helpMessage)
```

Each individual check MUST end with `.Assert(...)` since `Assert` is the "finalizer" i.e. defer execution until assertion time for better error context.

### Domain Assertions

After `.Returns()`, it's possible to do different checks depending on the domain. For instance, for CLI, we'd have:

```go
CLI(args...).Returns().Exit(X).Output(Y).Assert(helpMessage)
```

HTTP:

```go
// .Within(Y) checks that a request completes within the specified duration
// .JSON(path, Z) checks parts of the JSON response using JSONPath syntax
HTTP(service, method, path, args...).Returns().Status(X).Within(Y).JSON("$.a[:1].b", Z).Assert(helpMessage)
```

### Timing

For more complex tests, we need to run tests over a time frame. That's where `.Eventually()` and `.Consistently()` come in:

```go
// Consistently checks that the condition is always true for a given time period (default: 5s)
HTTP("primary", "/kv/stable", "GET").Consistently().Returns().Status(200).Body("hey")

// Eventually checks that the condition becomes true within a given time period (default: 5s)
HTTP("replica", "/kv/stable", "GET").Eventually().Returns().Status(200).Body("hey")

// Override default timeouts with .Within() and .For()
HTTP("replica", "/kv/stable", "GET").Eventually().Within(10*time.Second).Returns().Status(200).Body("hey")
HTTP("primary", "/kv/stable", "GET").Consistently().For(2*time.Second).Returns().Status(200).Body("hey")
```

To add new timing modes, extend the `timing` enum and the `Eventually/Consistently` logic.

### Error Messages

Checks should have detailed error messages that help learners understand failures:

#### HTTP Request Failures

```console
PUT /kv/ "value"
  Expected response: "key cannot be empty"
  Actual response: ""

  Your server accepted an empty key when it should reject it.
  Add validation to return 400 Bad Request for empty keys.
```

#### CLI Command Failures

```console
Expected output "Usage: myapp [options]", got ""

  Your command should show usage information when run with --help.
  Check that you're printing to stdout, not stderr.
```

#### Guidelines

`.Assert(...)` should provide specific, actionable guidance:

**Good help messages:**

- Explain what the server should do: "Your server should validate that keys are not empty"
- Provide concrete next steps: "Add validation to return 400 Bad Request for empty keys"
- Reference specific concepts: "Check your key lookup logic and error handling"

**Avoid generic messages:**

- "Fix your code" (not actionable)
- "This is wrong" (doesn't explain what's expected)
- "Server error" (doesn't help debug)

## Test Harness & Runner

`suite.Do` provides the test harness and acts as the test runner:

**Service Management:**

- `do.Start(service, port, args...)` - Start a service
- `do.Kill(service)` - Send a SIGKILL to the service
- `do.Restart(service)` - Restart the service
- `do.Done()` - Clean up all services

**Testing Operations:**

- `do.HTTP(service, method, path, body, headers)` - Make HTTP requests
- `do.Exec(args...)` - Execute CLI commands
- `do.Concurrently(funcs...)` - Run operations in parallel

## Suite

`suite.Suite` orchestrates test execution with setup, test cases, and result reporting:

```go
suite.New().
    // 0
    Setup(func(do *suite.Do) {
        do.Start("primary")

        // Clear key-value store
        do.HTTP("primary", "POST", "/clear").
            Returns().Status(200).
            Assert("Your server should implement a /clear endpoint.\n" +
                "Add a POST /clear method that deletes all key-value pairs.")
    }).

    // 1
    Test("PUT Basic Operations", func(do *suite.Do) {
        do.HTTP("primary", "PUT", "/kv/kenya:capital", capital).
            Returns().Status(200).
            Assert("Your server should accept PUT requests and return 200 OK.\n" +
                "Ensure your HTTP handler processes PUT requests to /kv/{key}.")
    }).

    // 2
    Test("GET Basic Operations", func(do *suite.Do) {
        do.HTTP("primary", "GET", "/kv/kenya:capital").
            Returns().Status(200).Body("Nairobi").
            Assert("Your server should return stored values with GET requests.\n" +
                "Ensure your key-value storage and retrieval logic is working correctly.")
    }).

    // 3
    Test("Concurrent Operations", func(do *suite.Do) {
        do.Concurrently(
            func() { do.HTTP("primary", "PUT", "/kv/key1", "value1").Returns().Status(200).Assert(...) },
            func() { do.HTTP("primary", "PUT", "/kv/key2", "value2").Returns().Status(200).Assert(...) },
        )

        do.HTTP("primary", "GET", "/kv/key1").Returns().Status(200).Body("value1").Assert(...)
        do.HTTP("primary", "GET", "/kv/key2").Returns().Status(200).Body("value2").Assert(...)
    })
```

## Challenges

Challenges consist of stages that progressively and incrementally build on top of previous stages.

Challenges are defined in `challenges/<challenge>/`.

### Stage Structure

Each challenge stage follows this pattern:

```go
func HTTPAPI() *suite.Suite {
	return suite.New().
		// 0
		Setup(...).

		// 1
		Test(...).

		// 2
		Test(...)
}
```

Stages are stored in `challenges/<challenge>/<stage.go>`.

### Discovery

Each challenge should have an `init` file `challenges/<challenge>/init.go` that registers the challenge with lsfr's registry:

```go
func init() {
	challenge := &registry.Challenge{
		Name: "Distributed Key-Value Store",
		Summary: `<short-description>`,
		Concepts: []string{"Storage Engines", "Fault Tolerance", "Replication", "Consensus"},
	}

	challenge.AddStage("http-api", "HTTP API with GET/PUT/DELETE Operations", HTTPAPI)

	registry.RegisterChallenge("kv-store", challenge)
}
```

For auto-discovery, each challenge should also be imported in `challenges/challenges.go`:

```go
import (
	_ "github.com/st3v3nmw/lsfr/challenges/kvstore"
)
```
