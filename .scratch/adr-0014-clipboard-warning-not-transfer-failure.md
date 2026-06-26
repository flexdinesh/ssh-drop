# ADR 0014: Clipboard failure is a warning, not transfer failure

## Status

Accepted

## Context

After a successful upload, `ssh-drop` copies the destination path to the local clipboard. Clipboard support depends on local platform tools and can fail independently from the upload.

## Decision

If upload succeeds but clipboard copy fails, the transfer remains successful with a clipboard warning.

The TUI always displays the destination path after upload. On clipboard failure, it also displays warning details so the user can manually copy the path from the screen.

Even when clipboard copy succeeds, the TUI displays the copied destination path as status.

## Consequences

- Summary records need separate upload and clipboard outcomes.
- Users are not misled into thinking the remote upload failed when only clipboard copy failed.
- Clipboard backend failures are visible and actionable.

