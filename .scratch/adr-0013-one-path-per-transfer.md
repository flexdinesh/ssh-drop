# ADR 0013: One path per transfer

## Status

Accepted

## Context

V1 accepts plain local paths only and starts upload when the user presses Enter. Supporting multiple paths in one input would require additional parsing rules and batching UI.

## Decision

V1 accepts exactly one submitted path per transfer attempt.

Multiple pasted paths, newline-separated batches, and multi-file drops are out of scope.

## Consequences

- Validation and error messages can focus on one local file.
- Each transfer attempt maps directly to one summary row.
- Multi-file workflows are handled by repeating the drop/paste and Enter flow.

