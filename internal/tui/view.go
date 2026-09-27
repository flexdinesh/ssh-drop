package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

var (
	accent        = lipgloss.AdaptiveColor{Light: "#356300", Dark: "#B4F566"}
	ink           = lipgloss.AdaptiveColor{Light: "#182522", Dark: "#EAF0EA"}
	muted         = lipgloss.AdaptiveColor{Light: "#52645C", Dark: "#A6BAB4"}
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(ink)
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(accent)
	mutedStyle    = lipgloss.NewStyle().Foreground(muted)
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#A32633", Dark: "#FF8C96"})
	successStyle  = lipgloss.NewStyle().Foreground(accent)
	warningStyle  = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#805000", Dark: "#EAB95A"})
	inputBoxStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(accent).Padding(0, 1)
)

func (m *Model) setSize(width, height int) {
	if width > 0 {
		m.width = width
	}
	if height > 0 {
		m.height = height
	}
	m.input.Width = m.inputTextWidth()
}

func (m Model) viewWidth() int      { return max(1, min(84, m.width)) }
func (m Model) innerWidth() int     { return max(1, m.viewWidth()-4) }
func (m Model) inputTextWidth() int { return max(1, m.innerWidth()-7) }

func (m Model) View() string {
	var content string
	switch m.state {
	case StateRemotePicker:
		content = m.renderPicker()
	case StatePassword:
		content = m.renderPasswordView()
	case StateConnecting, StateConnectionFailed:
		content = m.renderConnection()
	default:
		content = m.renderDropView()
	}
	// Terminal cells, not byte counts: long paths and Unicode stay in bounds.
	content = ansi.Hardwrap(content, m.innerWidth(), true)
	return lipgloss.NewStyle().Padding(1, 2).Render(content)
}

func (m Model) brand() string { return selectedStyle.Render("ssh-drop") }

func (m Model) renderPicker() string {
	lines := []string{m.brand(), "", titleStyle.Render("Choose a remote"), mutedStyle.Render("Connect once. Drop again whenever you need."), ""}
	start := max(0, m.cursor-max(1, (m.height-10)/3)+1)
	end := min(len(m.start.Config.Remotes), start+max(1, (m.height-10)/3))
	compact := m.height < 20
	if compact {
		lines = []string{m.brand(), titleStyle.Render("Choose a remote")}
	}
	for i := start; i < end; i++ {
		remote := m.start.Config.Remotes[i]
		marker, style := "  ", titleStyle
		if i == m.cursor {
			marker, style = "> ", selectedStyle
		}
		if compact {
			lines = append(lines, style.Render(ansi.Truncate(marker+remote.Name+"  "+remote.Target(), m.innerWidth(), "…")), mutedStyle.Render(ansi.Truncate("  -> "+remote.Destination, m.innerWidth(), "…")))
		} else {
			lines = append(lines, style.Render(ansi.Truncate(marker+remote.Name+"  "+remote.Target(), m.innerWidth(), "…")), mutedStyle.Render(ansi.Truncate("  -> "+remote.Destination, m.innerWidth(), "…")), "")
		}
	}
	if end < len(m.start.Config.Remotes) || start > 0 {
		lines = append(lines, mutedStyle.Render(fmt.Sprintf("%d of %d remotes", m.cursor+1, len(m.start.Config.Remotes))))
	}
	lines = append(lines, mutedStyle.Render("↑/↓ choose · enter connect · q quit"))
	return strings.Join(lines, "\n")
}

func (m Model) remoteLine() string {
	r := m.SelectedRemote()
	return mutedStyle.Render("To ") + selectedStyle.Render(r.Name) + mutedStyle.Render(" / "+r.Target())
}

