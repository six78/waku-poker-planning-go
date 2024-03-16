package view

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"sort"
	"strings"
	"waku-poker-planning/config"
	"waku-poker-planning/protocol"
)

func (m model) renderAppState() string {
	switch m.state {
	case Idle:
		return "nothing is happening. boring life."
	case Initializing:
		return m.spinner.View() + " Starting Waku..."
	case InputPlayerName:
		return m.renderPlayerNameInput()
	case WaitingForPeers:
		return m.spinner.View() + " Connecting to Waku peers ..."
	case UserAction:
		return m.renderGame()
	}

	return "unknown app state"
}

func (m model) renderPlayerNameInput() string {
	return fmt.Sprintf(" \n\n%s\n%s", m.input.View(), m.lastCommandError)
}

func (m model) renderGame() string {
	if m.gameSessionID == "" {
		return fmt.Sprintf(
			" Join a game session or create a new one ...\n\n%s%s",
			m.input.View(),
			m.lastCommandError,
		)
	}

	if m.gameState == nil {
		return m.spinner.View() + " Waiting for initial game state ..."
	}

	return fmt.Sprintf(`
SESSION ID:   %s
DECK:         %s
ISSUE:        %s

%s

%s
`,
		m.gameSessionID,
		renderDeck(&m.gameState.Deck),
		renderIssue(&m.gameState.VoteItem),
		m.renderPlayers(),
		m.renderActionInput(),
	)
}

type PlayerVoteResult struct {
	Player protocol.Player
	Vote   string
	Style  lipgloss.Style
}

func (m model) renderActionInput() string {
	return fmt.Sprintf("%s\n%s",
		m.input.View(),
		m.renderActionError(),
	)
}

func (m model) renderActionError() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#d78700"))
	return style.Render(m.lastCommandError)
}

func (m model) renderPlayers() string {
	players := make([]PlayerVoteResult, 0, len(m.gameState.Players))

	for playerID, player := range m.gameState.Players {
		vote, style := renderVote(&m, playerID)
		players = append(players, PlayerVoteResult{
			Player: player,
			Vote:   vote,
			Style:  style,
		})
	}

	sort.Slice(players[:], func(i, j int) bool {
		playerI := players[i].Player
		playerJ := players[j].Player
		if playerI.Order != playerJ.Order {
			return playerI.Order < playerJ.Order
		}
		return playerI.Name < playerJ.Name
	})

	var votes []string
	var playerNames []string
	playerColumn := -1

	for i, player := range players {
		votes = append(votes, player.Vote)
		playerNames = append(playerNames, player.Player.Name)
		if player.Player.ID == m.app.Game.Player().ID {
			playerColumn = i
		}
	}

	var CommonStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA")).
		//Background(lipgloss.Color("#7D56F4")).
		//PaddingTop(2).
		PaddingLeft(1).
		PaddingRight(1).
		Align(lipgloss.Center)

	var HeaderStyle = CommonStyle.Copy().Bold(true)

	rows := [][]string{
		votes,
	}

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))).
		StyleFunc(func(row, col int) lipgloss.Style {
			switch {
			case row == 0:
				if col == playerColumn {
					return HeaderStyle
				} else {
					return CommonStyle
				}
			default:
				return players[col].Style
			}
		}).
		Headers(playerNames...).
		Rows(rows...)

	return t.String()
}

var CommonVoteStyle = lipgloss.NewStyle().
	PaddingLeft(1).
	PaddingRight(1).
	Align(lipgloss.Center)

var NoVoteStyle = CommonVoteStyle.Copy().Foreground(lipgloss.Color("#444444"))
var ReadyVoteStyle = CommonVoteStyle.Copy().Foreground(lipgloss.Color("#5fd700"))
var LightVoteStyle = CommonVoteStyle.Copy().Foreground(lipgloss.Color("#00d7ff"))
var MediumVoteStyle = CommonVoteStyle.Copy().Foreground(lipgloss.Color("#ffd787"))
var DangerVoteStyle = CommonVoteStyle.Copy().Foreground(lipgloss.Color("#ff005f"))

func voteStyle(vote protocol.VoteResult) lipgloss.Style {
	if vote >= 13 {
		return DangerVoteStyle
	}
	if vote >= 5 {
		return MediumVoteStyle
	}
	return LightVoteStyle
}

func renderVote(m *model, playerID protocol.PlayerID) (string, lipgloss.Style) {
	if m.gameState.VoteState == protocol.IdleState {
		return "", CommonVoteStyle
	}
	vote, ok := m.gameState.TempVoteResult[playerID]
	if !ok {
		if m.gameState.VoteState == protocol.RevealedState ||
			m.gameState.VoteState == protocol.FinishedState {
			return "X", NoVoteStyle
		}
		return m.spinner.View(), CommonVoteStyle
	}
	if playerID == m.app.Game.Player().ID {
		playerVote := m.app.Game.PlayerVote()
		vote = &playerVote
	}
	if vote == nil {
		return "✓", ReadyVoteStyle
	}
	return fmt.Sprintf("%d", *vote), voteStyle(*vote)
}

func renderLogPath() string {
	return fmt.Sprintf("LOG FILE: file:///%s", config.LogFilePath)
}

func renderDeck(deck *protocol.Deck) string {
	votes := make([]string, 0, len([]protocol.VoteResult(*deck)))
	for _, vote := range []protocol.VoteResult(*deck) {
		votes = append(votes, fmt.Sprintf("%d", vote))
	}
	return strings.Join(votes, ", ")
}

func renderIssue(item *protocol.VoteItem) string {
	if item.URL == "" {
		return item.Name
	}
	if item.Name == "" {
		return item.URL
	}
	return fmt.Sprintf("%s (%s)", item.URL, item.Name)
}
