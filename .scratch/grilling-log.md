# ssh-drop Grilling Log

Date: 2026-06-26

## Current Goal

Plan the CLI surface area and interactive flow for a Go TUI tool that accepts dragged-and-dropped local files, uploads them to a configured SSH remote with `rsync`, copies the resulting remote path to the local clipboard, and prints a session summary on quit.

## Starting Product Shape

- Interactive TUI starts in a waiting-for-file state.
- User drags and drops a local file path into the terminal.
- Tool asks which configured remote to use unless a destination is supplied by CLI argument.
- Tool uploads the file via `rsync` over SSH.
- Remote destination directory defaults to `/tmp` but is configurable per remote.
- On success, the remote destination path is copied to the local clipboard.
- User can repeat the flow for multiple files in one session.
- User exits with `q` or `ctrl+c`.
- On exit, tool prints a summary of all attempted transfers.
- Supported platforms for now: Linux and macOS.

## Decisions

- 2026-06-26: v1 is interactive-first. `ssh-drop` always opens the TUI; CLI flags may preselect values and skip prompts, but v1 does not provide a separate non-interactive upload mode.
- 2026-06-26: V1 CLI remote selection is named-config only. `--to <remote-name>` may preselect a configured remote and skip selection; ad hoc SSH details and flag-based overrides are out of scope for v1.
- 2026-06-26: Remote selection is session-level. If multiple remotes are configured and `--to` is absent, the TUI opens with a remote picker. Once selected, the remote is sticky until changed. If exactly one remote is configured, it is auto-selected.
- 2026-06-26: From the file drop screen, pressing `r` opens the remote picker and lets the user change the sticky session remote.
- 2026-06-26: Dropped or pasted path input is editable and only submits when the user presses Enter.
- 2026-06-26: V1 accepts plain local paths only. It does not parse shell-escaped paths, quoted paths, or `file://` URLs.
- 2026-06-26: V1 accepts exactly one submitted path per transfer attempt. Multi-path input is out of scope.
- 2026-06-26: Clipboard failure after successful upload is a warning, not transfer failure. The TUI always displays the destination path after upload, whether clipboard copy succeeds or fails.
- 2026-06-26: After transfer completion, the TUI clears the path input, returns to the file drop state, and keeps the last result visible as status.
- 2026-06-26: During upload, v1 streams `rsync` progress/output into the TUI instead of showing only a spinner.
- 2026-06-26: During active upload, `esc` cancels the running `rsync` process, records the attempt as canceled, and returns to the file drop state. V1 does not guarantee remote partial-file cleanup.
- 2026-06-26: During active upload, pressing `q` opens a confirmation prompt to cancel the upload and quit.
- 2026-06-26: On quit, v1 prints a compact summary with success/failure/canceled counts and successful destination paths.
- 2026-06-26: Normal interactive quit exits with status `0`, even if transfer attempts failed or were canceled. Startup/config/dependency errors may exit non-zero before the TUI.
- 2026-06-26: Startup requires `rsync`. Clipboard availability is checked only when copying after successful upload, and failures are warnings.
- 2026-06-26: Missing `~/.config/ssh-drop.conf` exits non-zero before TUI and prints a minimal config sample. The tool does not auto-create config.
- 2026-06-26: V1 supports `--config <path>`; otherwise it loads `~/.config/ssh-drop.conf`.
- 2026-06-26: Config values expand `~` and environment variables locally while loading config. Environment variables in remote `destination` values are local env vars, not remote shell vars.
- 2026-06-26: Missing environment variables referenced by config values are config errors before the TUI opens.
- 2026-06-26: Before upload, v1 runs remote `mkdir -p <destination>` over SSH. If directory creation fails, the transfer attempt fails before `rsync`.
- 2026-06-26: V1 assumes POSIX-like remote targets with POSIX shell, `mkdir -p`, and POSIX path semantics. Non-POSIX SSH targets are out of scope.
- 2026-06-26: V1 lets OpenSSH own authentication prompts. It does not implement custom TUI auth prompts and does not force SSH batch mode by default.
- 2026-06-26: V1 does not use Bubble Tea's alternate screen. The TUI runs in the normal terminal buffer.
- 2026-06-26: Remote picker options show remote name, SSH target, and destination directory. The file drop screen also shows the currently selected remote details.
- 2026-06-26: Remote picker preserves remote order from the config file.
- 2026-06-26: V1 CLI flags are limited to `--config <path>`, `--to <remote-name>`, `--help`, and `--version`.
- 2026-06-26: V1 accepts regular files only. Directories, symlinks, and other non-regular file types are rejected before upload.
- 2026-06-26: V1 overwrites existing files at the computed remote destination. It does not do a pre-upload conflict check.
- 2026-06-26: V1 copies the bare remote destination path, such as `/tmp/file.txt`, to the local clipboard after successful upload.
- 2026-06-26: V1 config uses INI-style syntax at `~/.config/ssh-drop.conf`, with remote sections shaped like `[remote.<name>]`.
- 2026-06-26: Remote config must support both OpenSSH aliases (`host = cb`) and explicit SSH options matching commands like `ssh -i ~/.ssh/files -A deploy@files.example.com`.
- 2026-06-26: V1 does not support raw `ssh = ...` command strings in config; remotes use aliases or structured SSH fields.
- 2026-06-26: V1 config `destination` is an optional remote directory, not a template. It defaults to `/tmp`, and the local file base name is appended automatically.

