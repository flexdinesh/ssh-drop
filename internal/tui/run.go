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
	StatePassword
	StateUpload
	StateConfirmQuit
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
	default:
		return "unknown"
	}
}

type statusKind int

const (
	statusIdle statusKind = iota
	statusSyncing
	statusSuccess
	statusWarning
	statusError
	statusCanceled
)

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
	passwordInput   textinput.Model
	passwordError   string
	status          string
	statusKind      statusKind
	lastDestination string
	currentRequest  session.TransferRequest
	transferEvents  <-chan session.TransferEvent
	cancelTransfer  context.CancelFunc
	uploadOutput    string
	services        Services
	summary         session.Summary
	quitting        bool
	quitAfterCancel bool
	width           int
	height          int
}

var (
	titleStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	sectionStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	selectedStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	mutedStyle      = lipgloss.NewStyle().Faint(true)
	errorStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	successStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	warningStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	syncingStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	statusNameStyle = lipgloss.NewStyle().Bold(true)
	inputBoxStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("8")).Padding(0, 1)
	panelStyle      = lipgloss.NewStyle().Padding(1, 2)
	ruleStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
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
	input.Width = 68
	passwordInput := textinput.New()
	passwordInput.EchoMode = textinput.EchoPassword
	passwordInput.EchoCharacter = '*'
	passwordInput.CharLimit = 1024
	passwordInput.Width = 28
	passwordInput.Focus()

	model := Model{
		start:         start,
		state:         StateRemotePicker,
		selected:      -1,
		input:         input,
		passwordInput: passwordInput,
		services:      services,
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
		b.WriteString(m.header("Choose remote"))
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
		b.WriteString(m.renderDropView())
	case StatePassword:
		b.WriteString(m.renderPasswordView())
	case StateUpload, StateConfirmQuit:
		b.WriteString(m.renderDropView())
	}
	return panelStyle.Width(m.viewWidth()).Render(b.String())
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
			m.statusKind = statusIdle
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
		m.currentRequest = session.TransferRequest{}
		m.passwordInput.SetValue("")
		m.passwordError = ""
		m.state = StateDrop
		return m, nil
	case tea.KeyCtrlC:
		m.quitting = true
		return m, tea.Quit
	case tea.KeyEnter:
		if m.passwordInput.Value() == "" {
			m.passwordError = "enter password"
			return m, nil
		}
		m.currentRequest.Password = m.passwordInput.Value()
		m.passwordInput.SetValue("")
		m.passwordError = ""
		return m.startTransfer()
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
		m.statusKind = statusCanceled
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
		m.statusKind = statusError
		m.state = StateDrop
		return m, nil
	}
	m.summary.Successes++
	m.summary.SuccessfulDestinations = append(m.summary.SuccessfulDestinations, m.currentRequest.DestinationPath)
	if err := m.services.Clipboard.Copy(m.currentRequest.DestinationPath); err != nil {
		m.status = transferResultStatus(m.currentRequest.LocalPath, m.currentRequest.DestinationPath) + fmt.Sprintf("\nclipboard warning: %v", err)
		m.statusKind = statusWarning
	} else {
		m.status = transferResultStatus(m.currentRequest.LocalPath, m.currentRequest.DestinationPath)
		m.statusKind = statusSuccess
	}
	m.state = StateDrop
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
	localPath, info, err := resolveLocalPath(localPath, m.services.Stat)
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
	remote := m.SelectedRemote()
	destination := path.Join(remote.Destination, filepath.Base(localPath))
	m.lastDestination = destination
	m.currentRequest = session.TransferRequest{
		LocalPath:       localPath,
		DestinationDir:  remote.Destination,
		DestinationPath: destination,
		Remote:          remote,
	}
	if remoteNeedsPassword(remote) {
		m.passwordInput.SetValue("")
		m.passwordError = ""
		m.state = StatePassword
		return *m, nil
	}
	return m.startTransfer()
}

func (m *Model) startTransfer() (tea.Model, tea.Cmd) {
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelTransfer = cancel
	m.transferEvents = m.services.Transferer.Begin(ctx, m.currentRequest)
	m.uploadOutput = ""
	m.status = ""
	m.statusKind = statusSyncing
	m.state = StateUpload
	return *m, waitForTransfer(m.transferEvents)
}

func (m *Model) setSize(width int, height int) {
	if width <= 0 {
		width = m.width
	}
	m.width = width
	if height > 0 {
		m.height = height
	}
	m.input.Width = m.inputTextWidth()
}

func (m Model) viewWidth() int {
	return max(40, m.width)
}

func (m Model) innerWidth() int {
	return max(36, m.viewWidth()-4)
}

func (m Model) innerHeight() int {
	return max(12, m.height-2)
}

func (m Model) inputBoxWidth() int {
	return max(24, m.innerWidth()-4)
}

func (m Model) inputContentWidth() int {
	return max(20, m.inputBoxWidth()-4)
}

func (m Model) inputTextWidth() int {
	return max(18, m.inputContentWidth()-lipgloss.Width(m.input.Prompt))
}

