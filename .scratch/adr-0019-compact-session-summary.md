# ADR 0019: Compact session summary

## Status

Accepted

## Context

At quit, `ssh-drop` exits the TUI and prints a summary of work completed during the session. Transfer attempts can succeed, fail, or be canceled. Successful uploads can also have clipboard warnings.

## Decision

V1 prints a compact session summary.

The summary includes:

- Count of successful transfers.
- Count of failed transfers.
- Count of canceled transfers.
- List of successful destination paths.

The default summary does not print every attempt with full local path, remote name, clipboard status, and error details.

## Consequences

- The quit output stays short and easy to scan.
- The TUI must show actionable failure details at the time of failure because the compact summary will not include full diagnostics by default.
- Internal session state still tracks enough detail to compute counts and list successful destination paths.

