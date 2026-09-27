package tui

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/flexdinesh/ssh-drop/internal/clipboard"
	"github.com/flexdinesh/ssh-drop/internal/session"
	"github.com/flexdinesh/ssh-drop/internal/transfer"
)

type State int

const (
	StateRemotePicker State = iota
	StateDrop
	StatePassword
	StateUpload
	StateConfirmQuit
	StateConnecting
	StateConnectionFailed
)

func (s State) String() string {
	switch s {
	case StateRemotePicker:
		return "remote-picker"
	case StateDrop:
		return "drop"
	case StatePassword:
		return "password"
	case StateUpload:
		return "upload"
	case StateConfirmQuit:
		return "confirm-quit"
	case StateConnecting:
		return "connecting"
	case StateConnectionFailed:
		return "connection-failed"
	default:
		return "unknown"
	}
}

type statusKind int

const (
	statusIdle statusKind = iota
	statusSyncing
	statusCopying
	statusSuccess
	statusWarning
	statusError
	statusCanceled
)

type Services struct {
	Lstat      func(string) (os.FileInfo, error)
	Transferer Transferer
	Clipboard  Clipboard
	Connector  Connector
}

type Connector interface {
	Connect(context.Context, session.Remote, string) (string, error)
	Close(session.Remote, string)
}

type ConnectionMsg struct {
	Generation  int
	ControlPath string
	Remote      session.Remote
	Err         error
}

type Transferer interface {
	Begin(context.Context, session.TransferRequest) <-chan session.TransferEvent
}

type Clipboard interface {
	Copy(context.Context, string) error
}

type TransferEventMsg struct {
	Event session.TransferEvent
}

type Model struct {
	start                session.Start
	state                State
	cursor               int
	selected             int
	input                textinput.Model
	passwordInput        textinput.Model
	passwordError        string
	status               string
	statusKind           statusKind
	lastDestination      string
	currentRequest       session.TransferRequest
	transferEvents       <-chan session.TransferEvent
	transferContext      context.Context
	cancelTransfer       context.CancelFunc
	uploadOutput         string
	services             Services
	summary              session.Summary
	quitting             bool
	quitAfterCancel      bool
	width                int
	height               int
	controlPath          string
	connectionGeneration int
	connectionContext    context.Context
	cancelConnection     context.CancelFunc
	passwordAttempted    bool
	controlRemote        session.Remote
	connections          *connectionOwner
	clipboard            *clipboardOwner
	clipboardGeneration  int
	cancelClipboard      context.CancelFunc
}

func NewModel(start session.Start, services Services) Model {
	if services.Lstat == nil {
		services.Lstat = os.Lstat
	}
	if services.Transferer == nil {
		services.Transferer = transfer.Runner{}
	}
	if services.Clipboard == nil {
		services.Clipboard = clipboard.Copier{}
	}
	if services.Connector == nil {
		services.Connector = transfer.SSHConnection{}
	}
	input := textinput.New()
	input.Placeholder = "Drop or paste a local file path"
	input.Focus()
	input.CharLimit = 4096
	input.Width = 68
	input.PromptStyle = selectedStyle
	input.TextStyle = titleStyle.Bold(false)
	input.PlaceholderStyle = mutedStyle
	passwordInput := textinput.New()
	passwordInput.EchoMode = textinput.EchoPassword
	passwordInput.EchoCharacter = '*'
	passwordInput.CharLimit = 1024
	passwordInput.Width = 28
	passwordInput.Focus()
	passwordInput.PromptStyle = selectedStyle
	passwordInput.TextStyle = titleStyle.Bold(false)

	model := Model{
		start:         start,
		state:         StateRemotePicker,
		selected:      -1,
		input:         input,
		passwordInput: passwordInput,
		services:      services,
		connections:   &connectionOwner{connector: services.Connector, paths: make(map[string]session.Remote)},
		clipboard:     newClipboardOwner(services.Clipboard),
		width:         80,
		height:        24,
	}
	model.input.Width = model.inputTextWidth()
	if start.PreselectedRemote != "" {
		for i, remote := range start.Config.Remotes {
			if remote.Name == start.PreselectedRemote {
				model.selected = i
				model.cursor = i
				model.state = StateDrop
				break
			}
		}
	} else if len(start.Config.Remotes) == 1 {
		model.selected = 0
		model.state = StateDrop
	}
	if model.selected >= 0 {
		model.prepareConnection()
	}
	return model
}

func (m *Model) prepareConnection() {
	m.invalidateClipboard()
	if m.cancelConnection != nil {
		m.cancelConnection()
	}
	m.connectionContext, m.cancelConnection = context.WithCancel(context.Background())
	m.connectionGeneration++
	m.state = StateConnecting
}

