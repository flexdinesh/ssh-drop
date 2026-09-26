package transfer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/flexdinesh/ssh-drop/internal/session"
)

type connectionCapture struct {
	Args      []string
	Askpass   string
	Password  string
	SecretEnv string
}

func testConnection(t *testing.T, mode string) (SSHConnection, string) {
	t.Helper()
	log := filepath.Join(t.TempDir(), "connection.json")
	return SSHConnection{CommandContext: func(ctx context.Context, name string, args ...string) *exec.Cmd {
		cmd := exec.CommandContext(ctx, os.Args[0], append([]string{"-test.run=TestConnectionProcess", "--"}, args...)...)
		cmd.Env = append(os.Environ(), "SSH_DROP_TEST_MODE="+mode, "SSH_DROP_TEST_LOG="+log)
		return cmd
	}}, log
}

func TestConnectionUsesKeysBeforePasswordAndCleansUp(t *testing.T) {
	connector, log := testConnection(t, "ok")
	remote := session.Remote{Host: "files", User: "deploy", Port: "2222", IdentityFile: "/keys/my key"}
	controlPath, err := connector.Connect(context.Background(), remote, "")
	if err != nil {
		t.Fatal(err)
	}
	defer connector.Close(remote, controlPath)
	capture := readConnectionCapture(t, log)
	args := strings.Join(capture.Args, " ")
	for _, want := range []string{"-i /keys/my key", "-p 2222", "BatchMode=yes", "ControlPath=" + controlPath, "-M -N -f", "deploy@files"} {
		if !strings.Contains(args, want) {
			t.Fatalf("missing %q in %s", want, args)
		}
	}
	if capture.Askpass != "" {
		t.Fatal("key authentication used askpass")
	}
	info, err := os.Stat(filepath.Dir(controlPath))
	if err != nil || info.Mode().Perm() != 0o700 {
		t.Fatalf("socket directory must be private: %v", err)
	}
	connector.Close(remote, controlPath)
	if _, err := os.Stat(filepath.Dir(controlPath)); !os.IsNotExist(err) {
		t.Fatal("socket directory leaked")
	}
}

func TestConnectionPasswordIsShortLivedAndNotInMasterEnvironment(t *testing.T) {
	connector, log := testConnection(t, "ok")
	remote := session.Remote{Host: "files"}
	controlPath, err := connector.Connect(context.Background(), remote, "test-password")
	if err != nil {
		t.Fatal(err)
	}
	defer connector.Close(remote, controlPath)
	capture := readConnectionCapture(t, log)
	if capture.Password != "test-password" {
		t.Fatal("askpass did not receive password")
	}
	if capture.SecretEnv != "" {
		t.Fatal("password was retained in SSH master environment")
	}
	if _, err := os.Stat(filepath.Dir(capture.Askpass)); !os.IsNotExist(err) {
		t.Fatal("askpass files leaked")
	}
}

func TestConnectionPromptsOnlyForAuthenticationFailures(t *testing.T) {
	for _, test := range []struct {
		mode     string
		password bool
	}{{"auth", true}, {"network", false}, {"hostkey", false}} {
		t.Run(test.mode, func(t *testing.T) {
			connector, _ := testConnection(t, test.mode)
			_, err := connector.Connect(context.Background(), session.Remote{Host: "files"}, "")
			if err == nil {
				t.Fatal("expected connection error")
			}
			if errors.Is(err, session.ErrPasswordRequired) != test.password {
				t.Fatalf("incorrect password fallback: %v", err)
			}
		})
	}
}

func TestConnectionCanBeCanceled(t *testing.T) {
	connector, _ := testConnection(t, "wait")
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := connector.Connect(ctx, session.Remote{Host: "files"}, "")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}

func readConnectionCapture(t *testing.T, path string) connectionCapture {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var capture connectionCapture
	if err := json.Unmarshal(data, &capture); err != nil {
		t.Fatal(err)
	}
	return capture
}

func TestConnectionProcess(t *testing.T) {
	mode := os.Getenv("SSH_DROP_TEST_MODE")
	if mode == "" {
		return
	}
	var args []string
	for i, arg := range os.Args {
		if arg == "--" {
			args = os.Args[i+1:]
			break
		}
	}
	for _, arg := range args {
		if arg == "exit" {
			os.Exit(0)
		}
	}
	capture := connectionCapture{Args: args, Askpass: os.Getenv("SSH_ASKPASS"), SecretEnv: os.Getenv("SSH_DROP_PASSWORD")}
	if capture.Askpass != "" {
		output, err := exec.Command(capture.Askpass).Output()
		if err != nil {
			os.Exit(2)
		}
		capture.Password = string(output)
	}
	data, _ := json.Marshal(capture)
	if err := os.WriteFile(os.Getenv("SSH_DROP_TEST_LOG"), data, 0o600); err != nil {
		os.Exit(2)
	}
	switch mode {
	case "auth":
		fmt.Fprintln(os.Stderr, "Permission denied (publickey,password).")
		os.Exit(255)
	case "network":
		fmt.Fprintln(os.Stderr, "Connection refused")
		os.Exit(255)
	case "hostkey":
		fmt.Fprintln(os.Stderr, "Host key verification failed.")
		os.Exit(255)
	case "wait":
		time.Sleep(time.Minute)
	}
	os.Exit(0)
}
