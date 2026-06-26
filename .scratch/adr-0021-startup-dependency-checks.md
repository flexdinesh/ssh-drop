# ADR 0021: Require rsync at startup, check clipboard on demand

## Status

Accepted

## Context

`ssh-drop` relies on external tools. `rsync` is required for the core upload behavior. Clipboard support is useful after successful upload, but upload can still succeed without it because the TUI displays the destination path.

## Decision

At startup, v1 requires `rsync` to be available. If `rsync` is missing, the app exits before opening the TUI with a non-zero status and a clear error.

Clipboard backends are not required at startup. Clipboard copy is attempted after successful upload; if it fails, the transfer remains successful with a warning.

## Consequences

- The TUI does not open when the core transfer capability is unavailable.
- Users without clipboard tools can still upload files and manually copy the visible destination path.
- Clipboard errors follow the same warning path whether caused by missing tools or runtime failure.

