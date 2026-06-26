# ADR 0010: Choose sticky remote at session start

## Status

Accepted

## Context

The original flow selected or confirmed the remote after each dropped file. That makes every transfer explicit, but interrupts repeated uploads.

The revised flow treats the selected remote as session state.

## Decision

When `ssh-drop` opens in interactive mode:

- If `--to <remote-name>` is provided, that remote is selected and the TUI starts at the file drop screen.
- If exactly one remote is configured, that remote is auto-selected and the TUI starts at the file drop screen.
- If multiple remotes are configured and `--to` is not provided, the TUI starts with a remote picker.

After a remote is selected, it is sticky for the session. Subsequent dropped files use the same remote by default.

The TUI should provide a way to change the selected remote during the session.

From the file drop screen, pressing `r` opens the remote picker. Selecting a different remote updates the sticky session remote and returns to the file drop screen.

## Consequences

- Multi-file sessions are faster because remote selection does not interrupt every upload.
- The current remote must remain visible in the file drop and transfer states.
- The UI needs a clear footer hint for `r` to change remotes.
- Summary entries should include the remote used for each transfer because the session remote may change.
