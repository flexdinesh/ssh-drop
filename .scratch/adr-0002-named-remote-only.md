# ADR 0002: Named remote CLI selection only

## Status

Accepted

## Context

`ssh-drop` v1 is interactive-first. The user wants CLI arguments that can avoid repeatedly confirming the destination remote, but the product premise is that SSH details are made easy through predefined configuration.

## Decision

V1 supports named remote selection only.

The CLI may accept a flag such as `--to <remote-name>` to preselect a remote from config and skip the remote selection prompt. V1 does not support full ad hoc SSH destinations through flags such as `--host`, `--user`, or `--port`, and does not support per-invocation remote overrides such as `--dest`.

## Consequences

- Remote definitions live in config.
- CLI parsing remains small and predictable.
- TUI state can treat the selected remote as either unset or a fully validated configured remote.
- Validation errors for unknown `--to` values are shown before entering the main transfer loop.
- A future version can add overrides or ad hoc remotes if there is real demand.

