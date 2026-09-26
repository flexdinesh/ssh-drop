package tui_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/flexdinesh/ssh-drop/internal/session"
	"github.com/flexdinesh/ssh-drop/internal/tui"
)

type contextConnection struct {
	ctx context.Context
}

func (c *contextConnection) Connect(ctx context.Context, remote session.Remote, password string) (string, error) {
	c.ctx = ctx
	return "queued-socket", nil
}

func (c *contextConnection) Close(session.Remote, string) {}

func TestQuitCancelsQueuedSuccessfulConnection(t *testing.T) {
	connector := &contextConnection{}
	model := tui.NewModel(session.Start{Config: configWithRemotes(), PreselectedRemote: "cb"}, tui.Services{Connector: connector})
	queued := model.Init()()
	if connector.ctx.Err() != nil {
		t.Fatal("connection canceled before delivery")
	}
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	model, ok := updated.(tui.Model)
	if !ok || !model.Quitting() || connector.ctx.Err() == nil {
		t.Fatal("quit did not release queued connection ownership")
	}
	if _, ok := queued.(tui.ConnectionMsg); !ok {
		t.Fatal("expected queued connection result")
	}
}

func TestAcceptedConnectionLivesUntilHostChanges(t *testing.T) {
	connector := &contextConnection{}
	model := newModel(t, session.Start{Config: configWithRemotes(), PreselectedRemote: "cb"}, tui.Services{Connector: connector})
	if connector.ctx.Err() != nil {
		t.Fatal("accepted connection lost its owner context")
	}
	previous := connector.ctx
	model = update(t, model, key("r"))
	model = update(t, model, key("enter"))
	if previous.Err() == nil || connector.ctx.Err() != nil || model.State() != tui.StateDrop {
		t.Fatal("host switch did not replace connection ownership")
	}
}

func TestQuitHonoredForQueuedTransferOutcomes(t *testing.T) {
	for _, outcome := range []struct {
		name string
		err  error
	}{{"success", nil}, {"failure", errors.New("upload failed")}, {"disconnected", session.ErrConnectionLost}} {
		for _, confirmed := range []bool{false, true} {
			t.Run(outcome.name+"/confirm="+strconv.FormatBool(confirmed), func(t *testing.T) {
				file := filepath.Join(t.TempDir(), "image.png")
				if err := os.WriteFile(file, []byte("image"), 0o600); err != nil {
					t.Fatal(err)
				}
				transfer := &fakeTransfer{}
				model := newModel(t, session.Start{Config: configWithRemotes(), PreselectedRemote: "cb"}, tui.Services{Transferer: transfer, Clipboard: &fakeClipboard{}})
				model = submitPath(t, model, file)
				if confirmed {
					model = update(t, model, key("q"))
					model = update(t, model, key("y"))
				} else {
					model = update(t, model, tea.KeyMsg{Type: tea.KeyCtrlC})
				}
				model = drainTransfer(t, model, transfer, session.TransferEvent{Done: true, Err: outcome.err})
				if !model.Quitting() {
					t.Fatal("queued outcome ignored pending quit")
				}
				if outcome.err == nil && model.Summary().Successes != 1 || outcome.err != nil && model.Summary().Failures != 1 {
					t.Fatal("quit lost terminal outcome")
				}
			})
		}
	}
}

func TestExplicitUserWithKeyAuthDoesNotPrompt(t *testing.T) {
	connector := &fakeConnection{}
	model := newModel(t, session.Start{Config: configWithRemotes(), PreselectedRemote: "files"}, tui.Services{Connector: connector})
	if model.State() != tui.StateDrop || connector.Calls != 1 || connector.Password != "" {
		t.Fatal("key auth should connect directly")
	}
}

