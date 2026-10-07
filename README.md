# nuke

[![CI](https://github.com/charlesonunze/nuke/actions/workflows/ci.yml/badge.svg)](https://github.com/charlesonunze/nuke/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/charlesonunze/nuke)](https://github.com/charlesonunze/nuke/releases/latest)
[![License](https://img.shields.io/github/license/charlesonunze/nuke)](LICENSE)

`nuke` is a native Go CLI for finding and terminating local processes by port,
PID, or exact process name.

It supports macOS and Linux. Windows support is planned.

## Install

### Homebrew (macOS)

```sh
brew install charlesonunze/tap/nuke
```

### Go (macOS and Linux)

```sh
go install github.com/charlesonunze/nuke@latest
```

Go 1.22 or newer is required. Make sure `$(go env GOPATH)/bin` is in your
`PATH`.

## Quick start

```sh
# Terminate the process listening on port 3000
nuke port 3000

# Terminate processes listening on several ports
nuke port 3000 5173 8080

# Find processes by exact name and ask before terminating them
nuke pid -n main

# Preview an operation without sending a signal
nuke port -r 3000-3999 --dry-run
```

## Commands

### Ports

```text
nuke port <port> [port...] [--force] [--dry-run] [--yes]
nuke port -r <start-end> [--force] [--dry-run] [--yes]
nuke port --range <start-end> [--force] [--dry-run] [--yes]
```

Examples:

```sh
nuke port 3000
nuke port 3000 5173 8080
nuke port -r 3000-3999
nuke port 3000 --force
```

Ports must be between `1` and `65535`.

### Processes

```text
nuke pid <pid> [pid...] [--force] [--dry-run] [--yes]
nuke pid -n <name> [--force] [--dry-run] [--yes]
nuke pid --name <name> [--force] [--dry-run] [--yes]
```

Examples:

```sh
nuke pid 12345
nuke pid 12345 67890
nuke pid -n main
nuke pid --name node --yes
```

Name matching uses the exact executable basename and is case-sensitive.

### Version

```sh
nuke -v
nuke version
```

## Safety

- Matching processes are displayed before a signal is sent.
- Graceful termination (`SIGTERM`) is used by default.
- `--force` sends `SIGKILL` instead.
- `--dry-run` shows what would happen without sending a signal.
- `--yes` skips confirmation prompts.
- Name and range selectors require confirmation unless `--yes` is set.
- Explicit targets require confirmation when they resolve to multiple processes.
- PID 1 and the running `nuke` process cannot be terminated by `nuke`.

You may need elevated permissions to terminate a process owned by another user.

Target selectors cannot be combined. These commands are invalid:

```sh
nuke port 3000 -r 4000-5000
nuke pid 1234 -n main
```

## Native platform support

`nuke` has no third-party runtime dependencies.

- Linux reads process and socket information directly from `/proc`.
- macOS uses the system-provided `/bin/ps` and `/usr/sbin/lsof` commands.
