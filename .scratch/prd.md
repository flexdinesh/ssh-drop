# ssh-drop V1 PRD

## Problem Statement

Moving a local file to a remote SSH machine is a repetitive workflow with too many small steps: remember the right SSH host or alias, build the right `rsync` command, choose the right remote destination, wait for transfer completion, and then copy the remote path for later use.

The user wants an interactive terminal tool that makes this workflow fast and repeatable. They should be able to start one command, choose a configured remote once, drag or paste a file path, upload it to the remote, get the resulting destination path copied to their local clipboard, and repeat the flow for more files without rebuilding SSH or `rsync` commands by hand.

## Solution

Build `ssh-drop`, an interactive Go CLI using Bubble Tea, Lip Gloss, and Bubbles.

`ssh-drop` starts an interactive TUI in the normal terminal buffer. It loads named remotes from an INI-style config file, lets the user select a sticky remote for the session, accepts one editable plain local file path at a time, uploads the regular file to the selected POSIX-like remote using `rsync` over OpenSSH, copies the bare remote destination path to the local clipboard, displays the destination path as status, and prints a compact summary when the user quits.

The v1 CLI is interactive-first. It is not a scripting interface. Flags only select config and optionally preselect a named remote.

## User Stories

1. As a user, I want to run `ssh-drop`, so that I can start a focused file-drop transfer session.
2. As a user, I want `ssh-drop` to load remotes from `~/.config/ssh-drop.conf`, so that I do not need to type SSH details every time.
3. As a user, I want to pass `--config <path>`, so that I can use alternate config files for testing, demos, or separate environments.
4. As a user, I want config to be INI-style, so that the `.conf` file is easy to edit by hand.
5. As a user, I want to configure a remote with only an SSH alias, so that I can reuse my existing `~/.ssh/config`.
6. As a user, I want to configure a remote with explicit SSH fields, so that I can model commands like `ssh -i ~/.ssh/files -A deploy@files.example.com`.
7. As a user, I want `identity_file = ~/.ssh/key` to work, so that local SSH paths are ergonomic.
8. As a user, I want config values to expand environment variables, so that I can keep environment-specific values outside the config file.
9. As a user, I want missing config environment variables to fail clearly, so that a bad destination is not silently created.
10. As a user, I want missing config to print a useful sample, so that I can create a valid first config quickly.
11. As a user, I want `rsync` to be checked at startup, so that I learn immediately if the core transfer dependency is missing.
12. As a user, I want clipboard support to be checked only when needed, so that missing clipboard tooling does not block uploads.
13. As a user with one configured remote, I want it selected automatically, so that the common case is fast.
14. As a user with multiple configured remotes, I want the TUI to open with a remote picker, so that I choose the session destination before dropping files.
15. As a user, I want remote picker rows to show name, SSH target, and destination directory, so that I can distinguish similar remotes.
16. As a user, I want remotes displayed in config order, so that I can put frequently used remotes first.
17. As a user, I want `--to <remote-name>` to preselect a configured remote, so that I can skip the picker for a known session.
18. As a user, I want unknown `--to` values to fail before the TUI opens, so that mistakes are caught early.
19. As a user, I want the selected remote to be sticky for the session, so that repeated uploads do not require repeated selection.
20. As a user, I want to press `r` from the drop screen to change remotes, so that I can switch session destination without restarting.
21. As a user, I want the current remote details visible on the drop screen, so that I know where the next file will go.
22. As a user, I want to drag or paste a file path into an editable input, so that terminal drag-and-drop remains visible before upload.
23. As a user, I want upload to start only when I press Enter, so that I can inspect or edit the path first.
24. As a user, I want v1 to accept one plain local path at a time, so that the transfer behavior is predictable.
25. As a user, I want invalid paths to fail before remote operations, so that mistakes do not produce SSH or rsync noise.
26. As a user, I want directories and symlinks rejected in v1, so that "drop a file" has literal behavior.
27. As a user, I want the configured destination to be a directory, so that `destination = /tmp` is enough to understand.
28. As a user, I want destination to default to `/tmp`, so that minimal alias config works.
29. As a user, I want the local file base name appended to the destination directory, so that the remote filename matches the local filename.
30. As a user, I want existing remote files overwritten, so that re-dropping a file updates the same remote path.
31. As a user, I want `ssh-drop` to create the remote destination directory, so that custom destinations work without manual setup.
32. As a user, I want POSIX-like remotes supported in v1, so that Linux and macOS SSH targets work with `mkdir -p` and POSIX paths.
33. As a user, I want OpenSSH authentication prompts to behave normally, so that host key confirmation, passphrases, passwords, and 2FA remain SSH-owned.
34. As a user, I want the TUI to run in the normal terminal buffer, so that SSH prompts, progress, and final summary remain visible in scrollback.
35. As a user, I want live `rsync` progress/output shown during upload, so that I can tell what is happening for large files.
36. As a user, I want to press `esc` during upload to cancel the transfer and stay in the app, so that I can recover from a mistaken or slow upload.
37. As a user, I want `q` during upload to ask before canceling and quitting, so that an accidental keypress does not stop a transfer.
38. As a user, I want upload failures shown in the TUI, so that I know what went wrong before trying another file.
39. As a user, I want successful uploads to copy the bare destination path to my local clipboard, so that I can paste it where I need it.
40. As a user, I want the destination path displayed even when clipboard copy succeeds, so that I can see exactly what was copied.
41. As a user, I want clipboard failure to be a warning, so that a successful upload is not reported as failed.
42. As a user, I want clipboard failure details and the destination path shown on screen, so that I can manually copy the path.
43. As a user, I want the input cleared after each completed attempt, so that I can immediately drop the next file.
44. As a user, I want the last transfer result visible after returning to the drop screen, so that I can verify the last destination.
45. As a user, I want `q` or `ctrl+c` to quit the session, so that normal terminal quit behavior works.
46. As a user, I want a compact summary on quit, so that I can quickly see counts and successful destination paths.
47. As a user, I want normal interactive quit to exit with status `0`, so that handled failures or cancellations do not make the app feel crashed.
48. As a future maintainer, I want transfer and clipboard effects behind testable boundaries, so that the TUI can be tested without real SSH targets.