func (m Model) renderConnection() string {
	heading, detail := "Connecting to "+m.SelectedRemote().Name+"…", "Trying SSH key or agent authentication."
	if m.passwordAttempted {
		detail = "Authenticating with password."
	}
	footer := "esc choose remote · q quit"
	if m.state == StateConnectionFailed {
		heading, detail = "Could not connect", m.status
		footer = "enter reconnect · r remote · q quit"
	}
	if m.height < 20 {
		return strings.Join([]string{m.brand(), ansi.Truncate(m.remoteLine(), m.innerWidth(), "…"), titleStyle.Render(ansi.Truncate(heading, m.innerWidth(), "…")), m.fitLines(detail, max(1, m.height-8)), mutedStyle.Render(footer)}, "\n")
	}
	return strings.Join([]string{m.brand(), "", m.remoteLine(), "", titleStyle.Render(heading), "", m.fitLines(detail, 4), "", mutedStyle.Render(footer)}, "\n")
}

func (m Model) renderDropView() string {
	if m.height < 20 {
		return m.renderCompactDrop()
	}
	heading := "Drop a file"
	if m.summary.Successes > 0 {
		heading = "Drop another image"
	}
	lines := []string{m.brand(), "", m.remoteLine() + mutedStyle.Render(" · connected"), "", titleStyle.Render(heading)}
	if m.height >= 20 {
		lines = append(lines, mutedStyle.Render("Paste or drag a local file path, then press enter."))
	}
	lines = append(lines, m.renderInput(), "", m.renderRoute())
	if m.state == StateUpload || m.state == StateConfirmQuit {
		lines = append(lines, m.fitLines("Source: "+m.currentRequest.LocalPath, 2), m.fitLines("Destination: "+m.currentRequest.DestinationPath, 2))
		if progress := lastOutputLine(m.uploadOutput); progress != "" {
			lines = append(lines, mutedStyle.Render(ansi.Truncate(progress, m.innerWidth(), "…")))
		}
	} else if m.status != "" {
		style := titleStyle.Bold(false)
		if m.statusKind == statusError {
			style = errorStyle
		}
		if m.statusKind == statusCanceled {
			style = warningStyle
		}
		lines = append(lines, style.Render(m.fitLines(m.status, max(2, m.height-17))))
	} else {
		lines = append(lines, mutedStyle.Render("Destination: "+m.SelectedRemote().Destination))
	}
	lines = append(lines, "", m.renderStatus(), m.renderFooter())
	return strings.Join(lines, "\n")
}

