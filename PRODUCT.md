# Product

<!-- impeccable:product-schema 1 -->

## Platform

terminal (Go CLI/TUI)

## Users

People working in a local terminal who need to show a screenshot from their machine to an AI harness in a remote SSH session.

## Product Purpose

SSH Drop sends a local screenshot to a remote machine over SSH, then copies the remote file path to the local clipboard so the user can paste it into the remote workflow. It also accepts other regular local files.

## Positioning

A terminal-first screenshot handoff: drop or paste a local file path, transfer it to a configured SSH remote, and get the destination path on the clipboard in one interactive session.

## Operating Context

- The user runs `ssh-drop` locally and selects a configured remote, or starts with `--to <name>`. SSH authentication happens before the drop screen.
- The user drops or pastes a local file path into the TUI. Transfers use SSH and `rsync`.
- Remotes come from an INI-style config file. A host may be an OpenSSH alias.
- After a successful transfer, the TUI reports the source and destination paths and attempts to copy the destination path to the clipboard.
- The input clears and stays ready for another drop. Repeated uploads reuse the selected remote's authenticated SSH connection.

## Capabilities and Constraints

- Transfers one regular local file at a time to a configured remote destination. The default destination directory is `/tmp/ssh-drop/`.
- Tries SSH key and agent authentication first; opens a masked password prompt only when authentication requires it. Supports identity files, agent forwarding, and ports.
- Requires `rsync` on the local `PATH`. Clipboard copy uses an available `pbcopy`, `wl-copy`, or `xclip` backend.
- Shows transfer progress and outcomes; users can cancel an upload.
- Remotes remain defined in `ssh-drop.conf`. OpenSSH can resolve their configured host aliases; the TUI does not discover hosts from `~/.ssh/config`.

## Brand Commitments

The product name is SSH Drop. Its command is `ssh-drop`.

## Evidence on Hand

- [README.md](README.md) documents the core workflow, install paths, usage, config, and a demo video link.
- [docs/adr/0003-session-authentication.md](docs/adr/0003-session-authentication.md) records authentication and connection reuse.
- No testimonials, benchmarks, or customer claims are present in the repository.

## Product Principles

- Keep the screenshot handoff quick within a terminal workflow.
- Make the remote destination path immediately usable after transfer.
- Work with the user's configured SSH remotes and authentication.
- Show clear transfer outcomes, including clipboard failures.