## Implementation Decisions

- The binary is `ssh-drop`.
- V1 is interactive-first. The command always opens the TUI except for startup failures, `--help`, and `--version`.
- V1 CLI surface is:

```text
ssh-drop [--config <path>] [--to <remote-name>]
ssh-drop --help
ssh-drop --version
```

- File path arguments, stdin paths, ad hoc remote flags, raw SSH command config, destination overrides, and debug flags are out of scope for v1.
- Default config path is `~/.config/ssh-drop.conf`; `--config <path>` overrides it.
- Config is INI-style, with remote sections shaped as `[remote.<name>]`.
- Remote config requires `host`. Optional keys include `user`, `port`, `identity_file`, `forward_agent`, and `destination`.
- `host` may be an OpenSSH alias or a real hostname.
- `destination` is a remote directory, not a template. It defaults to `/tmp`.
- The destination path is computed by appending the local file base name to the destination directory.
- Config values expand `~` and environment variables locally during config loading.
- Missing environment variables referenced by config are config errors.
- Remote section order in config is preserved and used by the picker.
- Missing config exits non-zero before TUI and prints sample config.
- Missing `rsync` exits non-zero before TUI.
- Clipboard backend availability is checked only after successful upload.
- If `--to` is passed, it must match a configured remote and the picker is skipped.
- If exactly one remote is configured, it is auto-selected and the picker is skipped.
- If multiple remotes are configured and `--to` is absent, the TUI opens with the remote picker.
- Remote selection is sticky for the session.
- Pressing `r` from the drop screen opens the remote picker and updates the sticky remote.
- Remote picker rows show remote name, SSH target, and destination directory.
- The drop screen shows the currently selected remote name, target, and destination directory.
- The TUI uses normal terminal buffer mode, not Bubble Tea alternate screen.
- Path input is editable and submitted with Enter.
- V1 accepts plain local paths only.
- V1 accepts exactly one submitted path per transfer attempt.
- V1 accepts regular files only. Directories, symlinks, sockets, devices, and other non-regular file types are rejected before upload.
- Before upload, `ssh-drop` runs remote `mkdir -p <destination>` over SSH.
- Remote targets are POSIX-like systems with a POSIX shell, `mkdir -p`, and POSIX path semantics.
- If remote directory creation fails, the transfer attempt fails before `rsync`.
- Upload uses `rsync` over OpenSSH.
- OpenSSH owns authentication prompts. V1 does not implement custom TUI auth prompts and does not force batch mode by default.
- Existing files at the computed remote destination are overwritten.
- During upload, the TUI streams `rsync` progress/output and shows selected remote plus computed destination path.
- During upload, `esc` cancels the running transfer and returns to the drop screen.
- During upload, `q` opens a confirmation prompt to cancel the upload and quit.
- V1 does not guarantee cleanup of remote partial files after cancellation.
- After completion, the TUI returns to the drop state, clears the input, and keeps last result status visible.
- On upload success, the bare destination path is copied to the local clipboard.
- Clipboard failure after upload success is a warning, not transfer failure.
- The TUI always displays the destination path after upload, whether clipboard copy succeeds or fails.
- On quit, the app prints a compact summary: success count, failure count, canceled count, and successful destination paths.
- Normal interactive quit exits `0`; startup/config/dependency errors may exit non-zero before TUI.

