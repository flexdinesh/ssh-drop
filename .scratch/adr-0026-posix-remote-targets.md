# ADR 0026: POSIX remote targets

## Status

Accepted

## Context

V1 creates remote destination directories with `mkdir -p` before uploading. This assumes a shell and filesystem behavior on the remote target.

## Decision

V1 supports POSIX-like remote targets.

The remote is expected to provide a POSIX shell, `mkdir -p`, and POSIX path semantics.

## Consequences

- Linux and macOS remotes are in scope.
- Non-POSIX SSH targets are out of scope for v1.
- Remote destination paths are treated as POSIX paths.
- Shell quoting for remote commands can target POSIX shell rules.

