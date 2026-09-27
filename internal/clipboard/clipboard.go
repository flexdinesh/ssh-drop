package clipboard

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"time"
)

type Copier struct {
	LookPath       func(string) (string, error)
	CommandContext func(context.Context, string, ...string) *exec.Cmd
	Timeout        time.Duration
}

func (c Copier) Copy(ctx context.Context, value string) error {
	if c.LookPath == nil {
		c.LookPath = exec.LookPath
	}
	if c.CommandContext == nil {
		c.CommandContext = exec.CommandContext
	}
	if c.Timeout <= 0 {
		c.Timeout = 2 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()
	for _, backend := range []string{"pbcopy", "wl-copy", "xclip"} {
		if _, err := c.LookPath(backend); err == nil {
			return c.copyWith(ctx, backend, value)
		}
	}
	return errors.New("no clipboard backend found")
}

func (c Copier) copyWith(ctx context.Context, backend string, value string) error {
	args := []string{}
	if backend == "xclip" {
		args = []string{"-selection", "clipboard"}
	}
	cmd := c.CommandContext(ctx, backend, args...)
	cmd.WaitDelay = 250 * time.Millisecond
	cmd.Stdin = bytes.NewBufferString(value)
	err := cmd.Run()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}
