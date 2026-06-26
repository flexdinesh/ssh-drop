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
	"github.com/charmbracelet/lipgloss"
	"github.com/dineshpandiyan/ssh-drop/internal/clipboard"
	"github.com/dineshpandiyan/ssh-drop/internal/session"
	"github.com/dineshpandiyan/ssh-drop/internal/transfer"
)

type State int

const (
	StateRemotePicker State = iota
	StateDrop
	StateUpload
	StateConfirmQuit
)

func (s State) String() string {
	switch s {
	case StateRemotePicker:
		return "remote-picker"
	case StateDrop:
		return "drop"
	case StateUpload:
		return "upload"
	case StateConfirmQuit:
		return "confirm-quit"
	default:
		return "unknown"
	}
}

type Services struct {
	Stat       func(string) (os.FileInfo, error)
	Transferer Transferer
	Clipboard  Clipboard
}

type Transferer interface {
	Begin(context.Context, session.TransferRequest) <-chan session.TransferEvent
}

type Clipboard interface {
	Copy(string) error
}

type TransferEventMsg struct {
	Event session.TransferEvent
}

type Model struct {
	start           session.Start
	state           State
	cursor          int
	selected        int
	input           textinput.Model
	status          string
	lastDestination string
	currentRequest  session.TransferRequest
	transferEvents  <-chan session.TransferEvent
	cancelTransfer  context.CancelFunc
	uploadOutput    string
	services        Services
	summary         session.Summary
	quitting        bool
	quitAfterCancel bool
}

var (
	titleStyle    = lipgloss.NewStyle().Bold(true)
	selectedStyle = lipgloss.NewStyle().Bold(true)
	mutedStyle    = lipgloss.NewStyle().Faint(true)
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
)

func NewModel(start session.Start, services Services) Model {
	if services.Stat == nil {
		services.Stat = os.Stat
	}
	if services.Transferer == nil {
		services.Transferer = transfer.Runner{}
	}
	if services.Clipboard == nil {
		services.Clipboard = clipboard.Copier{}
	}
	input := textinput.New()
	input.Placeholder = "Drop or paste a plain local file path"
	input.Focus()
	input.CharLimit = 4096
	input.Width = 80

	model := Model{
		start:    start,
		state:    StateRemotePicker,
		selected: -1,
		input:    input,
		services: services,
	}
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
	return model
}

func Run(start session.Start) (session.Summary, error) {
	program := tea.NewProgram(NewModel(start, Services{}))
	final, err := program.Run()
	if err != nil {
		return session.Summary{}, err
	}
	model, ok := final.(Model)
	if !ok {
		return session.Summary{}, fmt.Errorf("unexpected final model %T", final)
	}
	return model.summary, nil
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case StateRemotePicker:
			return m.updatePicker(msg)
		case StateDrop:
			return m.updateDrop(msg)
		case StateUpload:
			return m.updateUpload(msg)
		case StateConfirmQuit:
			return m.updateConfirmQuit(msg)
		}
	case TransferEventMsg:
		return m.updateTransfer(msg.Event)
	}
	return m, nil
}

