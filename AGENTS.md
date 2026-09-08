# qe-tools - Agent Quick Reference

Go CLI (cobra) with QE utility subcommands: prow reports, OCI artifact download, review estimation, Slack messaging, webhooks.

## Key Paths

`cmd/<subcommand>/` command definitions | `pkg/` shared libraries | `config/` YAML configs + data files
`main.go` entry point | `Makefile` build/test/lint | `Dockerfile` multi-stage UBI9 image

## Build & Test

```bash
make build           # Binary with git tag version
make test            # All tests + coverage (parallel=1)
make lint            # golangci-lint -c .golang-ci.yml
make fmt             # gofumpt + gci
make pre-commit      # All hooks
```

## Adding a Subcommand

1. Create `cmd/<name>/<name>.go` with cobra `Command` var + `init()` registering flags via viper; validate in `PreRunE`
2. Register in `cmd/root.go` via `rootCmd.AddCommand()`; shared types go in `pkg/types/types.go`
3. Add tests next to source: `<name>_test.go`

## Env Vars

| Variable | Used By |
|----------|---------|
| `SLACK_TOKEN` | coffee-break, send-slack-message |
| `GITHUB_TOKEN` | health-check, estimate-review |
| `PROW_URL` | periodic-report |
| `ARTIFACT_DIR` | prowjob create-report, health-check |
| `OCI_REF` | analyze-test-results |
## CI Gates

| Workflow | Check |
|----------|-------|
| `test.yml` | `make test` on ubuntu + macOS, Go 1.23 |
| `lint.yml` | golangci-lint v1.54.2 |
| `pre-commit.yml` | All pre-commit hooks |
| `commitlint.yml` | Conventional commit messages |
| `estimate-review.yml` | Runs estimate-review on the PR |
## Testing & Security

- Every new feature, modified function, or bug fix **must** include unit tests — no exceptions
- If you encounter code without tests, add tests for it before making other changes
- Bug fixes **require** a regression test proving the fix: test the failing input before and after
- Include both positive (valid input) and negative (edge cases, nil/empty, missing data) test cases
- Tests next to source: `foo.go` -> `foo_test.go`; use table-driven patterns
- Mock external APIs (GitHub, Slack, Quay) — never call real services in tests
- Never embed tokens or secrets — use env vars via viper; HTTP clients need explicit timeouts
- Regex-based parsing must validate match results before indexing; buffered writers must be flushed/closed

## Common Pitfalls

- Viper binds lowercase flag names but reads UPPERCASE env vars (e.g., `slack_token` -> `SLACK_TOKEN`)
- Config files read relative to CWD (container WORKDIR `/qe-tools`); OCI download hardcodes `quay.io/` — other registries rejected
- Go version mismatch: `go.mod`=1.23, CI=1.23 (except `slack-message.yml`, still 1.22), Dockerfile UBI9 go-toolset varies by tag
