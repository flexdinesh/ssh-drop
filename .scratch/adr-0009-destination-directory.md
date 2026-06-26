# ADR 0009: Destination is a directory

## Status

Accepted

## Context

The initial product description used `/tmp/{file-name}` as the default destination shape. Config could expose that as a template, but v1 only supports regular files and always preserves the local file name.

## Decision

V1 config uses `destination` as a remote directory, not a path template.

Example:

```ini
[remote.cb]
host = cb
destination = /tmp
```

If the user drops `report.pdf`, the computed destination path is:

```text
/tmp/report.pdf
```

`destination` is optional. When omitted, it defaults to `/tmp`.

## Consequences

- Users do not need to learn template syntax in v1.
- The uploaded remote file name always matches the local base name.
- Destination validation applies only when a value is configured.
- A minimal SSH alias remote can be as small as `host = cb`.
- Future versions can add templating under a separate key if needed.
