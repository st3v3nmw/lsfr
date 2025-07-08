# lsfr

lsfr is a CLI tool for learning how to build complex systems from scratch. It helps developers take on challenges like distributed databases, message queues, load balancers, or even LLMs by implementing them step-by-step through progressive tests.

## Structure

This repository contains the implementation of the lsfr CLI tool itself:

- `cmd/lsfr/main.go`: Entry point for the app. It contains a small `main` function with the CLI definitions. It imports and invokes code from the `/internal`, `/pkg`, & `challenges/` directories
- `design/`: Contains the design & philosophy that guide the development of lsfr
  - `CLI.md`: CLI design
  - `TESTING.md`: Testing framework design
- `internal/`: Private application and library code
  - `cli/`: Manages the CLI commands (called by `main`)
  - `config/`: Manages the lsfr.yaml file which tracks the state and config
  -  `suite/`: The testing framework implementation
  - `registry/`: Manages the challenges registry
- `challenges/`: Contains the actual challenges: their metadata, stages, and tests
- `pkg/`: Library code that's ok to use by external applications
  - `threadsafe`: Thread-safe data structures

## Coding Guidelines

### Comments

- Add comments only for things that are not immediately obvious from the code or to demarcate & help break up large functions into logical sections
- Add comments to public functions/structs and important private functions/structs
- Comments should be concise

### Tests

- DON'T attempt to run tests or builds yourself, I'll do that
