# ADR 0012: Plain path input only

## Status

Accepted

## Context

Different terminals insert dragged files in different formats. Some insert plain paths, while others insert quoted paths, escaped spaces, or `file://` URLs.

## Decision

V1 accepts plain local paths only.

The TUI provides an editable input, so users can manually adjust dragged or pasted text before pressing Enter.

## Consequences

- Input parsing remains simple and predictable.
- The tool does not attempt shell-like parsing in v1.
- Drag-and-drop behavior may require manual editing in terminals that insert escaped or URL-style paths.
- Future versions can add tolerant parsing once real terminal behavior has been observed.

