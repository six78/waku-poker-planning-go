package deckview

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"

	"github.com/six78/2-story-points-cli/internal/config"
	"github.com/six78/2-story-points-cli/internal/view/components/cursor"
	"github.com/six78/2-story-points-cli/internal/view/components/voteview"
	"github.com/six78/2-story-points-cli/internal/view/messages"
	"github.com/six78/2-story-points-cli/pkg/protocol"
)

var (
	defaultBorderStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))
	votedBorderStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#aaaaaa"))
	highlightBorderStyle = lipgloss.NewStyle().Foreground(config.UserColor)
)

type Model struct {
	deck         protocol.Deck
	voteState    protocol.VoteState
	myVote       protocol.VoteValue
	focused      bool
	isDealer     bool
	commandMode  bool
	voteCursor   cursor.Model
	finishCursor cursor.Model
}

func New() Model {
	return Model{
		voteCursor:   cursor.New(false, false),
		finishCursor: cursor.New(false, false),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.voteCursor.Init(),
		m.finishCursor.Init(),
	)
}

func (m Model) Update(msg tea.Msg) Model {
	switch msg := msg.(type) {
	case messages.GameStateMessage:
		if msg.State == nil {
			m.deck = protocol.Deck{}
			m.voteState = protocol.IdleState
			m.voteCursor.SetRange(0, 0)
		} else {
			m.deck = msg.State.Deck
			m.voteState = msg.State.VoteState()
			m.voteCursor.SetRange(0, len(m.deck)-1)
			m.finishCursor.SetRange(0, len(m.deck)-1)
		}
		m.updateCursorsState()

	case messages.RoomJoin:
		m.isDealer = msg.IsDealer
		m.updateCursorsState()

	case messages.CommandModeChange:
		m.commandMode = msg.CommandMode
		m.updateCursorsState()

	case messages.MyVote:
		m.myVote = msg.Result.Value
	}

	m.voteCursor = m.voteCursor.Update(msg)
	m.finishCursor = m.finishCursor.Update(msg)

	return m
}

func (m Model) View() string {
	cards := make([]string, 0, len(m.deck)*2)

	for i, value := range m.deck {
		card := renderCard(
			value,
			m.voteCursor.Match(i),
			m.finishCursor.Match(i),
			value == m.myVote,
		)
		cards = append(cards, card, " ") // Add a space between cards
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, cards...)
}

func (m *Model) updateCursorsState() {
	m.voteCursor.SetFocus(!m.commandMode && m.focused && (m.voteState == protocol.VotingState))
	m.finishCursor.SetFocus(!m.commandMode && m.focused && (m.voteState == protocol.RevealedState) && m.isDealer)
}

func (m *Model) Focus() {
	m.focused = true
	m.updateCursorsState()
}

func (m *Model) Blur() {
	m.focused = false
	m.updateCursorsState()
}

func (m *Model) VoteCursor() int {
	return m.voteCursor.Position()
}

func (m *Model) FinishCursor() int {
	return m.finishCursor.Position()
}

func renderCard(value protocol.VoteValue, voteCursor bool, finishCursor bool, voted bool) string {
	card := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(*cardBorderStyle(voted, finishCursor)).
		StyleFunc(func(row, col int) lipgloss.Style {
			return *voteview.VoteStyle(value)
		}).
		Rows([]string{string(value)}).
		String()

	var column []string
	column = []string{}

	if !voted {
		column = append(column, "")
	}

	column = append(column, card)

	if voteCursor {
		if voted {
			column = append(column, "")
		}
		column = append(column, "  ^")
	} else {
		column = append(column, "")
	}

	return lipgloss.JoinVertical(lipgloss.Top, column...)
}

func cardBorderStyle(voted bool, highlight bool) *lipgloss.Style {
	if highlight {
		return &highlightBorderStyle
	}
	if voted {
		return &votedBorderStyle
	}
	return &defaultBorderStyle
}
