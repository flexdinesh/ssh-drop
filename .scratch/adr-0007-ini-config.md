# ADR 0007: INI-style config file

## Status

Accepted

## Context

The global config path is `~/.config/ssh-drop.conf`. The file needs to define multiple named SSH remotes and per-remote destination settings.

## Decision

V1 uses an INI-style config syntax at `~/.config/ssh-drop.conf`.

Remote sections use the shape `[remote.<name>]`.

Example:

```ini
[remote.dev]
host = dev.example.com
user = dinesh
port = 22
destination = /tmp/{file-name}
```

## Consequences

- The file matches user expectations for a `.conf` path.
- The schema should stay shallow and explicit.
- Repeated sections and nested config should be avoided in v1.
- Config validation should report section and key names clearly.

