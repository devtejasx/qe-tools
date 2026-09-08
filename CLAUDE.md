# qe-tools - Claude AI Context

## Purpose

Go CLI (cobra-based) containing utility tools used by RHTAP QE (Red Hat Trusted Application Pipeline Quality Engineering). The binary ships as a container image and is invoked from CI pipelines, GitHub Actions, and Tekton tasks.

### Subcommands

| Command | What it does |
|---------|-------------|
| `prowjob create-report` | Scans Prow job GCS artifacts, produces JUnit XML + HTML report |
| `prowjob periodic-report` | Parses periodic job build logs, summarizes pass/fail for Slack |
| `prowjob health-check` | Checks external service status pages, optionally comments on PR |
| `estimate-review` | Estimates PR review time from file diffs, optionally adds GitHub label |
| `download` | Downloads OCI artifacts from Quay, supports single repo or multi-repo with time filter |
| `analyze-test-results` | Scans OCI artifact for JUnit + logs, classifies failure type, writes analysis |
| `send-slack-message` | Posts a message to a Slack channel |
| `coffee-break` | Randomly picks monthly coffee break groups, posts to Slack |
| `webhook report-portal` | Triggers ReportPortal webhook from OpenShift CI job spec |

### Repository Layout

```
main.go                          # Entry point (delegates to cmd.Execute())
cmd/                             # Cobra command definitions (one package per subcommand)
  root.go                        # Root command, registers all subcommands
  analyzetestresults/            # analyze-test-results command
  coffeebreak/                   # coffee-break command
  estimate/                      # estimate-review command
  oci/                           # download command
  prowjob/                       # prowjob parent + create-report, periodic-report, health-check
  sendslackmessage/              # send-slack-message command
  webhook/                       # webhook parent + report-portal
pkg/                             # Shared libraries
  oci/                           # OCI artifact scanning, downloading, blob handling
  prow/                          # Prow job GCS artifact scanner
  testresults/                   # JUnit analysis + Markdown report formatting
  webhook/                       # Webhook creation + HMAC signing
  customjunit/                   # Custom JUnit types for Report Portal
  status/                        # External service status page types
  types/                         # Shared constants (param names, env vars)
  utils/                         # Utilities (ParseRepoAndTag)
config/                          # YAML configs + data files
  estimate/config.yaml           # Review time estimation weights and labels
  coffee-break/                  # Participants list + last-week state
  health-check/config.yaml       # External service status page URLs
```

### Data Flow

- **Prow reports**: GCS bucket (test-platform-results) -> `pkg/prow` scanner -> JUnit/HTML via `cmd/prowjob/createReport.go`
- **OCI artifacts**: Quay registry -> `pkg/oci` controller (ORAS library) -> local disk -> `pkg/testresults` analysis
- **Review estimation**: GitHub API (PR files + commits) -> weighted formula -> label via GitHub Issues API
- **Slack**: Slack API (direct HTTP or `slack-go` library) -> channel messages

## Build

```bash
make build          # Build binary with git tag version
make install        # Install to $GOPATH/bin
make bootstrap      # Install tool dependencies (gci, gofumpt, etc.)
```

### Container Image

```bash
# Multi-stage build: UBI9 go-toolset -> UBI9 minimal
# Runs as non-root (65532:65532)
podman build -t qe-tools .
```

### Lint and Format

```bash
make lint           # golangci-lint with .golang-ci.yml config
make fmt            # gofumpt + gci
make pre-commit     # All pre-commit hooks (build, test, vet, revive, gosec, fumpt, imports, lint, critic)
```

## Test

```bash
make test           # Run all tests with coverage (parallel=1)
make cover          # Run tests with race detector + coverage
go test ./cmd/prowjob/... -v    # Single package
```

Tests live next to source: `foo.go` -> `foo_test.go`. No external dependencies needed -- all tests use mocked data or local fixtures.

### Testing Expectations

