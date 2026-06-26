package app_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dineshpandiyan/ssh-drop/internal/app"
	"github.com/dineshpandiyan/ssh-drop/internal/session"
)

func TestHelpPrintsV1Surface(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := app.Run([]string{"--help"}, app.Deps{
		Stdout:  &stdout,
		Stderr:  &stderr,
		Version: "test",
	})

	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	out := stdout.String()
	for _, want := range []string{"ssh-drop [--config <path>] [--to <remote-name>]", "ssh-drop --version"} {
		if !strings.Contains(out, want) {
			t.Fatalf("help output missing %q:\n%s", want, out)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr, got %s", stderr.String())
	}
}

func TestVersionPrintsVersion(t *testing.T) {
	var stdout bytes.Buffer

	code := app.Run([]string{"--version"}, app.Deps{
		Stdout:  &stdout,
		Version: "1.2.3",
	})

	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if got := strings.TrimSpace(stdout.String()); got != "ssh-drop 1.2.3" {
		t.Fatalf("unexpected version output %q", got)
	}
}

func TestMissingConfigFailsWithSample(t *testing.T) {
	var stderr bytes.Buffer
	home := t.TempDir()

	code := app.Run(nil, app.Deps{
		Stderr:   &stderr,
		HomeDir:  home,
		LookPath: foundPath,
	})

	if code == 0 {
		t.Fatal("expected non-zero exit")
	}
	errOut := stderr.String()
	for _, want := range []string{
		"config error",
		filepath.Join(home, ".config", "ssh-drop.conf"),
		"[remote.cb]",
		"host = cb",
		"identity_file = ~/.ssh/files",
		"destination = /tmp/ssh-drop/",
	} {
		if !strings.Contains(errOut, want) {
			t.Fatalf("missing %q in stderr:\n%s", want, errOut)
		}
	}
}

func TestDefaultConfigPathUsesXDGConfigHome(t *testing.T) {
	var stderr bytes.Buffer
	xdgConfigHome := t.TempDir()

	code := app.Run(nil, app.Deps{
		Stderr:   &stderr,
		HomeDir:  t.TempDir(),
		LookPath: foundPath,
		EnvLookup: func(key string) (string, bool) {
			if key == "XDG_CONFIG_HOME" {
				return xdgConfigHome, true
			}
			return "", false
		},
	})

	if code == 0 {
		t.Fatal("expected non-zero exit")
	}
	want := filepath.Join(xdgConfigHome, "ssh-drop.conf")
	if !strings.Contains(stderr.String(), want) {
		t.Fatalf("missing default XDG config path %q in stderr:\n%s", want, stderr.String())
	}
}

func TestConfigLoadsRemotesInOrderAndStartsUI(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "ssh-drop.conf")
	writeFile(t, cfgPath, `
[remote.cb]
host = cb

[remote.files]
host = files.example.com
user = deploy
identity_file = ~/.ssh/files
forward_agent = true
destination = $DROP_DEST
`)

	var started session.Start
	code := app.Run([]string{"--config", cfgPath, "--to", "files"}, app.Deps{
		HomeDir:  dir,
		LookPath: foundPath,
		EnvLookup: func(key string) (string, bool) {
			if key == "DROP_DEST" {
				return "/var/tmp/drop zone", true
			}
			return "", false
		},
		RunUI: func(start session.Start) (session.Summary, error) {
			started = start
			return session.Summary{}, nil
		},
	})

	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if started.PreselectedRemote != "files" {
		t.Fatalf("unexpected preselected remote %q", started.PreselectedRemote)
	}
	if got := remoteNames(started.Config.Remotes); strings.Join(got, ",") != "cb,files" {
		t.Fatalf("remotes not in config order: %#v", got)
	}
	cb := started.Config.Remotes[0]
	if cb.Destination != "/tmp/ssh-drop/" {
		t.Fatalf("expected default destination /tmp/ssh-drop/, got %q", cb.Destination)
	}
	files := started.Config.Remotes[1]
	if files.User != "deploy" || files.IdentityFile != filepath.Join(dir, ".ssh/files") || !files.ForwardAgent || files.Destination != "/var/tmp/drop zone" {
		t.Fatalf("unexpected files remote: %#v", files)
	}
}