## Working Assumptions To Challenge

- Drag-and-drop input will arrive as text pasted into the terminal, possibly with shell escaping, quotes, spaces, or `file://` URLs depending on terminal and OS.
- The tool should upload one file path per submitted line first, not multi-file batches, unless we explicitly decide otherwise.
- Clipboard support should use platform commands available on the local machine (`pbcopy`, `xclip`, `wl-copy`, etc.) unless we choose a Go clipboard library with those dependencies behind it.
- Remote configuration should be static file based for v1, with no interactive config editor.
- `rsync` should be invoked as an external command, not reimplemented in Go.
- A remote destination copied to clipboard should be a useful SSH-style string, not only a bare remote path. Example decision needed: `/tmp/foo.txt` vs `host:/tmp/foo.txt` vs `user@host:/tmp/foo.txt`.

## Open Questions

### CLI Surface

1. What should the binary be called: `ssh-drop`, `drop`, or something else?
2. Should non-interactive usage be supported in v1, for example `ssh-drop --to prod ./file.txt`, or should CLI args only preselect the remote while still opening the TUI?
3. Should custom remote details be accepted as flags in addition to named config entries, for example `--host`, `--user`, `--port`, `--dest`?
4. If both CLI flags and config define the same value, which wins?
5. Should the tool support stdin paths, for example `echo ./file.txt | ssh-drop --to prod`, or is that out of scope?

### Remote Selection

6. If there is exactly one configured remote, should the confirmation screen still show?
7. Should the user be able to change the selected remote after seeing the confirmation screen?
8. Should the remote selector show only friendly names, or include host/user/destination preview in the list?
9. Should remotes have a default marker in config?
10. Should the last-used remote be remembered for this session?

### Path And Destination Semantics

11. Should destination templates support only `{file-name}` or richer placeholders such as `{base}`, `{ext}`, `{timestamp}`, `{date}`, `{uuid}`?
12. What should happen if the target path already exists on the remote: overwrite, fail, auto-rename, or ask?
13. Should directory uploads be supported in v1, or files only?
14. Should symlinks be followed, preserved, or rejected?
15. Should destination paths be normalized to POSIX remote paths only?

### Drag And Drop Input

16. Should the TUI accept pasted paths that include surrounding quotes, backslash-escaped spaces, or `file://` URLs?
17. Should the TUI accept multiple paths dropped at once?
18. Should the user press Enter after dropping a path, or should the upload start as soon as a path appears?
19. Should the path input remain editable before upload?
20. Should nonexistent local paths show inline validation before remote selection?

### Clipboard Output

21. What exact string should be copied after success?
22. Should failed uploads leave the previous clipboard alone?
23. Should there be a flag to disable clipboard copying?
24. Should the copied string be configurable with a template?
25. Should copying happen before or after the UI shows success?

### Summary

26. Should the exit summary include successes only, or successes and failures?
27. Should the summary be printed to stdout, stderr, or remain in the TUI alternate screen?
28. Should summary include duration, bytes transferred, remote name, destination path, and clipboard status?
29. Should the tool exit non-zero if any transfer failed?
30. Should a session with no transfers print a summary?

### Config

31. What config format should v1 use: TOML, YAML, JSON, or INI-like `.conf`?
32. Should `~/.config/ssh-drop.conf` be the only config path, or should XDG config paths and `--config` also be supported?
33. Should SSH identity file, port, proxy jump, rsync options, destination template, and copied-output template be supported per remote?
34. Should config support environment variable expansion?
35. Should config validation errors be shown before entering the TUI?

### Failure Modes

36. What should happen when `rsync` is missing?
37. What should happen when no clipboard backend is available?
38. Should `rsync` progress be streamed into the TUI?
39. Should users be able to cancel an in-progress upload?
40. Should the tool retry failed transfers?

## Next Grilling Focus

Narrow the v1 CLI and TUI contract before implementation planning:

- Decide interactive-first vs mixed interactive/non-interactive shape.
- Decide config schema and override rules.
- Decide exact clipboard and destination display semantics.
- Decide batch behavior and validation rules for drag-and-drop input.
- Decide summary and exit-code behavior.
