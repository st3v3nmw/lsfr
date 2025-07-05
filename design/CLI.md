# CLI

lsfr is a CLI tool for learning how to build complex systems from scratch. It helps developers take on challenges like distributed databases, message queues, load balancers, or even LLMs by implementing them step-by-step through progressive tests.

## Philosophy

**Succeed quietly; fail fast & loudly.** When tests pass, show brief confirmation and celebrate milestones. When things break, provide detailed diagnostics and actionable next steps.

**No surprises.** Every command does exactly what it says with no surprises or hidden side effects. Current state is visible in `lsfr.yaml`, not hidden internally.

**Make the common case effortless.** The most frequent workflow gets the shortest commands with minimal or no flags. Tracked state in `lsfr.yaml` supports this.

**Progressive disclosure of complexity.** Each stage builds on the previous one, introducing new concepts only when foundations are solid.

## Starting Challenges

`lsfr new <challenge> [<path>]`

```bash
# Basic usage (defaults to current directory)
$ lsfr new kv-store
Created challenge in current directory.
  run.sh       - Your implementation entry point
  README.md    - Challenge overview and requirements
  lsfr.yaml    - Tracks your progress

Implement http-api stage, then run 'lsfr test'.

# Specify custom path
$ lsfr new kv-store my-kv-store
Created challenge in directory: ./my-kv-store
  run.sh       - Your implementation entry point
  README.md    - Challenge overview and requirements
  lsfr.yaml    - Tracks your progress

cd my-kv-store and implement http-api stage, then run 'lsfr test'.

# State tracking & config
$ cat lsfr.yaml
challenge: kv-store
stages:
  current: http-api
  completed:

# Runner script
$ cat run.sh
#!/bin/bash

# This script builds and runs your implementation
# lsfr will execute this script to start your program
# "$@" passes any command-line arguments from lsfr to your program

echo "Replace this line with the command that runs your implementation."
# Examples:
#   go run ./cmd/server "$@"
#   python main.py "$@"
#   ./my-program "$@"

# README
$ cat README.md
# Distributed Key-Value Store Challenge

Build a distributed key-value database from scratch.
You'll start with a simple HTTP API and progressively add persistence, clustering, and fault tolerance.

## Stages

1. **http-api** - HTTP API with GET/PUT/DELETE Operations
2. **persistence** - Data survives restarts and crashes
3. **clustering** - Multi-node replication
4. **fault-tolerance** - Handle network partitions

## Getting Started

1. Edit _run.sh_ to start your implementation.
2. Run _lsfr test_ to test the current stage.
3. Run _lsfr next_ when ready to advance.

Good luck! 🚀
```

## Testing Stages

`lsfr test [<stage>]`

```bash
# Test current stage (reads from lsfr.yaml)
$ lsfr test
Running http-api: Basic Operations

✓ PUT operations
✓ GET operations
✓ DELETE operations
✓ Error handling

PASSED ✓

Your key-value store can now handle basic operations.

Run 'lsfr next' to advance to persistence.

# Test specific stage
$ lsfr test persistence
Running persistence: Data Persistence

✓ Data survives restart
✓ Handles crash recovery
✓ Maintains API compatibility

PASSED ✓

Run 'lsfr next' to advance to clustering.

# When tests fail, show detailed info automatically
$ lsfr test http-api
Running http-api: Basic Operations

✓ PUT operations
✓ GET operations
✗ Error handling

PUT http://127.0.0.1:8888/kv/ "foo"
  Expected response: "key cannot be empty"
  Actual response: ""

  Your server accepted an empty key when it should reject it.
  Add validation to return 400 Bad Request for empty keys.

FAILED ✗

Read the guide: lsfr.io/c/kv-store/http-api

# Stage that doesn't exist
$ lsfr test unknown
Stage 'unknown' does not exist for kv-store.

Available stages:
  http-api
  persistence
  clustering
  fault-tolerance
```

## Progression

`lsfr next`

```bash
# Advance to next stage
$ lsfr next
Advanced to persistence: Data Persistence

Read the guide: lsfr.io/c/kv-store/persistence

Run 'lsfr test' when ready.

# Update state/config file
$ cat lsfr.yaml
challenge: kv-store
stages:
  current: persistence
  completed:
    - http-api

# Try to advance without passing current stage
$ lsfr next
Running http-api: Basic Operations

✓ PUT operations
✗ GET operations

GET http://127.0.0.1:8888/kv/foo
  Expected 200 OK, got 404 Not Found

  Your server should return stored values with GET requests.
  Ensure your key-value storage and retrieval logic is working correctly.

FAILED ✗

Complete http-api before advancing.

# Already at final stage
$ lsfr next
You've completed all stages for kv-store! 🎉

Share your work: tag your repo with 'lsfr-go' (or your language).

Consider trying another challenge at lsfr.io/challenges.
```

## Information Commands

`lsfr status`

```bash
# Show current progress and challenge info
$ lsfr status
Distributed Key-Value Store

Learn distributed systems by building a key-value database from scratch.
You'll implement replication, consensus, and fault tolerance.

Progress:
✓ http-api          - HTTP API with GET/PUT/DELETE
→ persistence       - Survive restarts and crashes
  clustering        - Replication and eventual consistency
  fault-tolerance   - Handle network partitions

Read the guide: lsfr.io/c/kv-store/http-api

Implement persistence, then run 'lsfr test'.
```

`lsfr list`

```bash
# List available challenges
$ lsfr list
Available challenges:

  kv-store           - Distributed key-value store (4 stages)
  message-queue      - Distributed message queue (3 stages)
  load-balancer      - HTTP Load balancer (3 stages)

Start with: lsfr new <challenge-name>
```
