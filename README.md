# BMX

`bmx` is a small CLI for managing local packages from a declarative TOML file.

Today it targets Homebrew. You describe what should be installed, review the plan, then apply it.

## Core Principles

- **Declarative** — `bmxfile.toml` is the desired state.
- **Safe** — `bmx converge` shows a plan and asks before changing anything.
- **Repeatable** — applied state is tracked in `bmxfile.state.toml`.
- **Simple** — packages are plain strings like `brew:git` or `brew-cask:gimp`.

## How It Works

- `~/bmxfile.toml` — what you want installed
- `~/bmxfile.state.toml` — what `bmx` last applied

Apps live inside **groups**. Lists reference groups and act like install profiles, for example `macos` or `linux`.

## Common Cases

### Start a new config

```bash
bmx init
```

Creates a starter `bmxfile.toml`.

### Add a package

```bash
bmx add brew:age
bmx add brew:custom/formula
bmx add brew-cask:gimp
bmx add brew:age --converge
```

Adds the package to your config and reminds you to run `converge`. With `--converge` (or `-cv`), it immediately runs the same plan and approval workflow as `bmx converge`.

### See what is configured

```bash
bmx ls
bmx ls macos
```

### Apply the desired state

```bash
bmx converge --list macos
```

`bmx` will:

1. read config and state
2. build an install/uninstall plan
3. show the plan
4. ask for approval
5. apply changes
6. update the state file on success

## CLI Usage

```bash
bmx init [--config PATH]
bmx add <manager:package> [--config PATH] [--state PATH] [--group NAME] [--converge|-cv]
bmx rm <manager:package> [--config PATH] [--state PATH] [--group NAME] [--converge|-cv]
bmx remove <manager:package> [--config PATH] [--state PATH] [--group NAME] [--converge|-cv]
bmx ls [list-name] [--config PATH]
bmx converge [--config PATH] [--state PATH] [--list NAME]
bmx version

## Build

Preferred:

```bash
mise run build
```

This creates a snapshot binary in `./dist/bmx`.

Direct Go run during development:

```bash
go run ./cmd/bmx
```

## Package Format

- Homebrew formula: `brew:docker`
- Homebrew cask: `brew-cask:gimp`

Legacy `cask = true` config is not supported. Use the `brew-cask:` prefix instead.

## File Paths

Defaults:

- `~/bmxfile.toml`
- `~/bmxfile.state.toml`

Tests and local runs can override them with flags or environment variables:

- `--config`
- `--state`
- `BMX_CONFIG`
- `BMX_STATE`

Example:

```bash
bmx --config ./testdata/bmxfile.toml --state ./testdata/bmxfile.state.toml converge --list macos
```
