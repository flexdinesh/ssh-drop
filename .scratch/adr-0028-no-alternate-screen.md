# ADR 0028: Do not use the alternate screen

## Status

Accepted

## Context

`ssh-drop` streams `rsync` output, allows normal OpenSSH authentication prompts, and prints a final summary after the TUI exits.

Bubble Tea can run in the alternate screen for a full-screen app experience, but alternate screen behavior can complicate subprocess prompts and preserving useful output in the terminal scrollback.

## Decision

V1 does not use Bubble Tea's alternate screen.

The TUI runs in the normal terminal buffer.

## Consequences

- SSH prompts and command output are easier to reason about.
- The final summary appears naturally in the terminal after quit.
- The UI may feel less like a full-screen app, but better matches a command-oriented file transfer tool.
- Useful status and progress can remain visible in scrollback.

