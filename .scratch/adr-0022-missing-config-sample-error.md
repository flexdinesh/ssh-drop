# ADR 0022: Missing config exits with sample config

## Status

Accepted

## Context

V1 does not support ad hoc remotes. At least one configured remote is required before the tool can upload files.

## Decision

If `~/.config/ssh-drop.conf` is missing, `ssh-drop` exits before opening the TUI with a non-zero status.

The error output includes a minimal copy-pasteable config sample.

Example:

```ini
[remote.cb]
host = cb
```

And an explicit SSH example:

```ini
[remote.files]
host = files.example.com
user = deploy
identity_file = ~/.ssh/files
forward_agent = true
destination = /tmp
```

## Consequences

- The TUI does not open in an unusable state.
- The app does not write files into the user's config directory without explicit action.
- First-run setup is guided by error output.