func (m Model) renderCompactDrop() string {
	heading := "Drop a file"
	if m.summary.Successes > 0 {
		heading = "Drop another image"
	}
	input := m.input
	input.Width = max(1, m.innerWidth()-3)
	if m.state == StateUpload || m.state == StateConfirmQuit {
		input.Blur()
	}
	lines := []string{m.brand() + mutedStyle.Render(" · "+m.SelectedRemote().Name), titleStyle.Render(heading), input.View(), m.renderRoute()}
	if m.status != "" {
		lines = append(lines, m.fitLines(m.status, max(1, m.height-9)))
	} else if m.state == StateUpload || m.state == StateConfirmQuit {
		lines = append(lines, mutedStyle.Render(ansi.Truncate(lastOutputLine(m.uploadOutput), m.innerWidth(), "…")))
	} else {
		lines = append(lines, mutedStyle.Render(ansi.Truncate("To "+m.SelectedRemote().Destination, m.innerWidth(), "…")))
	}
	lines = append(lines, m.renderStatus())
	if m.state == StateDrop {
		lines = append(lines, mutedStyle.Render("enter upload · esc clear"), mutedStyle.Render("r remote · q quit"))
	} else {
		lines = append(lines, m.renderFooter())
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderRoute() string {
	copyLabel := "CLIPBOARD"
	if m.innerWidth() < 40 {
		copyLabel = "COPY"
	}
	file, ssh, copyPath := selectedStyle.Render("FILE"), mutedStyle.Render("SSH"), mutedStyle.Render(copyLabel)
	if m.state == StateUpload || m.state == StateConfirmQuit {
		file, ssh = successStyle.Render("FILE ✓"), selectedStyle.Render("SSH …")
	}
	if (m.statusKind == statusSuccess || m.statusKind == statusWarning || m.statusKind == statusCopying) && m.input.Value() == "" {
		file, ssh = successStyle.Render("FILE ✓"), successStyle.Render("SSH ✓")
		if m.statusKind == statusCopying {
			copyPath = selectedStyle.Render(copyLabel + " …")
		} else if m.statusKind == statusSuccess {
			copyPath = successStyle.Render(copyLabel + " ✓")
		} else {
			copyPath = warningStyle.Render(copyLabel + " !")
		}
	}
	gap := max(1, (m.innerWidth()-lipgloss.Width(file+ssh+copyPath)-6)/2)
	connector := mutedStyle.Render(" " + strings.Repeat("─", gap) + " ")
	return file + connector + ssh + connector + copyPath
}

func (m Model) renderInput() string {
	input := m.input
	input.Width = m.inputTextWidth()
	if m.state == StateUpload || m.state == StateConfirmQuit {
		input.Blur()
	}
	return inputBoxStyle.Width(max(1, m.innerWidth()-2)).Render(input.View())
}

func (m Model) renderStatus() string {
	if m.height < 20 {
		switch m.statusKind {
		case statusCopying:
			return selectedStyle.Render("Uploaded · copying path")
		case statusSuccess:
			return successStyle.Render("Uploaded · path copied")
		case statusWarning:
			return warningStyle.Render("Uploaded · clipboard unavailable")
		case statusError:
			return errorStyle.Render("Error · correct and retry")
		case statusCanceled:
			return warningStyle.Render("Canceled · ready to retry")
		}
	}
	switch m.statusKind {
	case statusCopying:
		return selectedStyle.Render("Uploaded. Copying remote path…")
	case statusSyncing:
		return selectedStyle.Render("Uploading…")
	case statusSuccess:
		return successStyle.Render("Uploaded. Remote path copied to clipboard.")
	case statusWarning:
		return warningStyle.Render("Uploaded. Clipboard not copied; use the path above.")
	case statusError:
		return errorStyle.Render("Check the error above, then retry.")
	case statusCanceled:
		return warningStyle.Render("Canceled. Ready to try again.")
	default:
		return mutedStyle.Render("Ready to upload")
	}
}

func (m Model) renderFooter() string {
	if m.state == StateConfirmQuit {
		return warningStyle.Render("Cancel upload and quit? y/n")
	}
	if m.state == StateUpload {
		return mutedStyle.Render("esc cancel · q quit · ctrl+c quit")
	}
	return mutedStyle.Render("enter upload · esc clear · r remote · q quit")
}

func (m Model) renderPasswordView() string {
	input := m.passwordInput
	input.Width = max(1, min(28, m.innerWidth()-8))
	content := []string{titleStyle.Render("SSH password"), mutedStyle.Render(m.SelectedRemote().Target()), "", input.View(), ""}
	if m.height < 16 {
		content = []string{titleStyle.Render("SSH password"), mutedStyle.Render(ansi.Truncate(m.SelectedRemote().Target(), m.innerWidth()-6, "…")), input.View()}
	}
	if m.passwordError != "" {
		content = append(content, errorStyle.Render(m.passwordError))
	}
	content = append(content, mutedStyle.Render("enter connect · esc cancel"))
	popup := inputBoxStyle.Width(max(1, min(46, m.innerWidth()-2))).Render(strings.Join(content, "\n"))
	if m.height < 16 {
		return popup
	}
	height := max(lipgloss.Height(popup), min(20, m.height-4))
	return m.brand() + "\n" + lipgloss.Place(m.innerWidth(), height, lipgloss.Center, lipgloss.Center, popup)
}

func (m Model) fitLines(text string, limit int) string {
	lines := strings.Split(ansi.Hardwrap(ansi.Strip(text), m.innerWidth(), true), "\n")
	if len(lines) > limit {
		lines = lines[:limit]
		lines[limit-1] = ansi.Truncate(lines[limit-1], max(1, m.innerWidth()-1), "") + "…"
	}
	return strings.Join(lines, "\n")
}

func lastOutputLine(output string) string {
	lines := strings.FieldsFunc(ansi.Strip(output), func(r rune) bool { return r == '\r' || r == '\n' })
	if len(lines) == 0 {
		return ""
	}
	return strings.TrimSpace(lines[len(lines)-1])
}
