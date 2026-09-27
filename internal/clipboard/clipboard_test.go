package clipboard

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCopyPassesExactPathAndXclipSelection(t *testing.T) {
	log := filepath.Join(t.TempDir(), "copy")
	copier := Copier{
		LookPath: func(name string) (string, error) {
			if name == "xclip" {
				return name, nil
			}
			return "", exec.ErrNotFound
		},
		CommandContext: func(ctx context.Context, name string, args ...string) *exec.Cmd {
			cmd := exec.CommandContext(ctx, os.Args[0], append([]string{"-test.run=^TestClipboardProcess$", "--", name}, args...)...)
			cmd.Env = append(os.Environ(), "SSH_DROP_CLIPBOARD_MODE=copy", "SSH_DROP_CLIPBOARD_LOG="+log)
			return cmd
		},
	}
	if err := copier.Copy(context.Background(), "/uploads/my image.png"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "xclip -selection clipboard\n/uploads/my image.png" {
		t.Fatalf("unexpected clipboard content: %q", data)
	}
}

func TestCopyTimesOutHungBackend(t *testing.T) {
	copier := Copier{
		Timeout:  50 * time.Millisecond,
		LookPath: func(name string) (string, error) { return name, nil },
		CommandContext: func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestClipboardProcess$")
			cmd.Env = append(os.Environ(), "SSH_DROP_CLIPBOARD_MODE=wait")
			return cmd
		},
	}
	if err := copier.Copy(context.Background(), "/uploads/image.png"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected clipboard timeout, got %v", err)
	}
}

func TestClipboardProcess(t *testing.T) {
	mode := os.Getenv("SSH_DROP_CLIPBOARD_MODE")
	if mode == "" {
		return
	}
	if mode == "wait" {
		time.Sleep(time.Minute)
		os.Exit(0)
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		os.Exit(2)
	}
	var args []string
	for i, arg := range os.Args {
		if arg == "--" {
			args = os.Args[i+1:]
			break
		}
	}
	if err := os.WriteFile(os.Getenv("SSH_DROP_CLIPBOARD_LOG"), append([]byte(strings.Join(args, " ")+"\n"), data...), 0o600); err != nil {
		os.Exit(2)
	}
	os.Exit(0)
}
