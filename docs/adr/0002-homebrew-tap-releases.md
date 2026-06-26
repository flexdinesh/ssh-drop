# ADR 0002: Publish Stable Releases Through The Homebrew Tap

## Status

Accepted

## Context

`ssh-drop` is a terminal CLI, and users should be able to install stable
versions without installing Go. The repository already uses Go-compatible SemVer
release tags in the README, but it had no release automation or Homebrew formula.

There is an existing `flexdinesh/homebrew-tap` repository with formula CI. That
tap is already the natural Homebrew distribution channel for personal CLI tools.

## Decision

Stable `ssh-drop` releases use GoReleaser to build prebuilt macOS and Linux
archives for `amd64` and `arm64`, publish GitHub Release artifacts, and generate
`Formula/ssh-drop.rb` in `flexdinesh/homebrew-tap`.

The release workflow opens or updates a pull request against the tap instead of
pushing directly to tap `main`. The tap branch is deterministic per version, such
as `ssh-drop-v0.1.0`, so rerunning a release updates the same pull request.

The formula declares `rsync` as a runtime dependency and leaves Linux clipboard
tools optional.

## Consequences

Users can install stable releases with `brew install flexdinesh/tap/ssh-drop`.

The source repository needs a `HOMEBREW_TAP_TOKEN` secret with write and pull
request access to `flexdinesh/homebrew-tap`.

The tap repository remains responsible for Homebrew-native style, audit, install,
and formula test checks before a formula update is merged.

GoReleaser's formula publisher is deprecated in newer versions, so the workflows
pin a formula-compatible GoReleaser version unless the project later migrates to
casks or a different formula update path.
