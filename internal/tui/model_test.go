package tui_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dineshpandiyan/ssh-drop/internal/session"
	"github.com/dineshpandiyan/ssh-drop/internal/tui"
)

func TestMultipleRemotesStartAtPickerInConfigOrder(t *testing.T) {
	model := tui.NewModel(session.Start{Config: configWithRemotes()}, tui.Services{})

	if model.State() != tui.StateRemotePicker {
		t.Fatalf("expected remote picker, got %s", model.State())
	}
	view := model.View()
	first := strings.Index(view, "cb")
	second := strings.Index(view, "files")
	if first < 0 || second < 0 || first > second {
		t.Fatalf("picker should show remotes in config order:\n%s", view)
	}
	for _, want := range []string{"deploy@files.example.com", "-> /var/tmp"} {
		if !strings.Contains(view, want) {
			t.Fatalf("picker missing %q:\n%s", want, view)
		}
	}
}

func TestPreselectedRemoteStartsAtDropScreen(t *testing.T) {
	model := tui.NewModel(session.Start{
		Config:            configWithRemotes(),
		PreselectedRemote: "files",
	}, tui.Services{})

	if model.State() != tui.StateDrop {
		t.Fatalf("expected drop state, got %s", model.State())
	}
	if got := model.SelectedRemote().Name; got != "files" {
		t.Fatalf("expected files selected, got %q", got)
	}
	if !strings.Contains(model.View(), "deploy@files.example.com") {
		t.Fatalf("drop screen should show selected remote details:\n%s", model.View())
	}
}

func TestSingleRemoteStartsAtDropScreen(t *testing.T) {
	cfg := session.Config{Remotes: []session.Remote{{Name: "cb", Host: "cb", Destination: "/tmp"}}}

	model := tui.NewModel(session.Start{Config: cfg}, tui.Services{})

	if model.State() != tui.StateDrop {
		t.Fatalf("expected drop state, got %s", model.State())
	}
	if got := model.SelectedRemote().Name; got != "cb" {
		t.Fatalf("expected cb selected, got %q", got)
	}
}

func TestRemoteSelectionIsStickyAndCanBeChangedWithR(t *testing.T) {
	model := tui.NewModel(session.Start{Config: configWithRemotes()}, tui.Services{})

	model = update(t, model, key("down"))
	model = update(t, model, key("enter"))
	if model.State() != tui.StateDrop {
		t.Fatalf("expected drop state, got %s", model.State())
	}
	if got := model.SelectedRemote().Name; got != "files" {
		t.Fatalf("expected files selected, got %q", got)
	}

	model = update(t, model, key("r"))
	model = update(t, model, key("up"))
	model = update(t, model, key("enter"))
	if got := model.SelectedRemote().Name; got != "cb" {
		t.Fatalf("expected cb selected, got %q", got)
	}
}

func TestDropInputRejectsInvalidLocalFiles(t *testing.T) {
	dir := t.TempDir()
	model := tui.NewModel(session.Start{
		Config: session.Config{Remotes: []session.Remote{{Name: "cb", Host: "cb", Destination: "/tmp"}}},
	}, tui.Services{})

	model = submitPath(t, model, filepath.Join(dir, "missing.txt"))
	if !strings.Contains(model.View(), "does not exist") {
		t.Fatalf("expected missing path validation:\n%s", model.View())
	}

	model = tui.NewModel(session.Start{
		Config: session.Config{Remotes: []session.Remote{{Name: "cb", Host: "cb", Destination: "/tmp"}}},
	}, tui.Services{})
	model = submitPath(t, model, dir)
	if !strings.Contains(model.View(), "regular file") {
		t.Fatalf("expected regular file validation:\n%s", model.View())
	}
}