func (m Model) connectionCommand(password string) tea.Cmd {
	return func() tea.Msg {
		controlPath, err := m.connections.connect(m.connectionContext, m.SelectedRemote(), password)
		return ConnectionMsg{Generation: m.connectionGeneration, ControlPath: controlPath, Remote: m.SelectedRemote(), Err: err}
	}
}

func (m Model) beginConnection(password string) (tea.Model, tea.Cmd) {
	oldPath := m.controlPath
	m.controlPath = ""
	m.passwordAttempted = password != ""
	m.prepareConnection()
	connect := m.connectionCommand(password)
	return m, func() tea.Msg {
		m.connections.close(oldPath)
		return connect()
	}
}

func (m Model) updateConnection(msg ConnectionMsg) (tea.Model, tea.Cmd) {
	if msg.Generation != m.connectionGeneration {
		return m, func() tea.Msg { m.connections.close(msg.ControlPath); return nil }
	}
	if msg.Err != nil && m.cancelConnection != nil {
		m.cancelConnection()
		m.cancelConnection = nil
	}
	if errors.Is(msg.Err, session.ErrPasswordRequired) {
		m.passwordError = ""
		if m.state == StateConnecting && m.passwordAttempted {
			m.passwordError = "Password rejected. Try again."
		}
		m.state = StatePassword
		return m, textinput.Blink
	}
	if msg.Err != nil {
		m.status = msg.Err.Error()
		m.statusKind = statusError
		m.state = StateConnectionFailed
		return m, nil
	}
	m.controlPath = msg.ControlPath
	m.controlRemote = msg.Remote
	m.passwordAttempted = false
	m.status = ""
	m.statusKind = statusIdle
	m.state = StateDrop
	return m, textinput.Blink
}

func (m Model) updateConnectionKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyCtrlC || msg.String() == "q" {
		if m.cancelConnection != nil {
			m.cancelConnection()
		}
		m.quitting = true
		return m, tea.Quit
	}
	if msg.Type == tea.KeyEsc || msg.String() == "r" {
		if m.cancelConnection != nil {
			m.cancelConnection()
		}
		m.connectionGeneration++
		m.state = StateRemotePicker
		return m, nil
	}
	if m.state == StateConnectionFailed && msg.Type == tea.KeyEnter {
		return m.beginConnection("")
	}
	return m, nil
}

func Run(start session.Start) (session.Summary, error) {
	initial := NewModel(start, Services{})
	program := tea.NewProgram(initial)
	final, err := program.Run()
	model, ok := final.(Model)
	if !ok {
		model = initial
	}
	model.shutdown()
	if err != nil {
		return session.Summary{}, err
	}
	if !ok {
		return session.Summary{}, fmt.Errorf("unexpected final model %T", final)
	}
	return model.summary, nil
}

func (m Model) Init() tea.Cmd {
	if m.state == StateConnecting {
		return m.connectionCommand("")
	}
	return tea.Batch(textinput.Blink, tea.ClearScreen)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.setSize(msg.Width, msg.Height)
		return m, tea.ClearScreen
	case tea.KeyMsg:
		switch m.state {
		case StateRemotePicker:
			return m.updatePicker(msg)
		case StateDrop:
			return m.updateDrop(msg)
		case StatePassword:
			return m.updatePassword(msg)
		case StateUpload:
			return m.updateUpload(msg)
		case StateConfirmQuit:
			return m.updateConfirmQuit(msg)
		case StateConnecting, StateConnectionFailed:
			return m.updateConnectionKey(msg)
		}
	case ConnectionMsg:
		return m.updateConnection(msg)
	case TransferEventMsg:
		return m.updateTransfer(msg.Event)
	case ClipboardMsg:
		return m.updateClipboard(msg)
	}
	var cmd tea.Cmd
	if m.state == StateDrop {
		m.input, cmd = m.input.Update(msg)
	}
	if m.state == StatePassword {
		m.passwordInput, cmd = m.passwordInput.Update(msg)
	}
	return m, cmd
}

func (m Model) State() State {
	return m.state
}

func (m Model) SelectedRemote() session.Remote {
	if m.selected < 0 || m.selected >= len(m.start.Config.Remotes) {
		return session.Remote{}
	}
	return m.start.Config.Remotes[m.selected]
}

func (m Model) LastDestination() string {
	return m.lastDestination
}

func (m Model) Summary() session.Summary {
	return m.summary
}

func (m Model) Quitting() bool {
	return m.quitting
}

