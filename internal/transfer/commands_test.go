package transfer_test

import (
	"reflect"
	"testing"

	"github.com/dineshpandiyan/ssh-drop/internal/session"
	"github.com/dineshpandiyan/ssh-drop/internal/transfer"
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
