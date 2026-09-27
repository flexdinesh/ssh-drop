package tui

import (
	"context"
	"sync"

	"github.com/flexdinesh/ssh-drop/internal/session"
)

// Own completed and pending connections before their messages reach the model.
type connectionOwner struct {
	connector Connector
	mu        sync.Mutex
	pending   sync.WaitGroup
	closing   bool
	paths     map[string]session.Remote
}

func (o *connectionOwner) connect(ctx context.Context, remote session.Remote, password string) (string, error) {
	o.mu.Lock()
	if o.closing {
		o.mu.Unlock()
		return "", context.Canceled
	}
	o.pending.Add(1)
	o.mu.Unlock()
	defer o.pending.Done()
	path, err := o.connector.Connect(ctx, remote, password)
	if path != "" {
		o.mu.Lock()
		o.paths[path] = remote
		o.mu.Unlock()
	}
	return path, err
}

func (o *connectionOwner) close(path string) {
	o.mu.Lock()
	if o.closing {
		o.mu.Unlock()
		return
	}
	remote, owned := o.paths[path]
	delete(o.paths, path)
	if owned {
		o.pending.Add(1)
	}
	o.mu.Unlock()
	if owned {
		defer o.pending.Done()
		o.connector.Close(remote, path)
	}
}

func (m Model) shutdown() {
	if m.cancelTransfer != nil {
		m.cancelTransfer()
	}
	if m.transferEvents != nil {
		for range m.transferEvents {
		}
	}
	m.clipboard.shutdown()
	if m.cancelConnection != nil {
		m.cancelConnection()
	}
	m.connections.shutdown()
}

func (o *connectionOwner) shutdown() {
	o.mu.Lock()
	o.closing = true
	o.mu.Unlock()
	// No command may start after closing; every started command registers
	// its result before Done, including results queued when the TUI quits.
	o.pending.Wait()
	o.mu.Lock()
	paths := o.paths
	o.paths = make(map[string]session.Remote)
	o.mu.Unlock()
	for path, remote := range paths {
		o.connector.Close(remote, path)
	}
}
