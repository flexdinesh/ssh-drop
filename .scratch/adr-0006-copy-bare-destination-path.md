# ADR 0006: Copy bare destination path

## Status

Accepted

## Context

After a successful upload, `ssh-drop` copies a string to the local clipboard. The copied value could include the remote identity, but the expected workflow favors quickly pasting the path on or about the target machine.

## Decision

V1 copies the bare remote destination path to the local clipboard.

Example:

```text
/tmp/file.txt
```

It does not copy an `rsync`-style reference such as `user@host:/tmp/file.txt` and does not support per-remote copy templates in v1.

## Consequences

- The copied value is short and directly usable in remote shell contexts.
- The copied value is ambiguous outside the remote context.
- The TUI and session summary should display the remote name alongside the destination path so users can recover the full context.
