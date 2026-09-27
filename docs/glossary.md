# Glossary

## Askpass

An OpenSSH mechanism where `ssh` asks an external helper program for a password instead of reading directly from the terminal.

## Session Authentication

Selecting a remote establishes an SSH connection using keys or an agent first. If authentication requires a password, the TUI opens a masked prompt. Repeated uploads reuse that connection without receiving the password.

## Homebrew Tap

A Git repository that Homebrew can read formulae and casks from. `ssh-drop` uses the `flexdinesh/homebrew-tap` tap for stable Homebrew installs.

## Stable Release

A SemVer Git tag on `main`, such as `v0.1.0`, that GoReleaser turns into GitHub Release artifacts and a Homebrew cask update.
