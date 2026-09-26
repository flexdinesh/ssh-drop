package transfer

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"testing"
	"time"

	"github.com/flexdinesh/ssh-drop/internal/session"
)

func TestAuthenticatedConnectionTransfersRepeatedFiles(t *testing.T) {
	sshd, err := exec.LookPath("sshd")
	if err != nil {
		t.Skip("local sshd unavailable")
	}
	for _, name := range []string{"ssh", "ssh-keygen", "rsync"} {
		if _, err := exec.LookPath(name); err != nil {
			t.Skip(name + " unavailable")
		}
	}
	current, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for _, name := range []string{"host", "client"} {
		if output, err := exec.Command("ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-f", filepath.Join(dir, name)).CombinedOutput(); err != nil {
			t.Fatalf("key generation: %v %s", err, output)
		}
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	config := filepath.Join(dir, "sshd.conf")
	body := fmt.Sprintf("Port %d\nListenAddress 127.0.0.1\nHostKey %s\nAuthorizedKeysFile %s\nPidFile %s\nStrictModes no\nUsePAM no\nPasswordAuthentication no\nKbdInteractiveAuthentication no\nPubkeyAuthentication yes\nPermitRootLogin yes\nLogLevel ERROR\n", port, filepath.Join(dir, "host"), filepath.Join(dir, "client.pub"), filepath.Join(dir, "pid"))
	if err := os.WriteFile(config, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command(sshd, "-t", "-f", config).CombinedOutput(); err != nil {
		t.Skipf("local sshd prerequisites unavailable: %s", output)
	}
	logPath := filepath.Join(dir, "sshd.log")
	log, err := os.Create(logPath)
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	server := exec.Command(sshd, "-D", "-e", "-f", config)
	server.Stderr = log
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = server.Process.Kill(); _ = server.Wait() }()
	address := fmt.Sprintf("127.0.0.1:%d", port)
	deadline := time.Now().Add(3 * time.Second)
	for {
		conn, err := net.DialTimeout("tcp", address, 50*time.Millisecond)
		if err == nil {
			conn.Close()
			break
		}
		if time.Now().After(deadline) {
			output, _ := os.ReadFile(logPath)
			t.Fatalf("sshd did not start: %s", output)
		}
		time.Sleep(10 * time.Millisecond)
	}
	remote := session.Remote{Host: "127.0.0.1", Port: fmt.Sprint(port), User: current.Username, IdentityFile: filepath.Join(dir, "client"), Destination: filepath.Join(dir, "uploaded files")}
	connector := SSHConnection{CommandContext: func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, name, append([]string{"-F", "/dev/null", "-o", "IdentityAgent=none", "-o", "IdentitiesOnly=yes", "-o", "UserKnownHostsFile=" + filepath.Join(dir, "known_hosts")}, args...)...)
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	controlPath, err := connector.Connect(ctx, remote, "")
	if err != nil {
		output, _ := os.ReadFile(logPath)
		t.Fatalf("connect: %v\nsshd: %s", err, output)
	}
	defer connector.Close(remote, controlPath)
	for _, name := range []string{"first image.png", "second image.png"} {
		local := filepath.Join(dir, name)
		if err := os.WriteFile(local, []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
		req := session.TransferRequest{LocalPath: local, Remote: remote, DestinationDir: remote.Destination, DestinationPath: filepath.Join(remote.Destination, name), ControlPath: controlPath}
		for event := range (Runner{}).Begin(ctx, req) {
			if event.Output != "" {
				t.Log(event.Output)
			}
			if event.Done && event.Err != nil {
				t.Fatalf("upload %s: %v", name, event.Err)
			}
		}
		data, err := os.ReadFile(req.DestinationPath)
		if err != nil || string(data) != name {
			t.Fatalf("uploaded content: %q, %v", data, err)
		}
	}
	connector.Close(remote, controlPath)
	if err := exec.Command("ssh", "-S", controlPath, "-O", "check", remote.Target()).Run(); err == nil {
		t.Fatal("SSH master survived cleanup")
	}
}