## Testing Decisions

- Tests should verify external behavior and state transitions, not Lip Gloss styling details or exact command implementation internals unless the command is a public behavior boundary.
- The highest-value seam is an application session/controller boundary that accepts config, a selected remote, local path input, and injected interfaces for transfer, remote directory creation, clipboard, and dependency checks.
- Config loading should be tested with real INI text and temporary files, covering aliases, explicit SSH fields, destination defaults, `~` expansion, environment expansion, missing env vars, missing config, invalid sections, and preserved remote order.
- CLI parsing should be tested as a small contract: supported flags, unknown flags, invalid `--to`, default config path, and `--config`.
- TUI model tests should cover remote picker startup rules, sticky remote changes, path submission, validation failures, transfer completion, clipboard warning display, cancellation, quit confirmation, and summary data.
- Command-building tests should cover SSH alias targets, explicit `user@host` targets, identity file, port, agent forwarding, remote `mkdir -p`, destination path construction, and POSIX shell quoting for remote paths.
- Transfer runner tests should use fake command processes or a small process abstraction to simulate stdout/stderr streaming, success, failure, and cancellation.
- Clipboard tests should use an injected clipboard interface to simulate success and failure without depending on host clipboard tools.
- Integration smoke tests can run the compiled binary with `--help`, `--version`, missing config, bad config, and missing/invalid `--to` without requiring real SSH access.
- Real remote SSH transfer tests are out of scope for normal automated tests; they may be documented as manual verification.

## Out of Scope

- True non-interactive uploads such as `ssh-drop --to prod ./file.txt`.
- File path arguments and stdin path input.
- Ad hoc remote flags such as `--host`, `--user`, `--port`, or `--dest`.
- Raw `ssh = ...` command strings in config.
- Destination path templates.
- Multiple files in one submitted input.
- Shell-escaped, quoted, or `file://` path parsing.
- Directory uploads.
- Symlink uploads or preservation behavior.
- Conflict detection, fail-on-existing, auto-renaming, or overwrite prompts.
- Non-POSIX remote targets.
- Custom TUI prompts for SSH auth.
- Clipboard output templates.
- A config editor or auto-created config file.
- Alternate screen mode.
- Debug flag.
- Automatic remote partial-file cleanup after cancellation.
- Packaging and distribution beyond a runnable Go CLI.

## Further Notes

- `rsync` flags and exact clipboard backend order still need implementation-level decisions.
- Because OpenSSH prompts are allowed, the process runner must be designed carefully with Bubble Tea so command IO remains usable in the normal terminal buffer.
- The issue breakdown should preserve vertical slices: each slice should keep the CLI runnable and testable rather than building all config, then all UI, then all transfer logic in separate horizontal passes.
