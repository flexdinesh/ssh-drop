# ssh-drop V1 Issue Breakdown

These are local issue drafts for `.scratch/` because this repository does not currently have an issue tracker setup. They are written as vertical slices where possible, in dependency order.

## Proposed Breakdown

1. **Bootstrap interactive CLI with config loading**
   Blocked by: None
   User stories covered: 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 17, 18

2. **Select and display the sticky session remote**
   Blocked by: 1
   User stories covered: 13, 14, 15, 16, 19, 20, 21

3. **Submit and validate one local file path**
   Blocked by: 2
   User stories covered: 22, 23, 24, 25, 26, 27, 28, 29

4. **Upload one file with remote directory creation and rsync progress**
   Blocked by: 3
   User stories covered: 30, 31, 32, 33, 34, 35, 38

5. **Copy destination path and return to the drop loop**
   Blocked by: 4
   User stories covered: 12, 39, 40, 41, 42, 43, 44

6. **Cancel transfers and confirm quit during upload**
   Blocked by: 4
   User stories covered: 36, 37

7. **Print compact session summary and finalize exit behavior**
   Blocked by: 5, 6
   User stories covered: 45, 46, 47

8. **Harden command construction, process seams, and smoke coverage**
   Blocked by: 4, 5, 6, 7
   User stories covered: 48

## Issue 1: Bootstrap interactive CLI with config loading

### What to build

Create the initial Go CLI application for `ssh-drop` with the v1 command surface, config loading, startup dependency checks, and clear startup failures. The app should be runnable, load INI-style remotes, validate `--to`, check for `rsync`, and enter a placeholder TUI state when startup succeeds.

The behavior should support:

- `ssh-drop [--config <path>] [--to <remote-name>]`
- `ssh-drop --help`
- `ssh-drop --version`
- Default config path `~/.config/ssh-drop.conf`
- INI remote sections shaped as `[remote.<name>]`
- Required `host`
- Optional `user`, `port`, `identity_file`, `forward_agent`, and `destination`
- Default `destination = /tmp`
- Local expansion of `~` and environment variables
- Config-order preservation for remotes
- Missing config sample output
- Non-zero startup failures for config errors, invalid `--to`, and missing `rsync`

### Acceptance criteria

- [ ] Running `ssh-drop --help` prints the supported v1 flags.
- [ ] Running `ssh-drop --version` prints a version string.
- [ ] Running with no config exits non-zero and prints a minimal alias config sample plus an explicit SSH-field sample.
- [ ] Running with `--config <path>` loads that file instead of the default path.
- [ ] Config with `[remote.cb] host = cb` is valid and gets destination `/tmp`.
- [ ] Config with explicit `host`, `user`, `identity_file`, `forward_agent`, `port`, and `destination` is valid.
- [ ] Config expands `~` and set environment variables locally.
- [ ] Config referencing a missing environment variable exits non-zero and identifies the variable/key.
- [ ] Unknown `--to <remote-name>` exits non-zero before the TUI starts.
- [ ] Missing `rsync` exits non-zero before the TUI starts.
- [ ] Tests cover config parsing, expansion, remote order, CLI parsing, and startup errors.

### Blocked by

None - can start immediately.

## Issue 2: Select and display the sticky session remote

### What to build

Implement the first real TUI flow for choosing a session remote. When startup succeeds, the TUI should either skip selection or show a remote picker based on config and `--to`. The selected remote becomes sticky for the session and is visible on the drop screen. Pressing `r` from the drop screen reopens the picker and updates the sticky remote.

Remote picker rows should show remote name, SSH target, and destination directory in config order.

### Acceptance criteria

- [ ] With `--to <remote-name>`, the TUI starts on the drop screen with that remote selected.
- [ ] With exactly one configured remote, the TUI starts on the drop screen with that remote selected.
- [ ] With multiple configured remotes and no `--to`, the TUI starts on the remote picker.
- [ ] The picker shows remote name, SSH target, and destination directory.
- [ ] The picker preserves config order.
- [ ] Selecting a remote moves to the drop screen.
- [ ] The drop screen shows the selected remote name, target, and destination directory.
- [ ] Pressing `r` from the drop screen opens the remote picker and allows changing the sticky remote.
- [ ] The Bubble Tea program runs in the normal terminal buffer, not alternate screen.
- [ ] Tests cover startup state selection, picker ordering, sticky remote selection, and `r` behavior.

### Blocked by

Issue 1.

## Issue 3: Submit and validate one local file path

### What to build

Add the editable file path input on the drop screen. The user can drag or paste a plain local path into the input, edit it, and press Enter to submit. V1 accepts exactly one plain path per transfer attempt and only regular files.

This slice should compute the destination path by appending the local file base name to the selected remote destination directory, then show a dry-run style status rather than performing a real upload yet.

### Acceptance criteria

- [ ] The drop screen includes an editable path input.
- [ ] Pressing Enter with an empty input shows validation feedback.
- [ ] Pressing Enter with a nonexistent path shows validation feedback and does not start remote work.
- [ ] Pressing Enter with a directory is rejected.
- [ ] Pressing Enter with a symlink or other non-regular file is rejected.
- [ ] Pressing Enter with a valid regular file creates one transfer attempt in pending/dry-run form.
- [ ] Multiple paths in one submitted input are rejected or treated as invalid plain path input.
- [ ] The computed destination path is `<destination-directory>/<local-base-name>`.
- [ ] The path input remains editable before Enter.
- [ ] Tests cover local file validation, one-path behavior, destination path computation, and validation UI state.

### Blocked by

Issue 2.

## Issue 4: Upload one file with remote directory creation and rsync progress

