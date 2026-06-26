# Glossary

## Askpass

An OpenSSH mechanism where `ssh` asks an external helper program for a password instead of reading directly from the terminal.

## Password Remote

An `ssh-drop` remote with an explicit `user` and no `identity_file`. These remotes are treated as password-auth candidates and prompt inside the TUI before upload.

## Homebrew Tap

A Git repository that Homebrew can read formulae and casks from. `ssh-drop` uses the `flexdinesh/homebrew-tap` tap for stable Homebrew installs.

## Stable Release

A SemVer Git tag on `main`, such as `v0.1.0`, that GoReleaser turns into GitHub Release artifacts and a Homebrew cask update.
