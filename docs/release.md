# Releases

Releases are SemVer Git tags on `main`.

## Install

Stable Homebrew install:

```bash
brew install flexdinesh/tap/ssh-drop
```

Alternative stable install with Go:

```bash
go install github.com/flexdinesh/ssh-drop/cmd/ssh-drop@latest
```

Specific stable version:

```bash
go install github.com/flexdinesh/ssh-drop/cmd/ssh-drop@v0.1.0
```

## Current Policy

Stable releases are created manually from the latest code on `main` by running
the GitHub Actions release workflow. Each dispatch creates the next `v0.1.x`
release. If there are no `v0.1.x` release tags yet, the first dispatch creates
`v0.1.0`.

Examples:

```bash
v0.1.0
v0.1.1
v0.1.2
```

The workflow creates the tag, runs GoReleaser, publishes macOS and Linux
archives plus checksums, and opens or updates a pull request against
`flexdinesh/homebrew-tap`.

If GitHub Release publishing succeeds but a downstream publisher fails, rerun
the workflow from the same commit after fixing the downstream issue. The
workflow reuses the tag already on `HEAD`, and GoReleaser replaces the existing
GitHub Release assets before retrying the remaining publishers.

Do not create a moving `latest` tag. Go already resolves `@latest` to the
newest SemVer tag.

## Required Secret

The release workflow requires:

- `HOMEBREW_TAP_TOKEN`: a fine-grained GitHub token with contents write and pull request write access to `flexdinesh/homebrew-tap`.

The workflow also uses the built-in `GITHUB_TOKEN` to create tags and publish
the GitHub Release in this repository.

## Homebrew

The Homebrew formula installs prebuilt release archives instead of building from
source. It declares `rsync` as a runtime dependency because `ssh-drop` requires
`rsync` to transfer files.

The formula does not install Linux clipboard tools. Clipboard copy is optional
after a successful upload; Linux users can install `wl-clipboard` or `xclip` if
they want automatic clipboard copy.

The tap pull request branch is deterministic per version, such as
`ssh-drop-v0.1.0`, so rerunning a failed release updates the same tap pull
request.

## Release Steps

1. Merge the release-ready code to `main`.
2. Run the **Release** workflow from GitHub Actions.
3. Confirm the workflow created or reused the expected `v0.1.x` tag.
4. Review the generated GitHub Release artifacts and checksums.
5. Merge the generated `flexdinesh/homebrew-tap` pull request after tap CI passes.
6. Verify with `brew install flexdinesh/tap/ssh-drop` and `ssh-drop --version`.

## Verify Locally

```bash
go test ./...
go build ./cmd/ssh-drop
goreleaser release --snapshot --clean
```

The workflows pin GoReleaser `v2.9.0` because GoReleaser deprecated formula
publishing through `brews` in later versions. The snapshot command remains useful
locally with newer GoReleaser versions because it verifies archive and formula
generation without publishing.

## Switching Minor Versions

Switch manually when `0.1.x` no longer fits the release line.

To switch, update `.github/workflows/release.yml` so the tag selector uses the
new minor line, such as `v0.2.*`, and starts at `v0.2.0`.

After that, releases should continue as:

```bash
v0.2.0
v0.2.1
v0.2.2
```
