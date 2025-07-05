# lsfr

lsfr is a CLI tool for learning how to build complex systems from scratch. It helps developers take on challenges like distributed databases, message queues, load balancers, or even LLMs by implementing them step-by-step through progressive tests.

## Repository

This repository contains the implementation of the lsfr CLI tool itself:

- **cmd/lsfr/**: Entry point for the app. It contains a small `main` function with the CLI definitions. It imports and invokes the code from the `/internal`, `/pkg`, & `challenges/` directories.
- **design/**: Contains the design & philosophy that guide the development of lsfr.
  - **design/CLI.md**: CLI.
  - **design/TESTING.md**: Testing framework.
  - **design/CHALLENGES.md**: Challenges.
- **internal/cli/**: Manages the CLI commands (called by `main`).
- **internal/config/**: Manages the lsfr.yaml file which tracks the state and config.
- **internal/registry/**: Manages the challenges registry.
- **internal/suite.**: The testing framework implementation.
- **challenges/**: Contains the actual challenges: their metadata, stages, and tests.
- **pkg/**: Library code that's ok to use by external applications.

## Coding Guidelines

### Commments

- Add comments only for things that are not immediately obvious from the code or to demarcate & help break up large functions into logical sections.
- Add comments to public functions/structs and important private functions/structs.

### Tests

- Don't attempt to run tests yourself, I'll do that.
- At most, you can suggest what tests to run.
