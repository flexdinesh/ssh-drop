# ADR 0017: Escape cancels active upload and keeps session open

## Status

Accepted

## Context

V1 streams `rsync` progress in the TUI. Long or mistaken transfers need an explicit cancellation path that does not require quitting the whole app.

## Decision

During an active upload, pressing `esc` cancels the running `rsync` process and returns the TUI to the file drop state.

The canceled transfer attempt is recorded in the session summary with its local path, remote, computed destination, and canceled status.

`ctrl+c` and `q` remain app-level quit commands where appropriate.

## Consequences

- The command runner must support cancellation.
- The TUI needs to distinguish canceled transfers from rsync failures.
- The remote may contain a partial file depending on when cancellation occurs; v1 does not guarantee remote cleanup.
- The upload view needs a visible `esc cancel` hint.