- Every new feature, modified function, or bug fix **must** include unit tests
- If you encounter code without tests, add tests for it before making other changes
- Bug fixes require a regression test that proves the fix works
- Include both positive (valid input) and negative (edge cases, nil/empty, missing data) test cases
- Use table-driven test patterns with descriptive names
- Mock external APIs -- never call GitHub, Slack, or Quay in tests
- Test all code paths in functions with conditional branches (especially regex match/no-match, success/failure, empty/populated data)

### CI Gates (GitHub Actions)

| Workflow | Trigger | What it does |
|----------|---------|-------------|
| `test.yml` | PR + push to main (*.go) | `make test` on ubuntu + macOS, Go 1.23, uploads Codecov |
| `lint.yml` | PR + push to main (*.go) | golangci-lint v1.54.2 |
| `pre-commit.yml` | PR | All pre-commit hooks |
| `commitlint.yml` | PR | Conventional commit message validation |
| `release.yml` | tag push | GoReleaser |
| `estimate-review.yml` | PR | Runs estimate-review on the PR itself, adds label |
| `coffee.yml` | cron (monthly) | Runs coffee-break, commits updated state |
| `slack-message.yml` | cron | Prow periodic report to Slack |

### Tekton Pipelines

`.tekton/qe-tools-pull-request.yaml` and `.tekton/qe-tools-push.yaml` handle Konflux CI builds.

## Design Choices

### Cobra + Viper Pattern

Every subcommand follows the same pattern: flags defined in `init()`, bound to viper for env var fallback, validated in `PreRunE`, executed in `RunE`. Config files are read via `viper.ReadInConfig()`. This means any flag can alternatively be set via environment variable.

### OCI Architecture (pkg/oci)

The OCI package uses ORAS (OCI Registry As Storage) for artifact operations:
- `Controller` orchestrates: creates OCI store, manages blob/output directories
- `ProcessTag` copies manifest to local store, then extracts blobs to output directory
- `ProcessRepositories` fans out concurrently with a semaphore (10 goroutines)
- `FetchTags` paginates the Quay API (100 tags/page) to discover available artifacts

### Review Time Estimation

Uses a weighted formula: `commitCoefficient * fileCoefficient * sum(fileTimes)` where file times depend on extension weights (Go=1, YAML=2, etc.), additions use base weight (1.0), deletions use deletion weight (0.5). Coefficients are capped at ceiling values. Labels are assigned by matching estimated time to configured thresholds.

### Prow Artifact Scanner

Scans GCS bucket structure for Prow job artifacts. Maps step names to artifact filenames (finished.json, build-log.txt, junit.xml). Produces aggregated JUnit reports with openshift-ci metadata as a synthetic test suite.

## Pitfalls

- **Environment variables are uppercase**: Viper binds lowercase names but reads uppercase env vars (e.g., `slack_token` flag reads `SLACK_TOKEN`)
- **Config file location matters**: Commands reading config files use paths relative to CWD; the Dockerfile copies `config/` into the image at `/qe-tools/config`
- **Quay-only**: The OCI download command hardcodes `quay.io/` prefix validation -- other registries are rejected
- **HTTP clients use default timeouts**: Several commands make HTTP calls using the default Go client with no explicit timeout -- this can hang indefinitely on network issues
- **Buffered I/O needs explicit flushing**: Any code using `bufio.NewWriter` must call `Flush()` or `Close()` before the file handle is closed, or data may be silently truncated
- **Regex match safety**: Functions parsing text with regex should always validate the match result length before accessing capture group indices -- `FindStringSubmatch` returns nil on no match
- **Fallthrough in match functions**: Functions that iterate over a set of conditions and return on the first match should handle the case where nothing matches
- **pre-commit hooks run full test suite**: `go-test-mod` in `.pre-commit-config.yaml` runs all tests on every commit, which can be slow
- **Go version mismatch**: `go.mod` says 1.23, most CI workflows use 1.23 but `slack-message.yml` still pins 1.22, Dockerfile uses UBI9 go-toolset (Go version varies by tag)
