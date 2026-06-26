package clipboard

import (
	"bytes"
	"errors"
	"os/exec"
)

type Copier struct {
	LookPath func(string) (string, error)
	Command  func(string, ...string) *exec.Cmd
}

func (c Copier) Copy(value string) error {
	if c.LookPath == nil {
		c.LookPath = exec.LookPath
	}
	if c.Command == nil {
		c.Command = exec.Command
	}
	for _, backend := range []string{"pbcopy", "wl-copy", "xclip"} {
		if _, err := c.LookPath(backend); err == nil {
			return c.copyWith(backend, value)
		}
	}
	return errors.New("no clipboard backend found")
}

func (c Copier) copyWith(backend string, value string) error {
	args := []string{}
	if backend == "xclip" {
		args = []string{"-selection", "clipboard"}
	}
	cmd := c.Command(backend, args...)
	cmd.Stdin = bytes.NewBufferString(value)
	return cmd.Run()
}