### What to build

Replace the dry-run transfer with real remote execution. For each valid local file, first run remote `mkdir -p <destination>` over SSH, then run `rsync` over OpenSSH to upload the file to the computed destination path. Stream command output/progress into the TUI while the upload is active.

The command construction must support OpenSSH aliases as well as explicit `user`, `host`, `port`, `identity_file`, and `forward_agent` config fields. Existing remote destination files are overwritten. OpenSSH authentication prompts are allowed to behave normally.

### Acceptance criteria

- [ ] For a valid file, the upload view shows selected remote and computed destination path.
- [ ] Before `rsync`, the app runs remote `mkdir -p <destination>`.
- [ ] If remote directory creation fails, the transfer attempt fails before `rsync` and the TUI shows the failure.
- [ ] If directory creation succeeds, the app runs `rsync` over SSH to the computed destination path.
- [ ] `rsync` progress/output is streamed into the TUI.
- [ ] SSH alias remotes generate a target like `alias:<destination-path>`.
- [ ] Explicit remotes generate a target like `user@host:<destination-path>` when `user` is configured.
- [ ] Identity file, port, and agent forwarding are passed to OpenSSH when configured.
- [ ] Existing destination files are overwritten by normal rsync behavior.
- [ ] Tests cover command construction, POSIX quoting for `mkdir -p`, output streaming messages, success, and failure through fake process runners.

### Blocked by

Issue 3.

## Issue 5: Copy destination path and return to the drop loop

### What to build

After a successful upload, copy the bare destination path to the local clipboard and display the destination path as status. Clipboard failure is a warning, not transfer failure. After completion, clear the input and return to the drop screen with the last result visible.

### Acceptance criteria

- [ ] On successful upload, the app attempts to copy the bare destination path, such as `/tmp/file.txt`, to the local clipboard.
- [ ] On clipboard success, the drop screen shows the copied destination path as the last result.
- [ ] On clipboard failure, the transfer remains successful and the drop screen shows warning details plus the destination path for manual copy.
- [ ] Clipboard support is not required at startup.
- [ ] After success or clipboard warning, the path input is cleared.
- [ ] After success or clipboard warning, the app is ready for the next file without another remote picker.
- [ ] Session state tracks upload result and clipboard result separately.
- [ ] Tests cover clipboard success, clipboard failure warning, input clearing, and last-result status.

### Blocked by

Issue 4.

## Issue 6: Cancel transfers and confirm quit during upload

### What to build

Support cancellation and quit confirmation while an upload is running. Pressing `esc` cancels the running command, records the attempt as canceled, and returns to the drop screen. Pressing `q` during upload asks whether to cancel and quit. Confirming cancels the process, records the canceled attempt, exits the TUI, and proceeds to summary output.

V1 does not guarantee remote partial-file cleanup.

### Acceptance criteria

- [ ] During upload, the UI shows an `esc` cancel hint.
- [ ] Pressing `esc` cancels the active remote command or rsync process.
- [ ] A canceled attempt is recorded distinctly from a failed attempt.
- [ ] After `esc`, the TUI returns to the drop screen and remains usable.
- [ ] During upload, pressing `q` opens a confirmation prompt instead of immediately quitting.
- [ ] Declining the quit confirmation returns to upload progress.
- [ ] Confirming quit cancels the active upload, records the attempt as canceled, and exits the TUI.
- [ ] Tests cover cancellation, canceled state recording, quit confirmation accept/decline, and process cancellation calls.

### Blocked by

Issue 4.

## Issue 7: Print compact session summary and finalize exit behavior

### What to build

Implement final quit behavior and compact summary output. When the user quits normally with `q` or `ctrl+c` outside active upload, the TUI exits and prints success, failure, and canceled counts plus successful destination paths. Normal interactive quit exits with status `0`, even if some attempts failed or were canceled.

### Acceptance criteria

- [ ] Pressing `q` from the drop screen exits the TUI.
- [ ] Pressing `ctrl+c` from the drop screen exits the TUI.
- [ ] After TUI exit, the app prints success count, failure count, canceled count, and successful destination paths.
- [ ] Summary includes successful destination paths even if clipboard copy failed.
- [ ] Summary does not include full per-attempt diagnostics by default.
- [ ] A normal quit exits with status `0`.
- [ ] Startup/config/dependency failures still exit non-zero before TUI.
- [ ] Tests cover summary formatting and exit-code behavior.

### Blocked by

Issues 5 and 6.

## Issue 8: Harden command construction, process seams, and smoke coverage

### What to build

Harden the implementation after the core flow is complete. Focus on test seams, command safety, process behavior, and smoke coverage rather than adding new product features.

This issue should verify that the implementation has stable interfaces for config loading, TUI model updates, remote directory creation, transfer execution, clipboard copying, and summary rendering. It should also add smoke tests for the binary behaviors that do not require real SSH.

### Acceptance criteria

- [ ] There is a clear process runner seam for SSH directory creation and rsync upload.
- [ ] There is a clear clipboard seam for platform clipboard implementations.
- [ ] TUI tests do not require real SSH, real rsync transfers, or real clipboard access.
- [ ] Command construction tests cover aliases, explicit user/host, identity file, port, forward agent, and destination paths with spaces.
- [ ] POSIX shell quoting tests cover remote destination directory creation.
- [ ] Smoke tests cover `--help`, `--version`, missing config, invalid config, and invalid `--to`.
- [ ] Manual verification notes describe how to test a real SSH alias and an explicit identity-file remote.
- [ ] No new v1 scope is added while hardening.

### Blocked by

Issues 4, 5, 6, and 7.

