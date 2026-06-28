# Repository Guidelines

## Project Overview

`bmx` is a Go CLI for declarative local package management, currently centered on Homebrew. Desired state lives in `~/bmxfile.toml`; last applied state lives in `~/bmxfile.state.toml`. Main workflows are `bmx init`, `bmx add`, `bmx ls`, and `bmx converge`.

## Architecture & Data Flow

Layering is intentionally simple:

- `cmd/bmx` -> process entry point, signal-aware context setup.
- `internal/cli` -> urfave/cli command wiring, flags, stdio injection.
- `internal/usecase` -> command behavior/orchestration.
- `internal/config`, `internal/state`, `internal/planner`, `internal/model`, `internal/paths` -> domain logic and persistence.
- `internal/backend` -> package-manager abstraction.
- `internal/backend/brew` -> Homebrew executor.

Primary converge flow:

1. `internal/cli/app.go` resolves `--config` / `--state` via `internal/paths.Resolve`.
2. `internal/usecase/plan.go` loads desired config and applied state.
3. `internal/config.ResolveList` expands a selected list into apps.
4. `internal/planner.Build` computes installs / uninstalls / keeps.
5. `internal/usecase/converge.go` renders the plan, prompts `Apply plan? [y/N]:`, dispatches backend operations, then writes new state only on success.

Patterns to preserve:

- Keep CLI thin; put behavior in `internal/usecase`.
- Keep pure logic pure: `config.Validate`, `config.ResolveList`, `planner.Build`, `state.FromApps` do not perform I/O.
- Persist TOML in package-local helpers (`config.Load/Write`, `state.Load/Write`), not from CLI code.
- Backend selection is string-driven from `model.App.Manager`.

## Key Directories

- `cmd/bmx/` — binary entry point and build metadata variables.
- `internal/cli/` — root command and subcommand definitions.
- `internal/usecase/` — `init`, `add`, `list`, `plan`, `converge` workflows.
- `internal/config/` — desired config model, TOML parsing, validation, list resolution, config writing.
- `internal/state/` — applied-state schema and atomic state persistence.
- `internal/planner/` — pure diff logic.
- `internal/backend/` — backend interface + registry.
- `internal/backend/brew/` — Homebrew command construction/execution.
- `internal/model/` — canonical `<manager>:<package>` app model.
- `internal/paths/` — config/state path precedence.
- `.github/workflows/` — release automation.

## Development Commands

Canonical local commands come from `mise.toml`:

```bash
mise run              # default: tidy, build, tests, lint
mise run modules      # go mod tidy
mise run build        # goreleaser snapshot build to ./dist/bmx
mise run test:go-test # go test ./...
mise run test:lint    # golangci-lint run
```

Direct Go commands used in the repo:

```bash
go test ./...
go run ./cmd/bmx init --config ./bmxfile.toml
go run ./cmd/bmx ls --config ./bmxfile.toml
go run ./cmd/bmx converge --config ./bmxfile.toml --state ./bmxfile.state.toml --list macos
```


## Build, Test, and Verification Preference

- Prefer `mise` tasks over direct tool calls for build, test, lint, and change verification.
- Use `mise run build` instead of invoking GoReleaser directly.
- Use `mise run test:go-test` and `mise run test:lint` for routine verification.
- Use direct `go test` or `go run` only when a narrow targeted check is more appropriate than the repo-level `mise` task.
## Code Conventions & Common Patterns

- Formatting/linting: `gofmt` + `goimports` with local prefix `github.com/UsingCoding/bmx`; strict `golangci-lint` config in `.golangci.yml`.
- Naming: exported package APIs stay small and direct (`Load`, `Write`, `Build`, `ResolveList`, `Converge`).
- Error handling: return wrapped errors with context; avoid silent fallbacks.
- Dependency injection: pass `io.Reader`/`io.Writer` into usecases for prompts/output; pass `backend.Registry` into `Converge` rather than shelling out from CLI.
- State management: current installed set comes from `bmxfile.state.toml`, not live system inspection.
- Persistence: `state.Write` uses temp-file + rename atomic replacement.
- Config model: app names are exact strings like `brew:docker` or `brew-cask:gimp`; legacy `cask = true` is rejected with a migration hint.
- Config rewriting is normalized: `internal/config/write.go` emits deterministic TOML and does not preserve object-form app entries.

## Important Files

- `cmd/bmx/main.go` — process entry point.
- `cmd/bmx/version.go` — `version` / `commit` variables populated by GoReleaser ldflags.
- `internal/cli/app.go` — all commands, flags, stdio wiring, backend registry assembly.
- `internal/usecase/converge.go` — approval prompt, backend apply loop, state write boundary.
- `internal/usecase/plan.go` — list selection and plan construction.
- `internal/config/types.go` — TOML app parsing, including legacy cask rejection.
- `internal/state/state.go` — state schema and atomic write path.
- `internal/planner/planner.go` — install/uninstall/keep diff semantics.
- `mise.toml` — tool versions and everyday commands.
- `.golangci.yml` — lint/format policy.
- `.goreleaser.yml` — snapshot/release build config.
- `README.md` / `DIAGRAMS.md` — brief + architecture reference.

## Runtime/Tooling Preferences

- Language/runtime: Go `1.25.5` in `go.mod`; `mise.toml` pins Go `1.25`.
- CLI library: `github.com/urfave/cli/v3`.
- TOML parser: `github.com/BurntSushi/toml`.
- Tool runner: `mise`.
- Linting: `golangci-lint`.
- Release/build: `goreleaser`; static binaries with `CGO_ENABLED=0`.
- Import ordering should follow `goimports` local prefix rules.

## Testing & QA

- Framework: standard Go `testing` package only.
- Test files currently live in:
  - `internal/config/config_test.go`
  - `internal/planner/planner_test.go`
  - `internal/usecase/usecase_test.go`
- Common test patterns:
  - `t.Parallel()`
  - `t.TempDir()` for filesystem isolation
  - `bytes.Buffer` for captured output
  - `strings.NewReader(...)` for interactive input
  - small fake backend in `internal/usecase/usecase_test.go`
- Covered scenarios include config parsing, legacy cask rejection, list resolution dedupe, planner diffing, `init`, `converge` state safety, and duplicate `add` handling.
- Gaps worth checking before large changes: CLI wiring, `list`/`plan` usecases, interactive error branches, backend command construction, and standalone state package behavior.
- Before finishing non-trivial work, run at least:

```bash
go test ./...
mise run test:lint
```
