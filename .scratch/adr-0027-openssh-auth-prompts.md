# ADR 0027: Let OpenSSH own authentication prompts

## Status

Accepted

## Context

SSH connections may require passphrases, passwords, host key confirmation, or multi-factor prompts. Handling these prompts inside a custom TUI would add complexity and security risk.

## Decision

V1 lets OpenSSH own authentication prompts.

`ssh-drop` does not implement custom TUI prompts for SSH authentication and does not force batch mode by default.

## Consequences

- Users can rely on normal SSH behavior for passphrases, host key confirmation, and other prompts.
- The process runner needs to allow SSH/rsync to interact with the terminal where necessary.
- The TUI should document or handle the possibility that SSH prompt output appears during transfer.
- Environments that require non-interactive auth should configure SSH themselves.

