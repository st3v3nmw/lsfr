# lsfr

lsfr is a CLI tool for learning to build complex systems from scratch. Take on challenges like distributed databases, message queues, load balancers, or even LLMs by implementing them step-by-step through progressive tests.

This repository contains the implementation of the lsfr CLI tool itself:

- **cmd/lsfr/**: Entry point for the app. It should contain a small `main` function that imports and invokes the code from the `/internal`, `/pkg`, & `challenges/` directories and nothing else.
- **design/**: Contains the design & philosophy that guide the development of lsfr
  - **design/CLI.md**: CLI design
  - **design/TESTING.md**: Design of the testing framework
  - **design/CHALLENGES.md**: How to write challenges
- **internal/config/**: Manages the lsfr.yaml file which tracks the state
- **internal/registry/**: Manages the challenges registry
- **internal/suite.**: The testing framework
- **challenges/**: Contains the actual challenges: their metadata, stages, and tests
- **pkg/**: Library code that's ok to use by external applications
