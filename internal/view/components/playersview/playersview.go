package playersview

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"

	"github.com/six78/2-story-points-cli/internal/view/components/playervoteview"
	"github.com/six78/2-story-points-cli/internal/view/messages"
	"github.com/six78/2-story-points-cli/pkg/protocol"
)

const textColor = lipgloss.Color("#FAFAFA")
const borderColor = lipgloss.Color("#555555")

var (
	onlinePlayerStyle = lipgloss.NewStyle().
				Foreground(textColor).
				PaddingLeft(1).
				PaddingRight(1).
				Align(lipgloss.Center)
	offlinePlayerStyle = onlinePlayerStyle.Copy().
				Foreground(borderColor)
	myNameStyle = onlinePlayerStyle.Copy().Bold(true)
	borderStyle = lipgloss.NewStyle().Foreground(borderColor)
)

type PlayerVoteResult struct {
	Player protocol.Player
	Vote   string
	Style  lipgloss.Style
}

type Model struct {
	votes         []playervoteview.Model
	playerNames   []string
	playersOnline []bool
	playerID      protocol.PlayerID
	playerColumn  int
}

func New() Model {
	return Model{
		playerID: "",
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var commands []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case messages.PlayerIDMessage:
		m.playerID = msg.PlayerID
	case messages.GameStateMessage:
		handleNewState(&m, msg.State)

		for _, voteView := range m.votes {
			commands = append(commands, voteView.Init())
		}
	}

	for i, voteView := range m.votes {
		m.votes[i], cmd = voteView.Update(msg)
		commands = append(commands, cmd)
	}

	return m, tea.Batch(commands...)
}

func (m Model) View() string {
	var row []string
	for _, voteView := range m.votes {
		row = append(row, voteView.View())
	}

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(borderStyle).
		StyleFunc(func(row, col int) lipgloss.Style {
			switch {
			case row == 0:
				if col == m.playerColumn {
					return myNameStyle
				}
				if m.playersOnline[col] {
					return onlinePlayerStyle
				}
				return offlinePlayerStyle
			default:
				return m.votes[col].Style()
			}
		}).
		Headers(m.playerNames...).
		Rows([][]string{row}...)

	return t.String()
}

func handleNewState(m *Model, state *protocol.State) {
	// FIXME: only when players list changed
	if state == nil {
		m.playerNames = []string{}
		m.votes = []playervoteview.Model{}
		return
	}

	m.playerNames = make([]string, 0, len(state.Players))
	m.playersOnline = make([]bool, 0, len(state.Players))
	m.votes = make([]playervoteview.Model, 0, len(state.Players))

	for i, player := range state.Players {
		playerName := player.Name
		if player.ID == m.playerID {
			m.playerColumn = i
			playerName += " (You)"
		}
		m.playerNames = append(m.playerNames, playerName)
		m.playersOnline = append(m.playersOnline, player.Online)
		voteView := playervoteview.New(player.ID)
		m.votes = append(m.votes, voteView)
	}
}
