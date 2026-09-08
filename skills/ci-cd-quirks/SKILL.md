---
name: ci-cd-quirks
description: Use when confused by CI behavior, investigating why a workflow failed, or troubleshooting environment variable issues in qe-tools
---

# CI/CD Quirks and Non-Obvious Behaviors

## Overview

The qe-tools CI has several non-obvious behaviors around Go versions, pre-commit hooks, workflow triggers, and environment variables that can cause confusing failures.

## Quirk 1: Go Version Mismatch

Three different Go versions are in play:

| Location | Go Version | Source |
|----------|-----------|--------|
| `go.mod` | 1.23 | Module minimum compatibility |
| CI (`test.yml`, `lint.yml`, `pre-commit.yml`) | 1.23 | `actions/setup-go` in workflows (`slack-message.yml` still pins 1.22) |
| Dockerfile | Varies | UBI9 `go-toolset` (version depends on tag) |

**Consequence**: Code may compile locally with Go 1.23 features but fail in the container if the UBI9 image ships an older Go. Stick to Go 1.23 language features unless `go.mod` is bumped.

## Quirk 2: Pre-commit Runs Full Test Suite

`.pre-commit-config.yaml` includes `go-test-mod` which runs `go test ./...` on every commit. This means:
- Every `git commit` triggers the full test suite
- Slow tests block the commit
- Failing tests prevent committing

**Workaround**: `git commit --no-verify` to skip hooks (not recommended for final commits).

## Quirk 3: Lint Config Uses Non-Standard Name

The golangci-lint config is `.golang-ci.yml` (with hyphen), not the standard `.golangci.yml`. This means:
- `golangci-lint run` without `-c` flag won't find the config
- Always use: `golangci-lint run -c .golang-ci.yml ./...`
- The `Makefile` already handles this via `make lint`

## Quirk 4: estimate-review Workflow Runs on Its Own PR

`.github/workflows/estimate-review.yml` triggers on `pull_request: [opened, synchronize]` and runs the `estimate-review` command against the PR itself. This is self-referential -- the tool estimates its own review time and labels the PR.

## Quirk 5: AGENTS.md Line Limit is CI-Enforced

`validate-agents-md.yaml` fails the PR if `AGENTS.md` has 60 or more lines. Uses `grep -c ''` for accurate counting (not `wc -l` which can undercount without trailing newline).

## Environment Variables

| Variable | Required By | Default | Notes |
|----------|------------|---------|-------|
| `SLACK_TOKEN` | coffee-break, send-slack-message | none | Slack Bot OAuth token |
| `CHANNEL_ID` | send-slack-message | none | Slack channel ID |
| `HACBS_CHANNEL_ID` | coffee-break | none | Different from CHANNEL_ID |
| `GITHUB_TOKEN` | health-check, estimate-review | none | Needs PR read + label write scopes |
| `PROW_URL` | periodic-report | none | GCS base URL for Prow job artifacts |
| `JOB_SPEC` | webhook report-portal | none | OpenShift CI job spec JSON |
| `ARTIFACT_DIR` | create-report, health-check | none | Local directory for output files |
| `OCI_REF` | analyze-test-results | none | Quay artifact reference |

**Viper convention**: Flag names are lowercase (`slack_token`), env vars are UPPERCASE (`SLACK_TOKEN`). Viper handles the translation when bound with `viper.BindEnv()`.

## Workflow Summary

| Workflow | Trigger | Blocks Merge |
|----------|---------|-------------|
| `test.yml` | PR + push to main (*.go) | Yes |
| `lint.yml` | PR + push to main (*.go) | Yes |
| `pre-commit.yml` | PR | Yes |
| `commitlint.yml` | PR | Yes |
| `estimate-review.yml` | PR (opened, synchronize) | No (informational) |
| `validate-agents-md.yaml` | PR (AGENTS.md changes) | Yes |
| `release.yml` | tag push | N/A |
| `coffee.yml` | cron (monthly) | N/A |
| `slack-message.yml` | cron | N/A |
