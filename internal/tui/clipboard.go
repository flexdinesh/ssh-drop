package tui

import (
	"context"
	"fmt"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
)

type ClipboardMsg struct {
	Generation int
	Err        error
}

type clipboardOwner struct {
	copier  Clipboard
	ctx     context.Context
	cancel  context.CancelFunc
	mu      sync.Mutex
	pending sync.WaitGroup
	closing bool
}

func newClipboardOwner(copier Clipboard) *clipboardOwner {
	ctx, cancel := context.WithCancel(context.Background())
	return &clipboardOwner{copier: copier, ctx: ctx, cancel: cancel}
}

func (o *clipboardOwner) copy(ctx context.Context, value string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	o.mu.Lock()
	if o.closing {
		o.mu.Unlock()
		return context.Canceled
	}
	o.pending.Add(1)
	o.mu.Unlock()
	defer o.pending.Done()
	return o.copier.Copy(ctx, value)
}

func (o *clipboardOwner) shutdown() {
	o.mu.Lock()
	o.closing = true
	o.cancel()
	o.mu.Unlock()
	o.pending.Wait()
}

func (m *Model) invalidateClipboard() {
	if m.cancelClipboard != nil {
		m.cancelClipboard()
		m.cancelClipboard = nil
	}
	m.clipboardGeneration++
}

func (m *Model) copyCommand() tea.Cmd {
	ctx, cancel := context.WithCancel(m.clipboard.ctx)
	m.cancelClipboard = cancel
	generation := m.clipboardGeneration
	destination := m.currentRequest.DestinationPath
	owner := m.clipboard
	return func() tea.Msg {
		defer cancel()
		return ClipboardMsg{Generation: generation, Err: owner.copy(ctx, destination)}
	}
}

func (m Model) updateClipboard(msg ClipboardMsg) (tea.Model, tea.Cmd) {
	if msg.Generation != m.clipboardGeneration || m.state != StateDrop || m.statusKind != statusCopying {
		return m, nil
	}
	m.cancelClipboard = nil
	if msg.Err != nil {
		m.status += fmt.Sprintf("\nclipboard warning: %v", msg.Err)
		m.statusKind = statusWarning
	} else {
		m.statusKind = statusSuccess
	}
	return m, nil
}
