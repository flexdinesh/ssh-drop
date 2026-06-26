# Glossary

## Askpass

An OpenSSH mechanism where `ssh` asks an external helper program for a password instead of reading directly from the terminal.

## Password Remote

An `ssh-drop` remote with an explicit `user` and no `identity_file`. These remotes are treated as password-auth candidates and prompt inside the TUI before upload.
