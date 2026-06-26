# ssh-drop Homebrew Releases PRD

## Problem Statement

`ssh-drop` is currently installable only as a Go module command, and the repository has no release automation, no GitHub Actions workflows, no GoReleaser config, no local Git tags, and no Homebrew formula. The README already describes stable SemVer releases, but the implementation does not yet create GitHub Release artifacts or update the existing `flexdinesh/homebrew-tap` repository.

The user wants a repeatable release path where a stable `ssh-drop` version can be published from `main`, packaged with GoReleaser, and made available through a custom Homebrew tap without hand-editing formula checksums.

## Solution

Set up GoReleaser-powered releases for `ssh-drop`.

The repository will canonicalize on `github.com/flexdinesh/ssh-drop`, then add GoReleaser configuration, GitHub Actions CI, a manual stable release workflow, Homebrew formula publishing to `flexdinesh/homebrew-tap`, and release documentation. Stable releases will be created by manually dispatching the release workflow from `main`; the workflow will run tests, create or reuse the next `v0.1.x` tag, run GoReleaser, publish macOS and Linux archives/checksums to GitHub Releases, and open or update a pull request against the custom tap.

## User Stories

1. As a user, I want to install `ssh-drop` with Homebrew, so that I do not need Go installed locally.
2. As a user, I want the Homebrew install command to be `brew install flexdinesh/tap/ssh-drop`, so that it matches the existing custom tap.
3. As a user, I want Homebrew to install a prebuilt binary archive, so that installation is fast and does not depend on a local Go toolchain.
4. As a user, I want Homebrew to install `rsync` as a runtime dependency, so that the core transfer dependency is present.
5. As a Linux Homebrew user, I want the formula to work on Linux, so that I can install the same release channel outside macOS.
6. As an Apple Silicon Mac user, I want a native `darwin/arm64` binary, so that the installed CLI runs without translation.
7. As an Intel Mac user, I want a native `darwin/amd64` binary, so that the installed CLI works on older Macs.
8. As a Linux `amd64` user, I want a native `linux/amd64` binary, so that the installed CLI works on common Linux systems.
9. As a Linux `arm64` user, I want a native `linux/arm64` binary, so that the installed CLI works on ARM Linux systems.
10. As a user, I want `ssh-drop --version` to print the stable release version, so that I can verify what Homebrew installed.
11. As a user, I want `go install github.com/flexdinesh/ssh-drop/cmd/ssh-drop@latest` to remain supported, so that Go users can install through the Go toolchain.
12. As a user, I want stable releases to use SemVer tags like `v0.1.0`, so that `@latest` resolves normally.
13. As a maintainer, I want releases created from `main`, so that stable artifacts come from reviewed code.
14. As a maintainer, I want to start a release with a manual GitHub Actions dispatch, so that release timing is deliberate.
15. As a maintainer, I want the release workflow to create the next `v0.1.x` tag automatically, so that patch releases are consistent.
16. As a maintainer, I want rerunning the release workflow on an already-tagged commit to reuse that tag, so that failed publishing steps can be retried.
17. As a maintainer, I want GoReleaser to build release archives and checksums, so that artifact naming and checksum generation are standardized.
18. As a maintainer, I want GoReleaser to generate the Homebrew formula, so that formula URLs and checksums come from the published artifacts.
19. As a maintainer, I want the tap update to open a pull request, so that tap CI validates the formula before merge.
20. As a maintainer, I want the tap update branch to be deterministic per version, so that reruns update the same pull request.
21. As a maintainer, I want the tap repository to own Homebrew style, audit, install, and formula tests, so that release publishing and tap validation remain separate.
22. As a maintainer, I want a narrowly scoped `HOMEBREW_TAP_TOKEN`, so that the source repo workflow can write to the tap without overbroad credentials.
23. As a maintainer, I want CI to run tests and builds on pushes and pull requests, so that release-breaking changes are caught before `main`.
24. As a maintainer, I want `dev` branch pushes to run a GoReleaser snapshot, so that GoReleaser config problems are caught before stable release day.
25. As a maintainer, I want release documentation, so that the release process and required secret are discoverable.
26. As a maintainer, I want the README install section to list Homebrew first and Go install as an alternative, so that users see the intended stable install path.
27. As a maintainer, I want the GitHub module path, imports, README, GoReleaser config, and release URLs to agree on `github.com/flexdinesh/ssh-drop`, so that releases do not split identity across namespaces.
28. As a maintainer, I want no moving `latest` Git tag, so that Go's built-in SemVer resolution stays the source of truth.

