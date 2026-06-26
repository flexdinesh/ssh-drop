package transfer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/dineshpandiyan/ssh-drop/internal/session"
)

type Command struct {
	Name string
	Args []string
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
		if err := r.run(ctx, events, BuildMkdirCommand(req)); err != nil {
			events <- session.TransferEvent{Done: true, Err: classifyCancel(ctx, err)}
			return
		}
		if err := r.run(ctx, events, BuildRsyncCommand(req)); err != nil {
			events <- session.TransferEvent{Done: true, Err: classifyCancel(ctx, err)}
			return
		}
		events <- session.TransferEvent{Done: true}
	}()
	return events
}

func (r Runner) run(ctx context.Context, events chan<- session.TransferEvent, command Command) error {
	cmd := r.CommandContext(ctx, command.Name, command.Args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("%s stdout: %w", command.Name, err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("%s stderr: %w", command.Name, err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("%s start: %w", command.Name, err)
	}
	done := make(chan struct{})
	go streamOutput(events, stdout, done)
	go streamOutput(events, stderr, done)
	err = cmd.Wait()
	<-done
	<-done
	if err != nil {
		return fmt.Errorf("%s failed: %w", command.Name, err)
	}
	return nil
}

func streamOutput(events chan<- session.TransferEvent, reader io.Reader, done chan<- struct{}) {
	defer func() { done <- struct{}{} }()
	buf := make([]byte, 4096)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			events <- session.TransferEvent{Output: string(buf[:n])}
		}
		if err != nil {
			return
		}
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
	args = append(args, req.Remote.Target(), "mkdir -p "+quoteIfNeeded(req.DestinationDir))
	return Command{Name: "ssh", Args: args}
}

func BuildRsyncCommand(req session.TransferRequest) Command {
	args := []string{"--progress"}
	if transport := sshTransport(req.Remote); transport != "ssh" {
		args = append(args, "-e", transport)
	}
	args = append(args, req.LocalPath, fmt.Sprintf("%s:%s", req.Remote.Target(), quoteIfNeeded(req.DestinationPath)))
	return Command{Name: "rsync", Args: args}
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

func sshTransport(remote session.Remote) string {
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
