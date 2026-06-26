# ADR 0018: Confirm quit during active upload

## Status

Accepted

## Context

`q` is an app-level quit command, but during an active upload quitting requires canceling the running `rsync` process. Because `q` can be pressed accidentally, it should not immediately cancel an upload.

## Decision

During an active upload, pressing `q` opens a confirmation prompt asking whether to cancel the upload and quit.

If the user confirms, `ssh-drop` cancels the running upload, records the attempt as canceled, exits the TUI, and prints the session summary.

If the user declines, the TUI returns to the upload progress view and the upload continues.

`esc` remains the direct "cancel upload and stay in app" command.

## Consequences

- The TUI needs a quit confirmation state during active upload.
- The process runner must continue streaming output while the confirmation is displayed or must safely pause UI output buffering.
- Summary behavior remains consistent: canceled attempts are recorded.