func (m Model) updatePicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		m.quitting = true
		return m, tea.Quit
	case tea.KeyRunes:
		if string(msg.Runes) == "q" {
			m.quitting = true
			return m, tea.Quit
		}
	case tea.KeyUp:
		if m.cursor > 0 {
			m.cursor--
		}
	case tea.KeyDown:
		if m.cursor < len(m.start.Config.Remotes)-1 {
			m.cursor++
		}
	case tea.KeyEnter:
		if len(m.start.Config.Remotes) > 0 {
			m.selected = m.cursor
			m.status = ""
			m.statusKind = statusIdle
			return m.beginConnection("")
		}
	}
	return m, nil
}

func (m Model) updateDrop(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		m.quitting = true
		return m, tea.Quit
	case tea.KeyEsc:
		m.input.SetValue("")
		return m, nil
	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "q":
			if m.input.Value() == "" {
				m.quitting = true
				return m, tea.Quit
			}
		case "r":
			if m.input.Value() == "" {
				m.invalidateClipboard()
				m.state = StateRemotePicker
				m.cursor = m.selected
				m.statusKind = statusIdle
				return m, nil
			}
		}
	case tea.KeyEnter:
		return m.submitPath()
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) updatePassword(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.passwordInput.SetValue("")
		m.passwordError = ""
		m.state = StateRemotePicker
		return m, nil
	case tea.KeyCtrlC:
		m.quitting = true
		return m, tea.Quit
	case tea.KeyEnter:
		if m.passwordInput.Value() == "" {
			m.passwordError = "enter password"
			return m, nil
		}
		password := m.passwordInput.Value()
		m.passwordInput.SetValue("")
		m.passwordError = ""
		return m.beginConnection(password)
	}
	var cmd tea.Cmd
	m.passwordInput, cmd = m.passwordInput.Update(msg)
	return m, cmd
}

func (m Model) updateUpload(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		if m.cancelTransfer != nil {
			m.cancelTransfer()
		}
		return m, nil
	case tea.KeyCtrlC:
		if m.cancelTransfer != nil {
			m.cancelTransfer()
		}
		m.quitAfterCancel = true
		return m, nil
	case tea.KeyRunes:
		if string(msg.Runes) == "q" {
			m.state = StateConfirmQuit
			return m, nil
		}
	}
	return m, nil
}

func (m Model) updateConfirmQuit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		if m.cancelTransfer != nil {
			m.cancelTransfer()
		}
		m.quitAfterCancel = true
		return m, nil
	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "y", "Y":
			if m.cancelTransfer != nil {
				m.cancelTransfer()
			}
			m.quitAfterCancel = true
			return m, nil
		case "n", "N":
			m.state = StateUpload
			return m, nil
		}
	}
	return m, nil
}

