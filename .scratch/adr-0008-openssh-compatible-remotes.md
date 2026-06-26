# ADR 0008: OpenSSH-compatible remote configuration

## Status

Accepted

## Context

Users connect to SSH machines in different ways. Some use explicit commands, for example:

```sh
ssh -i ~/.ssh/files -A deploy@files.example.com
```

Others use aliases from `~/.ssh/config`, for example:

```sh
ssh cb
```

`ssh-drop` should make these workflows easy without forcing users to duplicate all OpenSSH configuration.

## Decision

V1 remote config supports both SSH aliases and explicit SSH connection fields.

The minimal remote config can be an SSH alias:

```ini
[remote.cb]
host = cb
```

An explicit remote can include user, host, identity file, port, and agent forwarding:

```ini
[remote.files]
host = files.example.com
user = deploy
identity_file = ~/.ssh/files
forward_agent = true
destination = /tmp
```

`ssh-drop` invokes `rsync` over OpenSSH using these fields. When optional SSH fields are absent, OpenSSH config and defaults apply. When `destination` is absent, `ssh-drop` defaults it to `/tmp`.

V1 does not support a raw `ssh = ...` command string in config.

## Consequences

- Users can reuse `~/.ssh/config` aliases.
- Users can also keep all required connection details in `ssh-drop` config.
- Config validation requires `host`; `user`, `port`, `identity_file`, `forward_agent`, and `destination` are optional.
- The generated `rsync` command needs to pass explicit SSH options with `-e ssh ...` when those options are configured.
- The config remains structured and avoids parsing shell command strings.
