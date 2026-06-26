# ADR 0023: Support --config path

## Status

Accepted

## Context

The default config path is `~/.config/ssh-drop.conf`. During development, testing, demos, and alternate setups, users may need to point at a different config file.

## Decision

V1 supports `--config <path>`.

When omitted, config is loaded from `~/.config/ssh-drop.conf`.

## Consequences

- Tests can use temporary config files without mutating the user's home directory.
- Users can maintain alternate remote sets if needed.
- Missing-config errors should mention the path that was attempted.