func (m Model) View() string {
	var b strings.Builder
	switch m.state {
	case StateRemotePicker:
		b.WriteString(titleStyle.Render("Choose remote"))
		b.WriteString("\n\n")
		for i, remote := range m.start.Config.Remotes {
			cursor := "  "
			lineStyle := lipgloss.NewStyle()
			if i == m.cursor {
				cursor = "> "
				lineStyle = selectedStyle
			}
			b.WriteString(lineStyle.Render(fmt.Sprintf("%s%-12s %-32s -> %s", cursor, remote.Name, remote.Target(), remote.Destination)))
			b.WriteString("\n")
		}
		b.WriteString("\n")
		b.WriteString(mutedStyle.Render("enter select · q quit"))
	case StateDrop:
		remote := m.SelectedRemote()
		b.WriteString(titleStyle.Render("Drop file"))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("Remote: %s  %s  -> %s\n\n", remote.Name, remote.Target(), remote.Destination))
		b.WriteString(m.input.View())
		b.WriteString("\n\n")
		if m.status != "" {
			if isErrorStatus(m.status) {
				b.WriteString(errorStyle.Render(m.status))
			} else {
				b.WriteString(m.status)
			}
			b.WriteString("\n\n")
		}
		b.WriteString(mutedStyle.Render("enter upload · r remote · q quit · ctrl+c quit"))
	case StateUpload, StateConfirmQuit:
		remote := m.currentRequest.Remote
		b.WriteString(titleStyle.Render("Uploading"))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("Remote: %s  %s  -> %s\n", remote.Name, remote.Target(), remote.Destination))
		b.WriteString(fmt.Sprintf("Destination: %s\n\n", m.currentRequest.DestinationPath))
		if m.uploadOutput != "" {
			b.WriteString(m.uploadOutput)
			if !strings.HasSuffix(m.uploadOutput, "\n") {
				b.WriteString("\n")
			}
			b.WriteString("\n")
		}
		if m.state == StateConfirmQuit {
			b.WriteString(errorStyle.Render("Cancel upload and quit? y/n"))
		} else {
			b.WriteString(mutedStyle.Render("esc cancel · q quit · ctrl+c quit"))
		}
	}
	return b.String()
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
			m.state = StateDrop
			m.status = ""
		}
	}
	return m, nil
}

func (m Model) updateDrop(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		m.quitting = true
		return m, tea.Quit
	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "q":
			if m.input.Value() == "" {
				m.quitting = true
				return m, tea.Quit
			}
		case "r":
			if m.input.Value() == "" {
				m.state = StateRemotePicker
				m.cursor = m.selected
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
		return m, waitForTransfer(m.transferEvents)
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
		return m, waitForTransfer(m.transferEvents)
	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "y", "Y":
			if m.cancelTransfer != nil {
				m.cancelTransfer()
			}
			m.quitAfterCancel = true
			return m, waitForTransfer(m.transferEvents)
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
	}
	if !event.Done {
		return m, waitForTransfer(m.transferEvents)
	}
	m.input.SetValue("")
	m.cancelTransfer = nil
	if errors.Is(event.Err, session.ErrTransferCanceled) {
		m.summary.Canceled++
		m.status = fmt.Sprintf("canceled upload to %s", m.currentRequest.DestinationPath)
		m.state = StateDrop
		if m.quitAfterCancel {
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil
	}
	if event.Err != nil {
		m.summary.Failures++
		m.status = transferFailureStatus(event.Err, m.uploadOutput)
		m.state = StateDrop
		return m, nil
	}
	m.summary.Successes++
	m.summary.SuccessfulDestinations = append(m.summary.SuccessfulDestinations, m.currentRequest.DestinationPath)
	if err := m.services.Clipboard.Copy(m.currentRequest.DestinationPath); err != nil {
		m.status = fmt.Sprintf("uploaded %s\nclipboard warning: %v", m.currentRequest.DestinationPath, err)
	} else {
		m.status = fmt.Sprintf("uploaded %s\ncopied %s", m.currentRequest.DestinationPath, m.currentRequest.DestinationPath)
	}
	m.state = StateDrop
	return m, nil
}

func (m *Model) submitPath() (tea.Model, tea.Cmd) {
	localPath := strings.TrimSpace(m.input.Value())
	if localPath == "" {
		m.status = "enter a file path"
		return *m, nil
	}
	if strings.Contains(localPath, "\n") {
		m.status = "enter one plain local file path"
		return *m, nil
	}
	localPath, info, err := resolveLocalPath(localPath, m.services.Stat)
	if err != nil {
		if os.IsNotExist(err) {
			m.status = fmt.Sprintf("%s does not exist", localPath)
		} else {
			m.status = err.Error()
		}
		return *m, nil
	}
	if !info.Mode().IsRegular() {
		m.status = fmt.Sprintf("%s is not a regular file", localPath)
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
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelTransfer = cancel
	m.transferEvents = m.services.Transferer.Begin(ctx, m.currentRequest)
	m.uploadOutput = ""
	m.status = ""
	m.state = StateUpload
	return *m, waitForTransfer(m.transferEvents)
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

func waitForTransfer(events <-chan session.TransferEvent) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-events
		if !ok {
			return TransferEventMsg{Event: session.TransferEvent{Done: true}}
		}
		return TransferEventMsg{Event: event}
	}
}
