# ADR 0001: Prompt For SSH Passwords In The TUI

## Status

Accepted

## Context

When a configured remote has an explicit `user` but no `identity_file`, OpenSSH may prompt for a password through the terminal. That prompt bypasses Bubble Tea's input model, produces a poor TUI experience, and can fail to register the entered password.

## Decision

For remotes with `user` set and `identity_file` unset, `ssh-drop` prompts for the SSH password inside a centered TUI popup before starting the upload.

The transfer runner passes that password only to the active `ssh` and `rsync` subprocesses using `SSH_ASKPASS`, `SSH_ASKPASS_REQUIRE=force`, and a short-lived askpass helper script. The helper is removed after the transfer finishes.

## Consequences

Password entry stays inside the TUI and is masked in rendered output.

The password is transfer-scoped and is not stored in the config.

The subprocess environment contains the password while the transfer is running, so SSH keys or agent auth remain preferred where available.
