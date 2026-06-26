# ADR 0020: Normal interactive quit exits zero

## Status

Accepted

## Context

`ssh-drop` v1 is interactive-first, not a scripting interface. A session may include failed or canceled transfer attempts, but the user can still quit the app normally after handling those outcomes in the TUI.

## Decision

Normal TUI quit exits with status code `0`, even if some transfer attempts failed or were canceled.

Startup errors, config parse errors, invalid `--to` values, and missing required external dependencies can exit non-zero before the main TUI session begins.

## Consequences

- Exit codes reflect whether the app session itself completed normally, not whether every transfer succeeded.
- The compact session summary carries transfer outcome information for the user.
- A future non-interactive mode can define stricter exit-code semantics separately.