func TestRepeatedDropsReuseConnectionAndCopyEachPath(t *testing.T) {
	connector := &fakeConnection{RequirePassword: true}
	transfer := &fakeTransfer{}
	clipboard := &fakeClipboard{}
	model := newModel(t, session.Start{Config: configWithRemotes(), PreselectedRemote: "files"}, tui.Services{Connector: connector, Transferer: transfer, Clipboard: clipboard})
	for _, r := range "secret-pass" {
		model = update(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	model = update(t, model, key("enter"))
	for _, name := range []string{"first.png", "second.png"} {
		file := filepath.Join(t.TempDir(), name)
		if err := os.WriteFile(file, []byte("image"), 0o600); err != nil {
			t.Fatal(err)
		}
		model = submitPath(t, model, file)
		if model.State() != tui.StateUpload {
			t.Fatalf("expected upload, got %s", model.State())
		}
		if transfer.Request.Password != "" || transfer.Request.ControlPath != "test-socket-files" {
			t.Fatal("upload must use authenticated connection, not password")
		}
		model = drainTransfer(t, model, transfer, session.TransferEvent{Done: true})
		if clipboard.Copied != "/var/tmp/"+name {
			t.Fatal("wrong copied path")
		}
		if !strings.Contains(model.View(), "Drop another image") || !viewContains(model.View(), clipboard.Copied) {
			t.Fatal("repeat workflow or last result missing")
		}
	}
	if connector.Calls != 2 || model.Summary().Successes != 2 {
		t.Fatal("repeated uploads reconnected or lost summary")
	}
}

func TestChangingRemoteClosesPreviousConnection(t *testing.T) {
	connector := &fakeConnection{}
	model := newModel(t, session.Start{Config: configWithRemotes(), PreselectedRemote: "files"}, tui.Services{Connector: connector})
	model = update(t, model, key("r"))
	model = update(t, model, key("up"))
	model = update(t, model, key("enter"))
	if len(connector.Closed) != 1 || connector.Closed[0] != "test-socket-files" {
		t.Fatal("previous host connection was not closed")
	}
	if model.SelectedRemote().Name != "cb" {
		t.Fatal("host did not change")
	}
}

func TestLostConnectionPreservesFileForReconnectAndRetry(t *testing.T) {
	file := filepath.Join(t.TempDir(), "image.png")
	if err := os.WriteFile(file, []byte("image"), 0o600); err != nil {
		t.Fatal(err)
	}
	transfer := &fakeTransfer{}
	connector := &fakeConnection{}
	model := newModel(t, session.Start{Config: configWithRemotes(), PreselectedRemote: "cb"}, tui.Services{Connector: connector, Transferer: transfer})
	model = submitPath(t, model, file)
	model = drainTransfer(t, model, transfer, session.TransferEvent{Done: true, Err: session.ErrConnectionLost})
	if model.State() != tui.StateConnectionFailed {
		t.Fatal("connection failure was not distinguished")
	}
	model = update(t, model, key("enter"))
	if !viewContains(model.View(), "image.png") {
		t.Fatal("file was lost during reconnect")
	}
	model = update(t, model, key("enter"))
	if model.State() != tui.StateUpload || transfer.Request.LocalPath != file {
		t.Fatal("could not retry after reconnect")
	}
}

func TestConnectionErrorsOfferRecovery(t *testing.T) {
	model := newModel(t, session.Start{Config: configWithRemotes(), PreselectedRemote: "cb"}, tui.Services{Connector: &fakeConnection{Err: errors.New("connection refused")}})
	if model.State() != tui.StateConnectionFailed || !viewContains(model.View(), "enter reconnect") {
		t.Fatal("connection error lacks recovery")
	}
}

func TestTerminalLayoutsStayWithinCellWidth(t *testing.T) {
	for _, width := range []int{32, 40, 80, 120} {
		for _, height := range []int{12, 24} {
			model := newModel(t, session.Start{Config: configWithRemotes(), PreselectedRemote: "cb"}, tui.Services{})
			model = update(t, model, tea.WindowSizeMsg{Width: width, Height: height})
			if got := widestLine(model.View()); got > width {
				t.Fatalf("%dx%d: width %d", width, height, got)
			}
			if got := lipgloss.Height(model.View()); got > height {
				t.Fatalf("%dx%d: height %d:\n%s", width, height, got, model.View())
			}
		}
	}
}

func TestAllStatesFitSmallTerminal(t *testing.T) {
	start := session.Start{Config: configWithRemotes(), PreselectedRemote: "cb"}
	file := filepath.Join(t.TempDir(), "screenshot.png")
	if err := os.WriteFile(file, []byte("png"), 0o600); err != nil {
		t.Fatal(err)
	}
	transfer := &fakeTransfer{}
	ready := newModel(t, start, tui.Services{Transferer: transfer})
	upload := submitPath(t, ready, file)
	success := drainTransfer(t, upload, transfer, session.TransferEvent{Done: true})
	failed := drainTransfer(t, upload, transfer, session.TransferEvent{Done: true, Err: errors.New("transfer failed")})
	password := newModel(t, start, tui.Services{Connector: &fakeConnection{RequirePassword: true}})
	picker := newModel(t, session.Start{Config: configWithRemotes()}, tui.Services{})
	connecting := tui.NewModel(start, tui.Services{Connector: &fakeConnection{}})
	connectionError := newModel(t, start, tui.Services{Connector: &fakeConnection{Err: errors.New("network unreachable")}})
	for _, model := range []tui.Model{ready, upload, success, failed, password, picker, connecting, connectionError} {
		model = update(t, model, tea.WindowSizeMsg{Width: 40, Height: 12})
		view := model.View()
		if widestLine(view) > 40 || lipgloss.Height(view) > 12 {
			t.Fatalf("%s exceeds 40x12:\n%s", model.State(), view)
		}
	}
}
