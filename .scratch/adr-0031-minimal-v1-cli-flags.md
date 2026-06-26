# ADR 0031: Minimal v1 CLI flags

## Status

Accepted

## Context

`ssh-drop` v1 is interactive-first. Command-line flags should configure or skip parts of the interactive flow without creating a second interface.

## Decision

V1 supports this CLI surface:

```text
ssh-drop [--config <path>] [--to <remote-name>]
ssh-drop --help
ssh-drop --version
```

No `--debug`, ad hoc remote flags, destination overrides, or non-interactive file arguments are included in v1.

## Consequences

- The CLI contract is easy to learn.
- Config remains the owner of remote details.
- Future flags can be added deliberately after observing use.

