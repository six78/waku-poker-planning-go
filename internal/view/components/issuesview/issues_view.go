package issuesview

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/six78/2-story-points-cli/internal/config"
	"github.com/six78/2-story-points-cli/internal/view/components/cursor"
	"github.com/six78/2-story-points-cli/internal/view/components/voteview"
	"github.com/six78/2-story-points-cli/internal/view/messages"
	"github.com/six78/2-story-points-cli/pkg/protocol"
)

const (
	cursorSymbol = ">"
)

var (
	highlightStyle = lipgloss.NewStyle().Foreground(config.UserColor)
)

type Model struct {
	issues      protocol.IssuesList
	activeIssue protocol.IssueID
	commandMode bool
	isDealer    bool
	focused     bool

	cursor  cursor.Model
	spinner spinner.Model
}

func New() Model {
	s := spinner.New()
	s.Spinner = spinner.MiniDot

	return Model{
		cursor:  cursor.New(true, false),
		spinner: s,
	}
}

func (m Model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case messages.GameStateMessage:
		if msg.State == nil {
			m.issues = nil
			m.activeIssue = ""
		} else {
			m.issues = msg.State.Issues
			m.activeIssue = msg.State.ActiveIssue
		}
		m.cursor.SetRange(0, len(m.issues)-1)
		m.updateCursorFocus()

	case messages.CommandModeChange:
		m.commandMode = msg.CommandMode
		m.updateCursorFocus()

	case messages.RoomJoin:
		m.isDealer = msg.IsDealer
		m.updateCursorFocus()
	}

	var spinnerCommand tea.Cmd
	m.spinner, spinnerCommand = m.spinner.Update(msg)
	m.cursor = m.cursor.Update(msg)

	return m, spinnerCommand
}

func (m Model) View() string {
	issues := m.issues
	activeIssue := m.activeIssue

	var items []string

	for i, issue := range issues {
		result := "  - "
		if issue.Result != nil {
			vote := fmt.Sprintf("%2s", string(*issue.Result))
			result = voteview.VoteStyle(*issue.Result).Render(vote)
		} else if issue.ID == activeIssue {
			result = fmt.Sprintf(" %2s ", m.spinner.View())
		}

		var item string
		var style lipgloss.Style

		if m.cursor.Match(i) {
			item += cursorSymbol
			style = highlightStyle
		} else {
			item += " "
		}

		item += fmt.Sprintf("%s  %s", result, issue.TitleOrURL)
		items = append(items, style.Render(item))
	}

	if len(items) == 0 {
		items = append(items, "- No issues dealt yet")
	}

	fullBlock := lipgloss.JoinVertical(lipgloss.Top, items...)
	return fmt.Sprintf("Issues:\n%s\n", fullBlock)
}

func (m *Model) updateCursorFocus() {
	m.cursor.SetFocus(!m.commandMode && m.isDealer && m.focused)
}

func (m *Model) Focus() {
	m.focused = true
	m.updateCursorFocus()
}

func (m *Model) Blur() {
	m.focused = false
	m.updateCursorFocus()
}

func (m *Model) CursorPosition() int {
	return m.cursor.Position()
}
