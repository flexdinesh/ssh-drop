# ADR 0015: Return to drop input after transfer completion

## Status

Accepted

## Context

The main session loop is repeated file uploads to the currently selected remote. After each transfer, the user needs both confirmation of what happened and a ready state for the next file.

## Decision

After a transfer completes, the TUI automatically returns to the file drop input state.

The path input is cleared. The last transfer result remains visible as status, including the destination path and any clipboard warning.

## Consequences

- Repeated uploads stay fast.
- The user can inspect the last copied path without dismissing a separate result screen.
- The summary retains the full history beyond the single visible last-result status.

