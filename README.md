# SSH Drop

ssh drop is an interactive CLI that transfers local files to remote machines over SSH using rsync. It was purpose-built to drag and drop screenshots into terminal windows for AI harnesses running in remote SSH sessions.

## Install

The stable install path is Homebrew:

```bash
brew install flexdinesh/tap/ssh-drop
```

Go install is also supported if you already have [Go](https://go.dev) installed.

`@latest` resolves to the newest stable SemVer tag, such as `v0.1.0`. There is no moving `latest` Git tag.

```bash
# Install the latest stable release.
go install github.com/flexdinesh/ssh-drop/cmd/ssh-drop@latest

# Install a specific stable release.
go install github.com/flexdinesh/ssh-drop/cmd/ssh-drop@v0.1.0
```

`ssh-drop` also requires `rsync` in your `PATH` for transfers. The Homebrew formula installs `rsync`.

Clipboard copy uses the first available backend from `pbcopy`, `wl-copy`, or `xclip`. macOS includes `pbcopy`; Linux users can install `wl-clipboard` or `xclip` if they want automatic clipboard copy after upload.

## Usage

```bash
# Start an interactive drop session and pick a configured remote.
ssh-drop

# Preselect a remote by name and skip the picker.
ssh-drop --to dev

# Use an alternate config file.
ssh-drop --config ~/alt-ssh-drop.conf

# Show help.
ssh-drop --help

# Show the installed version.
ssh-drop --version
```

## Config

`ssh-drop` loads remotes from `~/.config/ssh-drop.conf` (or `$XDG_CONFIG_HOME/ssh-drop.conf`). The file is INI-style with `[remote.<name>]` sections. `host` is required and may be an OpenSSH alias or a real hostname. `destination` defaults to `/tmp/ssh-drop/`.

```ini
# Reuse an existing ~/.ssh/config alias.
[remote.ox]
host = ox
destination = /tmp/ssh-drop/

# Or model explicit SSH fields with id file
[remote.captain]
host = captain.local
user = pika
identity_file = ~/.ssh/captain
forward_agent = true
destination = /tmp/ssh-drop/

# Or model explicit SSH fields with user/pass
# password will be prompted in the tui
[remote.awesome]
host = 192.168.1.79
user = shiny
destination = /tmp/ssh-drop/

```

Config values expand `~` and environment variables. If the config is missing, `ssh-drop` prints a sample to get you started.

## Releases

Releases are created from `main`. See [docs/release.md](docs/release.md).
