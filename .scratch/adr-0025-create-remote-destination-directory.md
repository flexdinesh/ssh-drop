# ADR 0025: Create remote destination directory before upload

## Status

Accepted

## Context

The configured `destination` is a remote directory. `/tmp` normally exists, but custom destination directories may not.

## Decision

Before uploading a file, v1 ensures the remote destination directory exists by running `mkdir -p <destination>` over SSH.

If directory creation fails, the transfer attempt fails before invoking `rsync`.

## Consequences

- Custom destination directories work without manual setup.
- Each transfer uses an additional remote SSH command before `rsync`.
- The implementation must quote the destination directory safely for a POSIX remote shell.
- Directory creation failures should be shown in the TUI and counted as failed transfers in the summary.