func TestUnknownRemoteFailsBeforeUI(t *testing.T) {
	cfgPath := writeConfig(t, `[remote.cb]
host = cb
`)
	var stderr bytes.Buffer
	calledUI := false

	code := app.Run([]string{"--config", cfgPath, "--to", "missing"}, app.Deps{
		Stderr:   &stderr,
		LookPath: foundPath,
		RunUI: func(session.Start) (session.Summary, error) {
			calledUI = true
			return session.Summary{}, nil
		},
	})

	if code == 0 {
		t.Fatal("expected non-zero exit")
	}
	if calledUI {
		t.Fatal("UI should not start")
	}
	if !strings.Contains(stderr.String(), `unknown remote "missing"`) {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestMissingEnvVarInConfigFailsBeforeUI(t *testing.T) {
	cfgPath := writeConfig(t, `[remote.prod]
host = prod
destination = $SSH_DROP_DIR
`)
	var stderr bytes.Buffer
	calledUI := false

	code := app.Run([]string{"--config", cfgPath}, app.Deps{
		Stderr:    &stderr,
		LookPath:  foundPath,
		EnvLookup: func(string) (string, bool) { return "", false },
		RunUI: func(session.Start) (session.Summary, error) {
			calledUI = true
			return session.Summary{}, nil
		},
	})

	if code == 0 {
		t.Fatal("expected non-zero exit")
	}
	if calledUI {
		t.Fatal("UI should not start")
	}
	if !strings.Contains(stderr.String(), "SSH_DROP_DIR") {
		t.Fatalf("stderr should identify missing env var: %s", stderr.String())
	}
}

func TestMissingRsyncFailsBeforeUI(t *testing.T) {
	cfgPath := writeConfig(t, `[remote.cb]
host = cb
`)
	var stderr bytes.Buffer
	calledUI := false

	code := app.Run([]string{"--config", cfgPath}, app.Deps{
		Stderr:   &stderr,
		LookPath: func(string) (string, error) { return "", errors.New("missing") },
		RunUI: func(session.Start) (session.Summary, error) {
			calledUI = true
			return session.Summary{}, nil
		},
	})

	if code == 0 {
		t.Fatal("expected non-zero exit")
	}
	if calledUI {
		t.Fatal("UI should not start")
	}
	if !strings.Contains(stderr.String(), "rsync is required") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestNormalQuitPrintsCompactSummaryAndExitsZero(t *testing.T) {
	cfgPath := writeConfig(t, `[remote.cb]
host = cb
`)
	var stdout bytes.Buffer

	code := app.Run([]string{"--config", cfgPath}, app.Deps{
		Stdout:   &stdout,
		LookPath: foundPath,
		RunUI: func(session.Start) (session.Summary, error) {
			return session.Summary{
				Successes:              2,
				Failures:               1,
				Canceled:               1,
				SuccessfulDestinations: []string{"/tmp/a.txt", "/tmp/b.txt"},
			}, nil
		},
	})

	if code != 0 {
		t.Fatalf("expected normal quit to exit 0, got %d", code)
	}
	out := stdout.String()
	for _, want := range []string{"Summary", "success: 2", "failed: 1", "canceled: 1", "/tmp/a.txt", "/tmp/b.txt"} {
		if !strings.Contains(out, want) {
			t.Fatalf("summary missing %q:\n%s", want, out)
		}
	}
}

func foundPath(name string) (string, error) {
	return "/usr/bin/" + name, nil
}

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ssh-drop.conf")
	writeFile(t, path, body)
	return path
}

func writeFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(strings.TrimSpace(body)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func remoteNames(remotes []session.Remote) []string {
	names := make([]string, 0, len(remotes))
	for _, remote := range remotes {
		names = append(names, remote.Name)
	}
	return names
}
