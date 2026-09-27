package transfer

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/flexdinesh/ssh-drop/internal/session"
)

func processRunner(mode, marker string) Runner {
	return Runner{CommandContext: func(ctx context.Context, name string, _ ...string) *exec.Cmd {
		processMode := "ok"
		if name == "ssh" {
			processMode = mode
		}
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestTransferProcess$")
		cmd.Env = append(os.Environ(), "SSH_DROP_TRANSFER_MODE="+processMode, "SSH_DROP_TRANSFER_MARKER="+marker)
		return cmd
	}}
}

func TestRunnerDrainsBothStreamsForSlowConsumer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	events := processRunner("output", "").Begin(ctx, session.TransferRequest{})
	time.Sleep(200 * time.Millisecond)
	stdout, stderr := 0, 0
	done := false
	for event := range events {
		stdout += strings.Count(event.Output, "x")
		stderr += strings.Count(event.Output, "y")
		if event.Done {
			done = true
			if event.Err != nil {
				t.Fatal(event.Err)
			}
		}
	}
	if !done || stdout != 73728 || stderr != 73728 {
		t.Fatalf("incomplete output: stdout=%d stderr=%d done=%v", stdout, stderr, done)
	}
}

func TestRunnerCancellationReleasesAbandonedOutputAndAskpass(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "askpass-path")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events := processRunner("flood", marker).Begin(ctx, session.TransferRequest{Password: "test-password"})
	deadline := time.After(5 * time.Second)
	for len(events) < cap(events) {
		select {
		case <-deadline:
			t.Fatal("process did not fill the output buffer")
		case <-time.After(10 * time.Millisecond):
		}
	}
	path, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	deadline = time.After(5 * time.Second)
	for {
		if _, err := os.Stat(filepath.Dir(string(path))); os.IsNotExist(err) {
			break
		}
		select {
		case <-deadline:
			t.Fatal("abandoned output prevented askpass cleanup")
		case <-time.After(10 * time.Millisecond):
		}
	}
	for range events {
	}
}

func TestTransferProcess(t *testing.T) {
	mode := os.Getenv("SSH_DROP_TRANSFER_MODE")
	if mode == "" {
		return
	}
	switch mode {
	case "output":
		if _, err := os.Stdout.Write([]byte(strings.Repeat("x", 73728))); err != nil {
			os.Exit(2)
		}
		if _, err := os.Stderr.Write([]byte(strings.Repeat("y", 73728))); err != nil {
			os.Exit(2)
		}
	case "flood":
		if err := os.WriteFile(os.Getenv("SSH_DROP_TRANSFER_MARKER"), []byte(os.Getenv("SSH_ASKPASS")), 0o600); err != nil {
			os.Exit(2)
		}
		output := []byte(strings.Repeat("x", 4096))
		for {
			if _, err := os.Stdout.Write(output); err != nil {
				os.Exit(2)
			}
		}
	}
	os.Exit(0)
}
