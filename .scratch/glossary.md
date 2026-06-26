# ssh-drop Glossary

## Local File

A path on the user's current machine that the user wants to upload. It may be typed, pasted, or inserted by dragging a file into the terminal.

For v1, a local file must be a regular file. Directories, symlinks, sockets, devices, and other file types are not valid transfer inputs.

## Dropped Path

The raw text inserted into the TUI by terminal drag-and-drop or paste. It may require parsing before it becomes a local file path.

## Remote

A named SSH destination from config. A remote includes enough SSH and transfer settings for `rsync` to upload a local file without additional prompts from `ssh-drop`.

A remote may point at an OpenSSH alias or define explicit SSH fields such as user, host, port, identity file, and agent forwarding.

## Remote Selector

The TUI state where the user confirms or chooses which configured remote should receive the current file.

## Destination Directory

A configured remote directory where uploaded files are placed. The default is `/tmp`.

## Destination Path

The final absolute path on the remote machine where the uploaded file lands.

## Copied Output

The string copied to the local machine's clipboard after a successful upload. This may be the destination path or a richer reference such as `user@host:/tmp/file.txt`.

## Transfer Attempt

One attempt to upload one local file to one remote destination. A transfer attempt can succeed, fail, or be canceled.

Upload result and clipboard result are tracked separately. A successful upload can still have a clipboard warning.

## Session Summary

The report printed when the user quits the TUI. It describes the transfer attempts made during the current run.
