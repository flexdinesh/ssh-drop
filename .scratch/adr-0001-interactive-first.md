# ADR 0001: Interactive-first CLI shape

## Status

Accepted

## Context

The core product promise is a low-friction terminal workflow: start `ssh-drop`, drag a file into the TUI, choose or confirm a remote, upload with `rsync`, copy the resulting remote reference, and repeat.

The tool can also accept custom arguments so users can bypass the remote confirmation prompt.

## Decision

`ssh-drop` v1 is interactive-first.

The command always opens the TUI. CLI flags may preselect a remote, destination, or related transfer settings, and those flags may skip specific prompts. They do not turn v1 into a separate non-interactive scripting interface.

## Consequences

This keeps v1 focused on the drag-and-drop transfer loop and avoids designing a second non-interactive UX contract at the same time.

Consequences:

- The primary command is `ssh-drop`.
- File path input is primarily collected inside the TUI.
- CLI arguments act as defaults or prompt bypasses for the TUI.
- Exit-code behavior can remain session-oriented instead of script-oriented.
- A future non-interactive mode can be added deliberately without constraining v1.
