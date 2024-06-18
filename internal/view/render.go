package view

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"github.com/six78/2-story-points-cli/internal/config"
	"github.com/six78/2-story-points-cli/internal/view/states"
	"strings"
)

/*
	TODO: Refactor this.
		I don't like that on each method we're copying the model, this is bad.
		Probably a good idea would be to have a separate Model (with Init/Update/View methods)
		for each component. This would be a proper way of using bubbletea!
*/

var (
	foregroundShadeStyle = lipgloss.NewStyle().Foreground(config.ForegroundShadeColor)
)

func (m model) renderAppState() string {
	switch m.state {
	case states.Idle:
		return "nothing is happening. boring life."
	case states.Initializing:
		return m.spinner.View() + " Starting Waku..."
	case states.InputPlayerName:
		return m.renderPlayerNameInput()
	case states.WaitingForPeers:
		return m.spinner.View() + " Connecting to Waku peers..."
	case states.Playing:
		return m.renderGame()
	}

	return "unknown app state"
}

func (m model) renderPlayerNameInput() string {
	return lipgloss.JoinVertical(
		lipgloss.Top,
		m.input.View(),
		m.errorView.View(),
	)
}

func (m model) renderGame() string {
	roomViewSeparator := ""
	if !m.roomID.Empty() {
		roomViewSeparator = "\n"
	}
	return lipgloss.JoinVertical(lipgloss.Top,
		m.wakuStatusView.View(),
		m.renderRoomID(),
		roomViewSeparator+m.renderRoomView(),
		m.renderActionInput(),
		m.errorView.View())
}

func (m model) renderRoomID() string {
	if m.roomID.Empty() {
		return "  Join a room or create a new one ..."
	}
	var dealerString string
	if m.game.IsDealer() {
		dealerString = foregroundShadeStyle.Render(" (dealer)")
	}
	return "  Room: " + m.roomID.String() + dealerString
}

func (m model) renderRoomView() string {
	switch m.roomViewState {
	case states.ActiveIssueView:
		return m.renderRoomCurrentIssueView()
	case states.IssuesListView:
		return m.issuesListView.View()
	default:
		return fmt.Sprintf("unknown view: %d", m.roomViewState)
	}
}

func (m model) renderRoomCurrentIssueView() string {
	if m.roomID.Empty() {
		return ""
	}

	if m.gameState == nil {
		return fmt.Sprintf("\n%s Waiting for initial game state ...\n",
			m.spinner.View(),
		)
	}

	style := lipgloss.NewStyle().Bold(true)
	italic := lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("#cccccc"))

	return lipgloss.JoinVertical(lipgloss.Top,
		m.issueView.View(),
		"",
		m.playersView.View(),
		italic.Render(
			fmt.Sprintf("%s %s %s %s",
				italic.Render("Advice: start discussion between"),
				style.Render("Alice"),
				italic.Render("and"),
				style.Render("Bob"),
			),
		),
		m.deckView.View(),
	)
}

func (m model) renderActionInput() string {
	if m.commandMode {
		return m.input.View()
	}
	return m.shortcutsView.View()
}

func renderLogPath() string {
	path := strings.Replace(config.LogFilePath, " ", "%20", -1)
	return fmt.Sprintf("Log: file:///%s", path)
}
