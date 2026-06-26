# ADR 0011: Enter submits dropped path input

## Status

Accepted

## Context

Terminal drag-and-drop behavior varies by terminal and operating system. A dropped file may appear as a plain path, a quoted path, an escaped path, a `file://` URL, or text with a trailing newline.

## Decision

V1 uses an editable path input. Dragging a file or pasting a path fills the input, and upload starts only when the user presses Enter.

## Consequences

- The user can inspect and edit the parsed path before upload.
- The tool avoids starting uploads accidentally due to terminal-specific paste behavior.
- The TUI needs clear input focus and an Enter-to-upload hint.
- Validation happens on Enter, before transfer begins.