func (m Model) renderDropView() string {
	var b strings.Builder
	remote := m.remoteForDropView()

	b.WriteString(m.header("Drop file"))
	b.WriteString("\n\n")
	b.WriteString(sectionStyle.Render("Remote"))
	b.WriteString("\n")
	b.WriteString(renderRemoteDetails(remote))
	b.WriteString("\n\n")
	b.WriteString(m.renderStatus())
	b.WriteString("\n\n")
	b.WriteString(sectionStyle.Render("File"))
	b.WriteString("\n")
	b.WriteString(m.renderInput())
	b.WriteString("\n\n")
	b.WriteString(m.renderTransferDetails())
	b.WriteString("\n\n")
	if m.uploadOutput != "" {
		b.WriteString(m.uploadOutput)
		if !strings.HasSuffix(m.uploadOutput, "\n") {
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	b.WriteString(m.renderFooter())
	return b.String()
}

func (m Model) renderPasswordView() string {
	return lipgloss.Place(
		m.innerWidth(),
		m.innerHeight(),
		lipgloss.Center,
		lipgloss.Center,
		m.renderPasswordPrompt(),
	)
}

func (m Model) remoteForDropView() session.Remote {
	if m.state == StatePassword || m.state == StateUpload || m.state == StateConfirmQuit {
		return m.currentRequest.Remote
	}
	return m.SelectedRemote()
}

func (m Model) renderTransferDetails() string {
	if m.state == StateUpload || m.state == StateConfirmQuit {
		return transferResultStatus(m.currentRequest.LocalPath, m.currentRequest.DestinationPath)
	}
	if m.status != "" {
		return ensureMinLines(renderStatusMessage(m.status, m.statusKind), 2)
	}
	return ensureMinLines("", 2)
}

func (m Model) renderFooter() string {
	if m.state == StatePassword {
		return mutedStyle.Render("enter upload · esc cancel · ctrl+c quit")
	}
	if m.state == StateConfirmQuit {
		return errorStyle.Render("Cancel upload and quit? y/n")
	}
	if m.state == StateUpload {
		return mutedStyle.Render("esc cancel · q quit · ctrl+c quit")
	}
	return mutedStyle.Render("enter upload · esc clear · r remote · q quit · ctrl+c quit")
}

func (m Model) renderInput() string {
	contentWidth := m.inputContentWidth()
	input := m.input
	input.Width = m.inputTextWidth()
	if input.Value() == "" {
		_ = input.SetCursorMode(textinput.CursorHide)
	}
	content := lipgloss.Place(contentWidth, 1, lipgloss.Left, lipgloss.Center, input.View())
	return inputBoxStyle.Width(m.inputBoxWidth()).Render(content)
}

func (m Model) renderPasswordPrompt() string {
	remote := m.currentRequest.Remote
	content := []string{
		sectionStyle.Render("SSH password for " + remote.Target()),
		m.renderPasswordInput(),
	}
	if m.passwordError != "" {
		content = append(content, errorStyle.Render(m.passwordError))
	}
	content = append(content, mutedStyle.Render("enter upload · esc cancel · ctrl+c quit"))
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("12")).
		Padding(0, 1).
		Render(strings.Join(content, "\n"))
}

func (m Model) renderPasswordInput() string {
	input := m.passwordInput
	input.Width = min(28, max(16, m.innerWidth()-18))
	contentWidth := input.Width + lipgloss.Width(input.Prompt)
	content := lipgloss.Place(contentWidth, 1, lipgloss.Left, lipgloss.Center, input.View())
	return inputBoxStyle.Width(contentWidth + 2).Render(content)
}

func renderRemoteDetails(remote session.Remote) string {
	return fmt.Sprintf(
		"%s %s  %s %s  %s %s",
		sectionStyle.Render("Name:"),
		selectedStyle.Render(remote.Name),
		sectionStyle.Render("Target:"),
		remote.Target(),
		sectionStyle.Render("Destination:"),
		remote.Destination,
	)
}

func remoteNeedsPassword(remote session.Remote) bool {
	return remote.User != "" && remote.IdentityFile == ""
}

func (m Model) header(title string) string {
	width := m.innerWidth()
	rule := strings.Repeat("-", max(0, width-lipgloss.Width(title)-1))
	return titleStyle.Render(title) + " " + ruleStyle.Render(rule)
}

func (m Model) renderStatus() string {
	label := "idle"
	style := mutedStyle

	switch m.statusKind {
	case statusSyncing:
		label = "syncing"
		style = syncingStyle
	case statusSuccess:
		label = "Sync successful. Remote path copied to clipboard. ✓"
		style = successStyle
	case statusWarning:
		label = "warning: clipboard not copied"
		style = warningStyle
	case statusError:
		label = "error"
		style = errorStyle
	case statusCanceled:
		label = "canceled"
		style = warningStyle
	}

	return sectionStyle.Render("Status: ") + style.Render(label)
}

func renderStatusMessage(status string, kind statusKind) string {
	style := lipgloss.NewStyle()
	switch kind {
	case statusError:
		style = errorStyle
	case statusWarning, statusCanceled:
		style = warningStyle
	case statusSuccess:
		style = successStyle
	}

	lines := strings.Split(status, "\n")
	rendered := make([]string, 0, len(lines))
	for _, line := range lines {
		rendered = append(rendered, renderStatusLine(line, style))
	}
	return strings.Join(rendered, "\n")
}

func renderStatusLine(line string, style lipgloss.Style) string {
	if strings.HasPrefix(line, "Source: ") {
		return line
	}
	if strings.HasPrefix(line, "Destination: ") {
		return line
	}
	return style.Render(line)
}

func transferResultStatus(source string, destination string) string {
	return fmt.Sprintf("Source: %s\nDestination: %s", source, destination)
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

func waitForTransfer(events <-chan session.TransferEvent) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-events
		if !ok {
			return TransferEventMsg{Event: session.TransferEvent{Done: true}}
		}
		return TransferEventMsg{Event: event}
	}
}
