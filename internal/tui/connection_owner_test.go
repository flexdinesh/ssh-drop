package tui

import (
	"context"
	"testing"
	"time"

	"github.com/flexdinesh/ssh-drop/internal/session"
)

type shutdownConnection struct {
	started chan struct{}
	closing chan struct{}
	release chan struct{}
	closed  chan string
	wait    bool
}

type shutdownTransfer struct {
	ctx     context.Context
	release chan struct{}
}

func (s *shutdownTransfer) Begin(ctx context.Context, _ session.TransferRequest) <-chan session.TransferEvent {
	s.ctx = ctx
	events := make(chan session.TransferEvent)
	go func() {
		<-ctx.Done()
		<-s.release
		close(events)
	}()
	return events
}

func TestShutdownCancelsAndAwaitsActiveTransfer(t *testing.T) {
	transfer := &shutdownTransfer{release: make(chan struct{})}
	model := NewModel(session.Start{}, Services{Transferer: transfer})
	model.startTransfer()
	done := make(chan struct{})
	go func() { model.shutdown(); close(done) }()
	select {
	case <-transfer.ctx.Done():
	case <-time.After(time.Second):
		close(transfer.release)
		model.cancelTransfer()
		t.Fatal("shutdown did not cancel active upload")
	}
	select {
	case <-done:
		close(transfer.release)
		t.Fatal("shutdown returned before upload cleanup")
	default:
	}
	close(transfer.release)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("shutdown did not await upload completion")
	}
}

func TestClosedCanceledTransferReportsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	events := make(chan session.TransferEvent)
	close(events)
	msg, ok := waitForTransfer(ctx, events)().(TransferEventMsg)
	if !ok || msg.Event.Err != session.ErrTransferCanceled {
		t.Fatal("canceled transfer was classified as a failure")
	}
}

func (c *shutdownConnection) Connect(ctx context.Context, _ session.Remote, _ string) (string, error) {
	close(c.started)
	if c.wait {
		<-ctx.Done()
	}
	return "owned-socket", nil
}

func (c *shutdownConnection) Close(_ session.Remote, path string) {
	close(c.closing)
	<-c.release
	c.closed <- path
}

func TestShutdownAwaitsQueuedAndPendingConnectionCleanup(t *testing.T) {
	for _, pending := range []bool{false, true} {
		name := "queued-success"
		if pending {
			name = "pending-connect"
		}
		t.Run(name, func(t *testing.T) {
			connector := &shutdownConnection{started: make(chan struct{}), closing: make(chan struct{}), release: make(chan struct{}), closed: make(chan string, 1), wait: pending}
			model := NewModel(session.Start{Config: session.Config{Remotes: []session.Remote{{Name: "files", Host: "files"}}}}, Services{Connector: connector})
			command := model.Init()
			if pending {
				go command()
				<-connector.started
			} else {
				command() // Success queued, never delivered to Update.
			}
			done := make(chan struct{})
			go func() { model.shutdown(); close(done) }()
			<-connector.closing
			select {
			case <-done:
				t.Fatal("shutdown returned before master closure")
			default:
			}
			close(connector.release)
			<-done
			select {
			case path := <-connector.closed:
				if path != "owned-socket" {
					t.Fatal("wrong connection closed")
				}
			default:
				t.Fatal("shutdown returned without releasing master")
			}
		})
	}
}

func TestShutdownPreventsLateConnectionCommands(t *testing.T) {
	connector := &shutdownConnection{started: make(chan struct{})}
	model := NewModel(session.Start{Config: session.Config{Remotes: []session.Remote{{Name: "files", Host: "files"}}}}, Services{Connector: connector})
	command := model.Init()
	model.shutdown()
	command()
	select {
	case <-connector.started:
		t.Fatal("connection started after shutdown")
	default:
	}
}
