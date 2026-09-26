package transfer

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/flexdinesh/ssh-drop/internal/session"
)

// SSHConnection owns a private OpenSSH master for one selected remote.
// Passwords are used only to authenticate the master, never for uploads.
type SSHConnection struct {
	CommandContext func(context.Context, string, ...string) *exec.Cmd
}

func (s SSHConnection) command(ctx context.Context, args ...string) *exec.Cmd {
	if s.CommandContext != nil {
		return s.CommandContext(ctx, "ssh", args...)
	}
	return exec.CommandContext(ctx, "ssh", args...)
}

func (s SSHConnection) Connect(ctx context.Context, remote session.Remote, password string) (string, error) {
	// Keep below Unix socket path limits, including macOS's long TMPDIR.
	dir, err := os.MkdirTemp("/tmp", "ssh-drop-")
	if err != nil {
		return "", err
	}
	controlPath := filepath.Join(dir, "socket")
	args := append(sshArgs(remote), "-M", "-N", "-f",
		"-o", "ControlPath="+controlPath, "-o", "ControlPersist=no",
		"-o", "ConnectTimeout=10", "-o", "ConnectionAttempts=1",
		"-o", "StrictHostKeyChecking=accept-new", "-o", "NumberOfPasswordPrompts=1")
	if password == "" {
		args = append(args, "-o", "BatchMode=yes")
	} else {
		args = append(args, "-o", "BatchMode=no", "-o", "PreferredAuthentications=keyboard-interactive,password")
	}
	args = append(args, remote.Target())
	env, cleanup, err := passwordAuthEnv(password)
	if err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}
	defer cleanup()
	connectContext, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	cmd := s.command(connectContext, args...)
	cmd.Env = append(cmd.Environ(), env...)
	// A background SSH master inherits stderr. A pipe would keep Run waiting
	// for EOF until the master exits, so capture diagnostics in a private file.
	logPath := filepath.Join(dir, "connect.log")
	log, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		s.Close(remote, controlPath)
		return "", err
	}
	cmd.Stdout, cmd.Stderr = log, log
	err = cmd.Run()
	_ = log.Close()
	output, _ := os.ReadFile(logPath)
	_ = os.Remove(logPath)
	if err != nil || connectContext.Err() != nil {
		s.Close(remote, controlPath)
		if connectContext.Err() != nil {
			return "", fmt.Errorf("%w\n%s", connectContext.Err(), strings.TrimSpace(string(output)))
		}
		text := strings.TrimSpace(string(output))
		if strings.Contains(text, "Permission denied") &&
			(strings.Contains(text, "password") || strings.Contains(text, "keyboard-interactive")) {
			return "", session.ErrPasswordRequired
		}
		return "", fmt.Errorf("could not connect: %w\n%s", err, text)
	}
	return controlPath, nil
}

func (s SSHConnection) Close(remote session.Remote, controlPath string) {
	if controlPath == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = s.command(ctx, "-S", controlPath, "-O", "exit", remote.Target()).Run()
	_ = os.RemoveAll(filepath.Dir(controlPath))
}

func controlArgs(controlPath string) []string {
	if controlPath == "" {
		return nil
	}
	// ProxyCommand=false prevents a disconnected master from silently opening
	// a new connection (and prompting outside Bubble Tea).
	return []string{"-o", "ControlMaster=no", "-o", "ControlPath=" + controlPath,
		"-o", "BatchMode=yes", "-o", "ProxyCommand=false"}
}
