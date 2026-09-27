package tui

import (
	"context"
	"errors"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/flexdinesh/ssh-drop/internal/session"
)

type blockingClipboard struct {
	started chan struct{}
	release chan struct{}
}

func (c *blockingClipboard) Copy(ctx context.Context, _ string) error {
	close(c.started)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.release:
		return errors.New("clipboard unavailable")
	}
}

func TestClipboardCopyKeepsInputResponsiveAndPreservesUpload(t *testing.T) {
	clipboard := &blockingClipboard{started: make(chan struct{}), release: make(chan struct{})}
	model := NewModel(session.Start{}, Services{Clipboard: clipboard})
	model.currentRequest = session.TransferRequest{LocalPath: "/source.png", DestinationPath: "/uploads/source.png"}
	updated, command := model.Update(TransferEventMsg{Event: session.TransferEvent{Done: true}})
	next, ok := updated.(Model)
	if !ok || command == nil || next.state != StateDrop || next.summary.Successes != 1 {
		t.Fatal("upload did not complete before copying")
	}
	result := make(chan tea.Msg, 1)
	go func() { result <- command() }()
	select {
	case <-clipboard.started:
	case <-time.After(time.Second):
		t.Fatal("clipboard command did not start")
	}
	updated, _ = next.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("next.png")})
	next, ok = updated.(Model)
	if !ok || next.input.Value() != "next.png" {
		t.Fatal("clipboard copy blocked input")
	}
	close(clipboard.release)
	updated, _ = next.Update(<-result)
	next, ok = updated.(Model)
	if !ok || next.summary.Successes != 1 || next.summary.Failures != 0 || next.statusKind != statusWarning || next.input.Value() != "next.png" {
		t.Fatal("clipboard failure lost upload success or new edits")
	}
	next.shutdown()
}

func TestStaleClipboardResultCannotReplaceNewerStatus(t *testing.T) {
	model := NewModel(session.Start{}, Services{})
	model.state = StateDrop
	previous := model.clipboardGeneration
	model.invalidateClipboard()
	model.status = "newer result"
	model.statusKind = statusError
	updated, _ := model.Update(ClipboardMsg{Generation: previous})
	next, ok := updated.(Model)
	if !ok || next.status != "newer result" || next.statusKind != statusError {
		t.Fatal("stale clipboard result replaced current status")
	}
	model.shutdown()
}

func TestClipboardResultPreservesNewValidationError(t *testing.T) {
	model := NewModel(session.Start{}, Services{})
	model.state = StateDrop
	model.statusKind = statusCopying
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model, ok := updated.(Model)
	if !ok {
		t.Fatal("unexpected model")
	}
	updated, _ = model.Update(ClipboardMsg{Generation: model.clipboardGeneration})
	next, ok := updated.(Model)
	if !ok || next.status != "enter a file path" || next.statusKind != statusError {
		t.Fatal("clipboard completion hid a validation error")
	}
	model.shutdown()
}

func TestShutdownCancelsRunningAndPreventsQueuedClipboardCopies(t *testing.T) {
	for _, queued := range []bool{true, false} {
		t.Run(map[bool]string{true: "queued", false: "running"}[queued], func(t *testing.T) {
			clipboard := &blockingClipboard{started: make(chan struct{}), release: make(chan struct{})}
			model := NewModel(session.Start{}, Services{Clipboard: clipboard})
			command := model.copyCommand()
			if queued {
				model.shutdown()
				command()
				select {
				case <-clipboard.started:
					t.Fatal("clipboard started after shutdown")
				default:
				}
				return
			}
			result := make(chan tea.Msg, 1)
			go func() { result <- command() }()
			<-clipboard.started
			model.shutdown()
			msg, ok := (<-result).(ClipboardMsg)
			if !ok || !errors.Is(msg.Err, context.Canceled) {
				t.Fatal("shutdown left clipboard operation active")
			}
		})
	}
}
