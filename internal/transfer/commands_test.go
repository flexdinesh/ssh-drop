package transfer_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/flexdinesh/ssh-drop/internal/session"
	"github.com/flexdinesh/ssh-drop/internal/transfer"
)

func TestAliasRemoteBuildsMkdirAndRsyncCommands(t *testing.T) {
	req := session.TransferRequest{
		LocalPath:       "/Users/dee/report.txt",
		DestinationPath: "/tmp/report.txt",
		DestinationDir:  "/tmp",
		Remote:          session.Remote{Name: "cb", Host: "cb", Destination: "/tmp"},
	}

	mkdir := transfer.BuildMkdirCommand(req)
	if mkdir.Name != "ssh" {
		t.Fatalf("expected ssh command, got %q", mkdir.Name)
	}
	if !reflect.DeepEqual(mkdir.Args, []string{"cb", "mkdir -p /tmp"}) {
		t.Fatalf("unexpected mkdir args: %#v", mkdir.Args)
	}

	rsync := transfer.BuildRsyncCommand(req)
	if rsync.Name != "rsync" {
		t.Fatalf("expected rsync command, got %q", rsync.Name)
	}
	if !reflect.DeepEqual(rsync.Args, []string{"--progress", "/Users/dee/report.txt", "cb:/tmp/report.txt"}) {
		t.Fatalf("unexpected rsync args: %#v", rsync.Args)
	}
}

func TestExplicitRemoteBuildsOpenSSHOptions(t *testing.T) {
	req := session.TransferRequest{
		LocalPath:       "/Users/dee/report.txt",
		DestinationPath: "/var/tmp/drop zone/report.txt",
		DestinationDir:  "/var/tmp/drop zone",
		Remote: session.Remote{
			Name:         "files",
			Host:         "files.example.com",
			User:         "deploy",
			Port:         "2222",
			IdentityFile: "/Users/dee/.ssh/files access",
			ForwardAgent: true,
			Destination:  "/var/tmp/drop zone",
		},
	}

	mkdir := transfer.BuildMkdirCommand(req)
	wantMkdir := []string{
		"-i", "/Users/dee/.ssh/files access",
		"-A",
		"-p", "2222",
		"deploy@files.example.com",
		"mkdir -p '/var/tmp/drop zone'",
	}
	if !reflect.DeepEqual(mkdir.Args, wantMkdir) {
		t.Fatalf("unexpected mkdir args:\nwant %#v\ngot  %#v", wantMkdir, mkdir.Args)
	}

	rsync := transfer.BuildRsyncCommand(req)
	wantRsync := []string{
		"--progress",
		"-e", "ssh -i '/Users/dee/.ssh/files access' -A -p 2222",
		"/Users/dee/report.txt",
		"deploy@files.example.com:'/var/tmp/drop zone/report.txt'",
	}
	if !reflect.DeepEqual(rsync.Args, wantRsync) {
		t.Fatalf("unexpected rsync args:\nwant %#v\ngot  %#v", wantRsync, rsync.Args)
	}
}

func TestPOSIXQuoteHandlesSingleQuotes(t *testing.T) {
	got := transfer.POSIXQuote("/tmp/dinesh's file")
	want := "'/tmp/dinesh'\"'\"'s file'"
	if got != want {
		t.Fatalf("unexpected quote:\nwant %s\ngot  %s", want, got)
	}
}

func TestRunnerUsesAskpassForPasswordRequests(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "askpass.log")
	runner := transfer.Runner{
		CommandContext: func(ctx context.Context, name string, args ...string) *exec.Cmd {
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestAskpassHelperProcess", "--", name)
			cmd.Env = append(os.Environ(),
				"GO_WANT_ASKPASS_HELPER=1",
				"SSH_DROP_HELPER_LOG="+logPath,
			)
			return cmd
		},
	}
	req := session.TransferRequest{
		LocalPath:       "/Users/dee/report.txt",
		DestinationPath: "/tmp/report.txt",
		DestinationDir:  "/tmp",
		Password:        "secret-pass",
		Remote: session.Remote{
			Name:        "files",
			Host:        "files.example.com",
			User:        "deploy",
			Destination: "/tmp",
		},
	}

	var done session.TransferEvent
	for event := range runner.Begin(context.Background(), req) {
		if event.Done {
			done = event
		}
	}
	if done.Err != nil {
		t.Fatalf("transfer failed: %v", done.Err)
	}
	bytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(string(bytes))
	want := strings.Join([]string{
		"ssh|force|ssh-drop|secret-pass",
		"rsync|force|ssh-drop|secret-pass",
	}, "\n")
	if got != want {
		t.Fatalf("unexpected askpass env:\nwant %s\ngot  %s", want, got)
	}
}

func TestAskpassHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_ASKPASS_HELPER") != "1" {
		return
	}
	askpass := os.Getenv("SSH_ASKPASS")
	output, err := exec.Command(askpass).Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "askpass failed: %v\n", err)
		os.Exit(2)
	}
	commandName := ""
	for i, arg := range os.Args {
		if arg == "--" && i+1 < len(os.Args) {
			commandName = os.Args[i+1]
			break
		}
	}
	line := fmt.Sprintf(
		"%s|%s|%s|%s\n",
		commandName,
		os.Getenv("SSH_ASKPASS_REQUIRE"),
		os.Getenv("DISPLAY"),
		strings.TrimSpace(string(output)),
	)
	file, err := os.OpenFile(os.Getenv("SSH_DROP_HELPER_LOG"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open log failed: %v\n", err)
		os.Exit(2)
	}
	if _, err := file.WriteString(line); err != nil {
		fmt.Fprintf(os.Stderr, "write log failed: %v\n", err)
		os.Exit(2)
	}
	if err := file.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "close log failed: %v\n", err)
		os.Exit(2)
	}
	os.Exit(0)
}
