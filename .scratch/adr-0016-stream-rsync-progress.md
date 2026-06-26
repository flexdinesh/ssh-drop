# ADR 0016: Stream rsync progress in the TUI

## Status

Accepted

## Context

Uploads may take long enough that a spinner alone does not provide enough confidence. `rsync` can report transfer progress and useful status while it runs.

## Decision

V1 streams `rsync` progress/output into the TUI during upload.

The upload view shows the selected remote, computed destination path, and live `rsync` output/progress. If `rsync` fails, the failure details are shown in the TUI and recorded in the session summary.

## Consequences

- Users can see transfer progress for large files.
- The implementation needs a command runner that streams stdout/stderr into Bubble Tea messages.
- The UI needs to keep progress readable without letting output permanently break the layout.
- Tests should cover command output handling through an abstraction rather than invoking real remote transfers.

