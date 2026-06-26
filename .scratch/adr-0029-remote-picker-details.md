# ADR 0029: Remote picker shows target and destination

## Status

Accepted

## Context

Remote selection is sticky for the session. Choosing the wrong remote can send multiple files to the wrong machine until changed.

## Decision

Remote picker options show:

- Remote name.
- SSH target.
- Destination directory.

Example:

```text
files  deploy@files.example.com  -> /tmp
cb     cb                        -> /tmp
```

The file drop screen also shows the currently selected remote with its target and destination.

## Consequences

- Users can distinguish similarly named remotes.
- The sticky remote remains visible during repeated uploads.
- The UI needs formatting that handles long hosts and paths without becoming unreadable.