## Implementation Decisions

- The canonical repository and Go module identity is `github.com/flexdinesh/ssh-drop`.
- Stable releases are plain SemVer tags on `main`, beginning with `v0.1.0`.
- The stable release workflow is manually dispatched.
- The release workflow selects the next `v0.1.x` tag, creates it if needed, and reuses an existing matching tag on `HEAD` when rerun.
- GoReleaser is the release engine for builds, archives, checksums, GitHub Release publishing, and Homebrew formula generation.
- Release archives target `darwin/amd64`, `darwin/arm64`, `linux/amd64`, and `linux/arm64`.
- Release archives contain the `ssh-drop` binary and README.
- Release builds stamp the existing `main.version` variable with the GoReleaser version.
- No new `internal/version` package is needed for this release setup.
- The Homebrew channel is the existing `flexdinesh/homebrew-tap` repository.
- The Homebrew install command is `brew install flexdinesh/tap/ssh-drop`.
- The formula installs prebuilt release archives rather than building from source.
- The formula declares `rsync` as a runtime dependency.
- The formula does not force-install a Linux clipboard backend; Linux clipboard support remains documented as optional.
- The release workflow uses a `HOMEBREW_TAP_TOKEN` secret with contents write and pull request write access to `flexdinesh/homebrew-tap`.
- The tap update opens or updates a pull request instead of pushing directly to the tap `main` branch.
- The tap update branch is deterministic per version, such as `ssh-drop-v0.1.0`.
- CI runs `go test ./...` and `go build ./cmd/ssh-drop`.
- Pushes to `dev` additionally run `goreleaser release --snapshot --clean` without publishing stable artifacts or touching Homebrew.
- There is no moving `latest` Git tag.

## Testing Decisions

- The main test seam is the release configuration boundary: validate GoReleaser config locally and run a snapshot build without publishing.
- CI should test source behavior with `go test ./...` and verify the binary builds with `go build ./cmd/ssh-drop`.
- GoReleaser snapshot validation should verify archive generation, version stamping, and Homebrew-generation compatibility without publishing.
- The stable release workflow should run tests before tag creation and before GoReleaser publishing.
- The Homebrew formula test should assert that `ssh-drop --version` includes the formula version.
- Tap CI should continue to own Homebrew-native validation, including style, audit, install, and formula test checks.
- Local verification should include `goreleaser check` and a snapshot release run before merging.

## Out of Scope

- Publishing to Homebrew core.
- Creating a dedicated `homebrew-ssh-drop` tap.
- Building the formula from source.
- Windows release artifacts.
- Linux package formats such as deb, rpm, snap, or winget.
- Code signing, notarization, SBOM generation, or provenance attestations.
- A moving `latest` Git tag.
- Automatic major/minor version selection beyond the initial `v0.1.x` patch line.
- Replacing the existing `flexdinesh/homebrew-tap` CI.
- Adding a richer version package with commit/date output.

## Further Notes

- GoReleaser currently documents Homebrew formulas as deprecated in favor of casks, but this project should use formulas because `ssh-drop` is a terminal CLI and the existing tap is formula-based.
- The source repository currently has no `.github` directory, so CI and release workflows will be new.
- The custom tap already contains a formula and CI pattern from `tokeninsights`; the `ssh-drop` release should reuse the tap PR model while letting GoReleaser generate the formula.
- The current `go.mod` and imports were already being updated from `github.com/dineshpandiyan/ssh-drop` to `github.com/flexdinesh/ssh-drop` when this PRD was written.
