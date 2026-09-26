# ADR 0003: Authenticate Once Per Selected Remote

## Status

Accepted. Supersedes ADR 0001's per-upload password prompt.

## Decision

Remote selection establishes a private OpenSSH control connection. Key and
agent authentication run first in batch mode. An authentication failure that
offers password or keyboard-interactive authentication opens a masked TUI
password prompt; network and host-key errors remain connection errors.

Uploads use the established control socket. They never receive the password
and cannot silently open an interactive fallback connection. A disconnected
master offers reconnection while retaining the file path for retry.

The askpass helper reads a private temporary password file. Both are removed
after authentication, so the password is absent from the long-lived master's
environment. Changing remote or exiting closes the master and removes its
private socket directory. Passwords are not saved in configuration.

## Consequences

Repeated file drops share authentication and return to the focused input.
Remotes still come from `ssh-drop.conf`; OpenSSH resolves configured host
aliases as before. New host keys use OpenSSH's `accept-new` policy; changed
host keys fail connection setup.