func (m Model) updateTransfer(event session.TransferEvent) (tea.Model, tea.Cmd) {
	if event.Output != "" {
		m.uploadOutput += event.Output
		if len(m.uploadOutput) > 16384 {
			m.uploadOutput = m.uploadOutput[len(m.uploadOutput)-16384:]
		}
	}
	if !event.Done {
		return m, waitForTransfer(m.transferContext, m.transferEvents)
	}
	if m.cancelTransfer != nil {
		m.cancelTransfer()
		m.cancelTransfer = nil
	}
	if errors.Is(event.Err, session.ErrTransferCanceled) {
		m.summary.Canceled++
		m.status = fmt.Sprintf("canceled upload to %s", m.currentRequest.DestinationPath)
		m.statusKind = statusCanceled
		m.state = StateDrop
	} else if event.Err != nil {
		m.summary.Failures++
		m.status = transferFailureStatus(event.Err, m.uploadOutput)
		m.statusKind = statusError
		m.state = StateDrop
		if errors.Is(event.Err, session.ErrConnectionLost) {
			m.state = StateConnectionFailed
			m.status = "Connection lost. Press enter to reconnect."
		}
	} else {
		m.input.SetValue("")
		m.summary.Successes++
		m.summary.SuccessfulDestinations = append(m.summary.SuccessfulDestinations, m.currentRequest.DestinationPath)
		m.status = transferResultStatus(m.currentRequest.LocalPath, m.currentRequest.DestinationPath)
		m.statusKind = statusCopying
		m.state = StateDrop
		copyPath := m.copyCommand()
		if m.quitAfterCancel {
			m.quitting = true
			return m, func() tea.Msg { copyPath(); return tea.Quit() }
		}
		return m, copyPath
	}
	if m.quitAfterCancel {
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}

func (m *Model) submitPath() (tea.Model, tea.Cmd) {
	localPath := strings.TrimSpace(m.input.Value())
	if localPath == "" {
		m.status = "enter a file path"
		m.statusKind = statusError
		return *m, nil
	}
	if strings.Contains(localPath, "\n") {
		m.status = "enter one plain local file path"
		m.statusKind = statusError
		return *m, nil
	}
	localPath, info, err := resolveLocalPath(localPath, m.services.Lstat)
	if err != nil {
		if os.IsNotExist(err) {
			m.status = fmt.Sprintf("%s does not exist", localPath)
		} else {
			m.status = err.Error()
		}
		m.statusKind = statusError
		return *m, nil
	}
	if !info.Mode().IsRegular() {
		m.status = fmt.Sprintf("%s is not a regular file", localPath)
		m.statusKind = statusError
		return *m, nil
	}
	localPath, err = filepath.Abs(localPath)
	if err != nil {
		m.status = err.Error()
		m.statusKind = statusError
		return *m, nil
	}
	remote := m.SelectedRemote()
	destination := path.Join(remote.Destination, filepath.Base(localPath))
	m.lastDestination = destination
	m.currentRequest = session.TransferRequest{
		LocalPath:       localPath,
		DestinationDir:  remote.Destination,
		DestinationPath: destination,
		Remote:          remote,
		ControlPath:     m.controlPath,
	}
	return m.startTransfer()
}

func (m *Model) startTransfer() (tea.Model, tea.Cmd) {
	m.invalidateClipboard()
	ctx, cancel := context.WithCancel(context.Background())
	m.transferContext = ctx
	m.cancelTransfer = cancel
	m.transferEvents = m.services.Transferer.Begin(ctx, m.currentRequest)
	m.uploadOutput = ""
	m.status = ""
	m.statusKind = statusSyncing
	m.state = StateUpload
	return *m, waitForTransfer(ctx, m.transferEvents)
}

func transferResultStatus(source string, destination string) string {
	return fmt.Sprintf("Destination: %s\nSource: %s", destination, source)
}

func ensureMinLines(value string, minLines int) string {
	if minLines <= 0 {
		return value
	}
	lineCount := 1
	if value != "" {
		lineCount = strings.Count(value, "\n") + 1
	}
	for lineCount < minLines {
		value += "\n"
		lineCount++
	}
	return value
}

func isErrorStatus(status string) bool {
	return strings.Contains(status, "does not exist") ||
		strings.Contains(status, "regular file") ||
		strings.HasPrefix(status, "upload failed:")
}

func transferFailureStatus(err error, output string) string {
	status := fmt.Sprintf("upload failed: %v", err)
	output = strings.TrimSpace(output)
	if output == "" {
		return status
	}
	return status + "\n\nOutput:\n" + output
}

func resolveLocalPath(input string, stat func(string) (os.FileInfo, error)) (string, os.FileInfo, error) {
	info, err := stat(input)
	if err == nil {
		return input, info, nil
	}

	normalized := normalizeDroppedPath(input)
	if normalized == input {
		return input, nil, err
	}

	normalizedInfo, normalizedErr := stat(normalized)
	if normalizedErr == nil {
		return normalized, normalizedInfo, nil
	}
	return normalized, nil, normalizedErr
}

func normalizeDroppedPath(input string) string {
	input = strings.TrimSpace(input)
	input = trimMatchingQuotes(input)
	if strings.HasPrefix(input, "file://") {
		if parsed, err := url.Parse(input); err == nil && parsed.Path != "" {
			input = parsed.Path
		}
	}

	var b strings.Builder
	b.Grow(len(input))
	escaped := false
	for _, r := range input {
		if escaped {
			b.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		b.WriteRune(r)
	}
	if escaped {
		b.WriteRune('\\')
	}
	return b.String()
}

func trimMatchingQuotes(input string) string {
	if len(input) < 2 {
		return input
	}
	first := input[0]
	last := input[len(input)-1]
	if (first == '\'' || first == '"') && first == last {
		return input[1 : len(input)-1]
	}
	return input
}

func waitForTransfer(ctx context.Context, events <-chan session.TransferEvent) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-events
		if !ok {
			if errors.Is(ctx.Err(), context.Canceled) {
				return TransferEventMsg{Event: session.TransferEvent{Done: true, Err: session.ErrTransferCanceled}}
			}
			return TransferEventMsg{Event: session.TransferEvent{Done: true, Err: errors.New("transfer ended without a result")}}
		}
		if event.Done {
			// A terminal message is not proof that subprocess cleanup finished.
			for range events {
			}
		}
		return TransferEventMsg{Event: event}
	}
}
