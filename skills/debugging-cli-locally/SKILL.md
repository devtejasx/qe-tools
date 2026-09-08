---
name: debugging-cli-locally
description: Use when debugging qe-tools commands locally, investigating panics or unexpected output, or setting up a local development environment
---

# Debugging the CLI Locally

## Overview

qe-tools is a Go CLI that calls external APIs (GitHub, Slack, Prow/GCS, Quay). Most commands need environment variables set to do anything useful. For debugging, you can run commands locally with `go run` or build first with `make build`.

## When to Use

- Investigating a panic or crash in a specific command
- Testing a command with real or mock inputs
- Understanding what a command does before modifying it
- **Not for**: Running in CI (CI workflows handle their own env setup)

## Quick Start

```bash
# Build the binary
make build

# Or run directly
go run main.go <command> [flags]

# Example: run estimate-review in dry mode
GITHUB_TOKEN="$YOUR_TOKEN" go run main.go estimate-review --owner konflux-ci --repo qe-tools --pr-number 42
```

## Debugging Panics

Most panics in this codebase come from one of three patterns:

### 1. Regex match index out of range

**Where it happens**: `constructMessage()` and `isJobFailed()` in `cmd/prowjob/periodicSlackReport.go` use `FindStringSubmatch` to parse Prow build logs. If the log format changes or the job fails before tests run, the regex returns nil but the code may still index into it.

**Debug**: Print the input string to see what the regex is matching against. Check if the function validates `len(matches)` before accessing capture groups like `matches[1]`.

### 2. Nil pointer from API response

**Where it happens**: `cmd/estimate/reviewTime.go` calls `client.PullRequests.ListFiles()` and `client.PullRequests.Get()`. The PR object may have nil fields even when `err` is nil.

**Debug**: Always use `GetX()` accessor methods on GitHub API objects (`pr.GetAdditions()` is nil-safe, `*pr.Additions` is not).

### 3. Missing environment variable

**Where it happens**: Commands that skip `PreRunE` validation. For example, if `GITHUB_TOKEN` or `SLACK_TOKEN` isn't set, viper returns `""` and the command proceeds with an empty token, getting a 401/403 from the API.

**Debug**: Check that `PreRunE` validates all required env vars before `RunE` executes. See `cmd/estimate/reviewTime.go` for the correct pattern.

## Running Commands Without External Services

For commands that call APIs, you have three options:

### Option A: Use real credentials

```bash
export GITHUB_TOKEN="$YOUR_GITHUB_TOKEN"
export SLACK_TOKEN="$YOUR_SLACK_TOKEN"
go run main.go <command> [flags]
```

### Option B: Mock at the test level

Write unit tests with mocked interfaces (preferred for development). See the `running-tests` skill.

### Option C: Inspect with delve

```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug a specific command
dlv debug . -- estimate-review --owner foo --repo bar --pr-number 1

# Set breakpoint
(dlv) break cmd/estimate/reviewTime.go:58
(dlv) continue
```

## Checking the Build

```bash
# Verify it compiles
go build ./...

# Verify tests pass
make test

# Verify lint passes
make lint

# All three (pre-commit runs everything)
make pre-commit
```

## Common Issues

| Symptom | Cause | Fix |
|---------|-------|-----|
| `panic: index out of range` | Regex `FindStringSubmatch` returned nil, code accessed `[1]` | Check `len(matches)` before indexing |
| `GITHUB_TOKEN must be set` | Env var not exported | `export GITHUB_TOKEN="$YOUR_TOKEN"` |
| `golangci-lint` ignores config | Config is `.golang-ci.yml` not `.golangci.yml` | Use `make lint` or `-c .golang-ci.yml` |
| Tests pass locally, fail in CI | Go version mismatch (local toolchain newer than `go.mod`'s 1.23) | Avoid language features newer than 1.23 |
| `permission denied` on binary | Built binary not executable | `chmod +x qe-tools` or use `go run` |