func TestDropInputAcceptsOneRegularFileAndComputesDestination(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "report.txt")
	if err := os.WriteFile(file, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	transfer := &fakeTransfer{}
	clipboard := &fakeClipboard{}
	model := tui.NewModel(session.Start{
		Config: session.Config{Remotes: []session.Remote{{Name: "cb", Host: "cb", Destination: "/uploads"}}},
	}, tui.Services{Transferer: transfer, Clipboard: clipboard})

	model = submitPath(t, model, file)
	model = drainTransfer(t, model, transfer, session.TransferEvent{Output: "sending report.txt\n"})
	model = drainTransfer(t, model, transfer, session.TransferEvent{Done: true})

	if model.State() != tui.StateDrop {
		t.Fatalf("expected drop state after transfer, got %s", model.State())
	}
	if got := model.LastDestination(); got != "/uploads/report.txt" {
		t.Fatalf("unexpected destination %q", got)
	}
	if clipboard.Copied != "/uploads/report.txt" {
		t.Fatalf("expected copied destination, got %q", clipboard.Copied)
	}
	if !strings.Contains(model.View(), "/uploads/report.txt") {
		t.Fatalf("view should display computed destination:\n%s", model.View())
	}
}

func TestDropInputAcceptsTerminalEscapedDroppedPath(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "Application Support", "CleanShot")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "screenshot.png")
	if err := os.WriteFile(file, []byte("png"), 0o600); err != nil {
		t.Fatal(err)
	}
	transfer := &fakeTransfer{}
	model := tui.NewModel(session.Start{
		Config: session.Config{Remotes: []session.Remote{{Name: "cb", Host: "cb", Destination: "/uploads"}}},
	}, tui.Services{Transferer: transfer, Clipboard: &fakeClipboard{}})

	escaped := strings.ReplaceAll(file, " ", `\ `)
	model = submitPath(t, model, escaped)

	if model.State() != tui.StateUpload {
		t.Fatalf("expected upload state, got %s:\n%s", model.State(), model.View())
	}
	if transfer.Request.LocalPath != file {
		t.Fatalf("expected normalized local path %q, got %q", file, transfer.Request.LocalPath)
	}
	if got := model.LastDestination(); got != "/uploads/screenshot.png" {
		t.Fatalf("unexpected destination %q", got)
	}
}

func TestClipboardFailureIsWarningNotTransferFailure(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "report.txt")
	if err := os.WriteFile(file, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	transfer := &fakeTransfer{}
	clipboard := &fakeClipboard{Err: errors.New("no clipboard")}
	model := tui.NewModel(session.Start{
		Config: session.Config{Remotes: []session.Remote{{Name: "cb", Host: "cb", Destination: "/uploads"}}},
	}, tui.Services{Transferer: transfer, Clipboard: clipboard})

	model = submitPath(t, model, file)
	model = drainTransfer(t, model, transfer, session.TransferEvent{Done: true})

	if model.Summary().Successes != 1 || model.Summary().Failures != 0 {
		t.Fatalf("clipboard failure should not fail transfer: %#v", model.Summary())
	}
	view := model.View()
	for _, want := range []string{"uploaded /uploads/report.txt", "clipboard warning: no clipboard"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view missing %q:\n%s", want, view)
		}
	}
}

func TestTransferFailureReturnsToDropAndCountsFailure(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "report.txt")
	if err := os.WriteFile(file, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	transfer := &fakeTransfer{}
	model := tui.NewModel(session.Start{
		Config: session.Config{Remotes: []session.Remote{{Name: "cb", Host: "cb", Destination: "/uploads"}}},
	}, tui.Services{Transferer: transfer, Clipboard: &fakeClipboard{}})

	model = submitPath(t, model, file)
	if model.State() != tui.StateUpload {
		t.Fatalf("expected upload state, got %s", model.State())
	}
	model = drainTransfer(t, model, transfer, session.TransferEvent{Output: "rsync failed\n"})
	model = drainTransfer(t, model, transfer, session.TransferEvent{Done: true, Err: errors.New("exit status 12")})

	if model.State() != tui.StateDrop {
		t.Fatalf("expected drop state after failure, got %s", model.State())
	}
	if model.Summary().Failures != 1 {
		t.Fatalf("expected one failure, got %#v", model.Summary())
	}
	view := model.View()
	for _, want := range []string{"upload failed: exit status 12", "Output:", "rsync failed"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected failure status to include %q:\n%s", want, view)
		}
	}
}

