# Releases

Stable releases are SemVer Git tags on `main`. The `dev` branch is an automatic
Go install channel for changes pushed to `main`.

## Install

Stable Homebrew install:

```bash
brew install --cask flexdinesh/tap/ssh-drop
```

Alternative stable install with Go:

```bash
go install github.com/flexdinesh/ssh-drop/cmd/ssh-drop@latest
```

Specific stable version:

```bash
go install github.com/flexdinesh/ssh-drop/cmd/ssh-drop@v0.1.0
```

Development install:

```bash
go install github.com/flexdinesh/ssh-drop/cmd/ssh-drop@dev
```

## Current Policy

Stable releases are created manually from the latest code on `main` by running
the GitHub Actions **Release** workflow with `main` selected. Dispatches from
other refs are skipped. The workflow checks out the latest `main` when it starts.
Each new release commit gets the next `v0.1.x` release; rerunning for an already
tagged commit reuses that tag. If there are no `v0.1.x` release tags yet, the
first dispatch creates `v0.1.0`.

Examples:

```bash
v0.1.0
v0.1.1
v0.1.2
```

The workflow creates the tag, runs GoReleaser, publishes macOS and Linux
archives plus checksums, marks the stable GitHub Release as **Latest**, and
opens or updates a pull request against
`flexdinesh/homebrew-tap`.

If GitHub Release publishing succeeds but a downstream publisher fails, rerun
the workflow from the same commit after fixing the downstream issue. The
workflow reuses the tag already on `HEAD`, and GoReleaser replaces the existing
GitHub Release assets before retrying the remaining publishers.

Do not create a moving `latest` tag. Go already resolves `@latest` to the
newest SemVer tag.

## Development Channel

Every push to `main` runs tests, builds the CLI, and validates GoReleaser snapshot
packaging. After those checks pass, CI creates or updates `dev` to the tested
commit. Pull requests run the same checks without publishing. Failed checks
leave `dev` unchanged; superseded runs skip publishing so they cannot roll
the channel backward.

`dev` is maintained by CI, not used for development work. CI replaces any
obsolete history on that branch. This replaces the old snapshot job triggered
by pushes to `dev`; no workflow now runs on pushes to that branch.

Go resolves `@dev` from the branch to a commit version (usually a pseudo-version).
There is no `dev` tag or development GitHub Release, and this channel does not
publish Homebrew updates. Only the manual stable workflow creates version tags
and changes the latest GitHub Release. Go proxies may briefly cache branch
queries; use `GOPROXY=direct go install github.com/flexdinesh/ssh-drop/cmd/ssh-drop@dev`
to query the repository directly when checking a just-published update.

## Required Secret

The release workflow requires:

- `HOMEBREW_TAP_TOKEN`: a fine-grained GitHub token with contents write and pull request write access to `flexdinesh/homebrew-tap`.

The workflow also uses the built-in `GITHUB_TOKEN` to create tags and publish
the GitHub Release in this repository.

## Homebrew

The Homebrew cask installs prebuilt release archives instead of building from
source. It documents `rsync` as a required command in the cask caveats because
`ssh-drop` requires `rsync` to transfer files.

The cask does not install `rsync` or Linux clipboard tools. Clipboard copy is
optional after a successful upload; Linux users can install `wl-clipboard` or
`xclip` if they want automatic clipboard copy.

The tap pull request branch is deterministic per version, such as
`ssh-drop-v0.1.0`, so rerunning a failed release updates the same tap pull
request.

## Release Steps

1. Merge the release-ready code to `main`.
2. Run the **Release** workflow from GitHub Actions with `main` selected.
3. Confirm the workflow created or reused the expected `v0.1.x` tag.
4. Review the generated GitHub Release artifacts and checksums.
5. Merge the generated `flexdinesh/homebrew-tap` pull request after tap CI passes.
6. Verify with `brew install --cask flexdinesh/tap/ssh-drop` and `ssh-drop --version`.

## Verify Locally

```bash
go test ./...
go build ./cmd/ssh-drop
goreleaser release --snapshot --clean
```

The workflows pin GoReleaser `v2.18.0` so releases can publish Homebrew casks
through `homebrew_casks`. The snapshot command remains useful locally because it
verifies archive and cask generation without publishing.

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
