# ADR 0004: Regular files only in v1

## Status

Accepted

## Context

The core v1 workflow is "drag and drop a file." `rsync` can upload directories and preserve symlinks, but those behaviors introduce destination semantics that are not necessary for the first version.

## Decision

V1 accepts one local regular file per transfer attempt.

Directories, symlinks, sockets, devices, and other non-regular file types are rejected before remote selection or upload.

## Consequences

- The destination template can safely assume a file name.
- The transfer command can avoid recursive directory flags.
- UI validation can give direct feedback before any remote operation.
- Directory and symlink support can be designed later with explicit destination semantics.

