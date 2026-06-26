# ADR 0005: Overwrite existing remote destination files

## Status

Accepted

## Context

The default destination shape is `/tmp/{file-name}`. Repeatedly dropping the same local file name can target the same remote path. Conflict handling could fail, rename, prompt, or overwrite.

## Decision

V1 overwrites existing remote destination files.

The tool does not perform a separate remote existence check before upload. `rsync` is allowed to replace the destination file at the computed destination path.

## Consequences

- The transfer path is simple and fast.
- Re-dropping the same file updates the same remote path.
- This is destructive if the destination already contains important content.
- The UI should show the computed destination clearly before or during transfer when a remote choice screen is shown, and documentation should state the overwrite behavior.

