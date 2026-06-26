# ADR 0024: Expand tilde and environment variables in config

## Status

Accepted

## Context

Config values may need to reference user-local paths such as SSH identity files, and users may want environment-driven configuration.

## Decision

V1 expands `~` and environment variables in config values.

Examples:

```ini
[remote.files]
host = files.example.com
user = deploy
identity_file = ~/.ssh/files
destination = $SSH_DROP_DEST
```

Expansion is performed by `ssh-drop` on the local machine while loading config.

If a config value references an environment variable that is not set, config loading fails before the TUI opens. The error identifies the missing variable and the config key that referenced it.

## Consequences

- `identity_file = ~/.ssh/key` works as expected.
- Environment variables in config reflect the local environment where `ssh-drop` runs.
- Environment variables in `destination` are expanded locally, not by the remote shell.
- Documentation should warn users that `$HOME` in `destination` means local `$HOME`, not the remote user's home directory.
- Missing environment variables are surfaced early instead of silently expanding to empty strings.
