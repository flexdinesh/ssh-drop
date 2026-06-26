# ADR 0002: Publish Stable Releases Through The Homebrew Tap

## Status

Accepted

## Context

`ssh-drop` is a terminal CLI, and users should be able to install stable
versions without installing Go. The repository already uses Go-compatible SemVer
release tags in the README, but it had no release automation or Homebrew cask.

There is an existing `flexdinesh/homebrew-tap` repository with Homebrew CI. That
tap is already the natural Homebrew distribution channel for personal CLI tools.

## Decision

Stable `ssh-drop` releases use GoReleaser to build prebuilt macOS and Linux
archives for `amd64` and `arm64`, publish GitHub Release artifacts, and generate
`Casks/ssh-drop.rb` in `flexdinesh/homebrew-tap`.

The release workflow opens or updates a pull request against the tap instead of
pushing directly to tap `main`. The tap branch is deterministic per version, such
as `ssh-drop-v0.1.0`, so rerunning a release updates the same pull request.

The cask documents `rsync` as a required command in its caveats and leaves Linux
clipboard tools optional.

## Consequences

Users can install stable releases with `brew install --cask flexdinesh/tap/ssh-drop`.

The source repository needs a `HOMEBREW_TAP_TOKEN` secret with write and pull
request access to `flexdinesh/homebrew-tap`.

The tap repository remains responsible for Homebrew-native style, audit, and
install checks before a cask update is merged.

GoReleaser's generated cask dependency array does not pass tap style checks, so
the cask does not declare `rsync` as a Homebrew dependency.

GoReleaser's formula publisher is deprecated in newer versions, so `ssh-drop`
uses the supported cask publisher.
