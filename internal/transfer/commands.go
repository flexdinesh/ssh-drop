package transfer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/flexdinesh/ssh-drop/internal/session"
)

type Command struct {
	Name string
	Args []string
	Env  []string
}

type Runner struct {
	CommandContext func(context.Context, string, ...string) *exec.Cmd
}

func (r Runner) Begin(ctx context.Context, req session.TransferRequest) <-chan session.TransferEvent {
	events := make(chan session.TransferEvent, 16)
	go func() {
		defer close(events)
		if r.CommandContext == nil {
			r.CommandContext = exec.CommandContext
		}
		localPath, err := filepath.Abs(req.LocalPath)
		if err != nil {
			sendEvent(ctx, events, session.TransferEvent{Done: true, Err: err})
			return
		}
		req.LocalPath = localPath
		if req.ControlPath != "" {
			check := r.CommandContext(ctx, "ssh", "-S", req.ControlPath, "-O", "check", req.Remote.Target())
			if err := check.Run(); err != nil {
				sendEvent(ctx, events, session.TransferEvent{Done: true, Err: classifyCancel(ctx, session.ErrConnectionLost)})
				return
			}
		}
		authEnv, cleanup, err := passwordAuthEnv(req.Password)
		if err != nil {
			sendEvent(ctx, events, session.TransferEvent{Done: true, Err: err})
			return
		}
		defer cleanup()
		if err := r.run(ctx, events, withEnv(BuildMkdirCommand(req), authEnv)); err != nil {
			sendEvent(ctx, events, session.TransferEvent{Done: true, Err: r.transferError(ctx, req, err)})
			return
		}
		if err := r.run(ctx, events, withEnv(BuildRsyncCommand(req), authEnv)); err != nil {
			sendEvent(ctx, events, session.TransferEvent{Done: true, Err: r.transferError(ctx, req, err)})
			return
		}
		sendEvent(ctx, events, session.TransferEvent{Done: true})
	}()
	return events
}

func (r Runner) transferError(ctx context.Context, req session.TransferRequest, err error) error {
	if ctx.Err() != nil {
		return classifyCancel(ctx, err)
	}
	if req.ControlPath != "" {
		if checkErr := r.CommandContext(ctx, "ssh", "-S", req.ControlPath, "-O", "check", req.Remote.Target()).Run(); checkErr != nil {
			return session.ErrConnectionLost
		}
	}
	return err
}

func (r Runner) run(ctx context.Context, events chan<- session.TransferEvent, command Command) error {
	cmd := r.CommandContext(ctx, command.Name, command.Args...)
	if len(command.Env) > 0 {
		cmd.Env = append(cmd.Environ(), command.Env...)
	}
	// exec owns the readers and drains both streams before Run returns.
	cmd.Stdout = eventWriter{ctx: ctx, events: events}
	cmd.Stderr = eventWriter{ctx: ctx, events: events}
	cmd.WaitDelay = 2 * time.Second
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s failed: %w", command.Name, err)
	}
	return nil
}

func withEnv(command Command, env []string) Command {
	if len(env) == 0 {
		return command
	}
	command.Env = append(command.Env, env...)
	return command
}

func passwordAuthEnv(password string) ([]string, func(), error) {
	if password == "" {
		return nil, func() {}, nil
	}
	dir, err := os.MkdirTemp("", "ssh-drop-askpass-")
	if err != nil {
		return nil, func() {}, fmt.Errorf("askpass temp dir: %w", err)
	}
	cleanup := func() {
		_ = os.RemoveAll(dir)
	}
	helper := filepath.Join(dir, "askpass")
	secret := filepath.Join(dir, "password")
	if err := os.WriteFile(secret, []byte(password), 0o600); err != nil {
		cleanup()
		return nil, func() {}, fmt.Errorf("askpass password: %w", err)
	}
	script := "#!/bin/sh\ncat " + POSIXQuote(secret) + "\n"
	if err := os.WriteFile(helper, []byte(script), 0o700); err != nil {
		cleanup()
		return nil, func() {}, fmt.Errorf("askpass helper: %w", err)
	}
	return []string{
		"SSH_ASKPASS=" + helper,
		"SSH_ASKPASS_REQUIRE=force",
		"DISPLAY=ssh-drop",
	}, cleanup, nil
}

type eventWriter struct {
	ctx    context.Context
	events chan<- session.TransferEvent
}

func (w eventWriter) Write(output []byte) (int, error) {
	sendEvent(w.ctx, w.events, session.TransferEvent{Output: string(output)})
	return len(output), nil
}

func sendEvent(ctx context.Context, events chan<- session.TransferEvent, event session.TransferEvent) {
	// Deliver a terminal result when space remains, even after cancellation.
	select {
	case events <- event:
		return
	default:
	}
	select {
	case events <- event:
	case <-ctx.Done():
	}
}

func classifyCancel(ctx context.Context, err error) error {
	if errors.Is(ctx.Err(), context.Canceled) {
		return session.ErrTransferCanceled
	}
	return err
}

func BuildMkdirCommand(req session.TransferRequest) Command {
	args := append([]string{}, sshArgs(req.Remote)...)
	args = append(args, controlArgs(req.ControlPath)...)
	args = append(args, req.Remote.Target(), "mkdir -p "+quoteIfNeeded(req.DestinationDir))
	return Command{Name: "ssh", Args: args}
}

func BuildRsyncCommand(req session.TransferRequest) Command {
	args := []string{"--progress"}
	if transport := sshTransport(req.Remote, req.ControlPath); transport != "ssh" {
		args = append(args, "-e", transport)
	}
	localPath := req.LocalPath
	if !filepath.IsAbs(localPath) {
		localPath = "./" + localPath
	}
	args = append(args, "--", localPath, fmt.Sprintf("%s:%s", req.Remote.Target(), quoteIfNeeded(req.DestinationPath)))
	// Keep our explicit POSIX quoting consistent across macOS's older rsync
	// and modern rsync, which otherwise adds a second layer of escaping.
	return Command{Name: "rsync", Args: args, Env: []string{"RSYNC_OLD_ARGS=1", "RSYNC_PROTECT_ARGS=0"}}
}

func sshArgs(remote session.Remote) []string {
	var args []string
	if remote.IdentityFile != "" {
		args = append(args, "-i", remote.IdentityFile)
	}
	if remote.ForwardAgent {
		args = append(args, "-A")
	}
	if remote.Port != "" {
		args = append(args, "-p", remote.Port)
	}
	return args
}

func sshTransport(remote session.Remote, controlPath string) string {
	args := []string{"ssh"}
	if remote.IdentityFile != "" {
		args = append(args, "-i", quoteIfNeeded(remote.IdentityFile))
	}
	if remote.ForwardAgent {
		args = append(args, "-A")
	}
	if remote.Port != "" {
		args = append(args, "-p", remote.Port)
	}
	for _, arg := range controlArgs(controlPath) {
		args = append(args, quoteIfNeeded(arg))
	}
	return strings.Join(args, " ")
}

func quoteIfNeeded(value string) string {
	if value == "" {
		return "''"
	}
	if strings.IndexFunc(value, func(r rune) bool {
		return !(r == '/' || r == '.' || r == '-' || r == '_' || r == ':' || r == '+' || r == '=' || r == ',' || r == '@' ||
			(r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'))
	}) == -1 {
		return value
	}
	return POSIXQuote(value)
}

func POSIXQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
