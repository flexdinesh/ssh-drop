# ADR 0003: Auto-select the only configured remote

## Status

Accepted

## Context

Each transfer needs a configured remote. When multiple remotes exist, the user needs a selection or confirmation step. When exactly one remote exists, a confirmation prompt adds friction to the main drag-and-drop loop.

## Decision

If there is exactly one configured remote and the user did not pass `--to`, v1 auto-selects that remote and does not show the remote selection screen.

If multiple remotes exist and `--to` was not provided, the TUI starts with a remote picker. The selected remote is sticky for the session.

## Consequences

- Single-remote users get the fastest path: start TUI, drop file, upload.
- Multi-remote users choose intentionally at session start unless they started with `--to`.
- The upload state should clearly display which remote and destination are being used because per-transfer confirmation is skipped.