func TestEscapeCancelsUploadAndKeepsSessionOpen(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "report.txt")
	if err := os.WriteFile(file, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	transfer := &fakeTransfer{}
	model := tui.NewModel(session.Start{
		Config: session.Config{Remotes: []session.Remote{{Name: "cb", Host: "cb", Destination: "/uploads"}}},
	}, tui.Services{Transferer: transfer, Clipboard: &fakeClipboard{}})

	model = submitPath(t, model, file)
	model = update(t, model, tea.KeyMsg{Type: tea.KeyEsc})
	if !transfer.Canceled() {
		t.Fatal("expected transfer cancellation")
	}
	model = drainTransfer(t, model, transfer, session.TransferEvent{Done: true, Err: session.ErrTransferCanceled})

	if model.State() != tui.StateDrop {
		t.Fatalf("expected drop state after cancel, got %s", model.State())
	}
	if model.Summary().Canceled != 1 {
		t.Fatalf("expected canceled summary, got %#v", model.Summary())
	}
}

func TestQDuringUploadRequiresConfirmation(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "report.txt")
	if err := os.WriteFile(file, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	transfer := &fakeTransfer{}
	model := tui.NewModel(session.Start{
		Config: session.Config{Remotes: []session.Remote{{Name: "cb", Host: "cb", Destination: "/uploads"}}},
	}, tui.Services{Transferer: transfer, Clipboard: &fakeClipboard{}})

	model = submitPath(t, model, file)
	model = update(t, model, key("q"))
	if model.State() != tui.StateConfirmQuit {
		t.Fatalf("expected confirm quit state, got %s", model.State())
	}
	if transfer.Canceled() {
		t.Fatal("q should not cancel before confirmation")
	}

	model = update(t, model, key("n"))
	if model.State() != tui.StateUpload {
		t.Fatalf("expected upload state after declining quit, got %s", model.State())
	}

	model = update(t, model, key("q"))
	model = update(t, model, key("y"))
	if !transfer.Canceled() {
		t.Fatal("confirming quit should cancel upload")
	}
	model = drainTransfer(t, model, transfer, session.TransferEvent{Done: true, Err: session.ErrTransferCanceled})
	if !model.Quitting() {
		t.Fatal("model should quit after confirmed cancellation")
	}
}

func update(t *testing.T, model tui.Model, msg tea.Msg) tui.Model {
	t.Helper()
	updated, _ := model.Update(msg)
	next, ok := updated.(tui.Model)
	if !ok {
		t.Fatalf("unexpected model type %T", updated)
	}
	return next
}

func drainTransfer(t *testing.T, model tui.Model, transfer *fakeTransfer, event session.TransferEvent) tui.Model {
	t.Helper()
	transfer.Events <- event
	updated, cmd := model.Update(tui.TransferEventMsg{Event: event})
	next, ok := updated.(tui.Model)
	if !ok {
		t.Fatalf("unexpected model type %T", updated)
	}
	_ = cmd
	return next
}

func submitPath(t *testing.T, model tui.Model, path string) tui.Model {
	t.Helper()
	for _, r := range path {
		model = update(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return update(t, model, key("enter"))
}

func key(value string) tea.KeyMsg {
	switch value {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(value)}
	}
}

func configWithRemotes() session.Config {
	return session.Config{Remotes: []session.Remote{
		{Name: "cb", Host: "cb", Destination: "/tmp"},
		{Name: "files", Host: "files.example.com", User: "deploy", Destination: "/var/tmp"},
	}}
}

type fakeTransfer struct {
	Events  chan session.TransferEvent
	Context context.Context
	Request session.TransferRequest
}

func (f *fakeTransfer) Begin(ctx context.Context, req session.TransferRequest) <-chan session.TransferEvent {
	f.Events = make(chan session.TransferEvent, 8)
	f.Context = ctx
	f.Request = req
	return f.Events
}

func (f *fakeTransfer) Canceled() bool {
	return f.Context != nil && errors.Is(f.Context.Err(), context.Canceled)
}

type fakeClipboard struct {
	Copied string
	Err    error
}

func (f *fakeClipboard) Copy(value string) error {
	f.Copied = value
	return f.Err
}
