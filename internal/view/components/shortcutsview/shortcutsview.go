package shortcutsview

import (
	"fmt"

	bubblekey "github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/six78/2-story-points-cli/internal/view/commands"
	"github.com/six78/2-story-points-cli/internal/view/messages"
	"github.com/six78/2-story-points-cli/internal/view/states"
	"github.com/six78/2-story-points-cli/pkg/protocol"
)

const (
	separator1 = " "
	separator2 = "  "
)

var (
	keyStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	textStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
)

type Model struct {
	roomView    states.RoomView
	commandMode bool
	isDealer    bool
	inRoom      bool
	voteState   protocol.VoteState
}

func New() Model {
	return Model{
		roomView:    states.ActiveIssueView,
		commandMode: false,
		isDealer:    false,
		inRoom:      false,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg, view states.RoomView) Model {
	m.roomView = view

	switch msg := msg.(type) {
	case messages.CommandModeChange:
		m.commandMode = msg.CommandMode
	case messages.RoomJoin:
		m.inRoom = !msg.RoomID.Empty()
		m.isDealer = msg.IsDealer
	case messages.GameStateMessage:
		if msg.State != nil {
			m.voteState = msg.State.VoteState()
		} else {
			m.voteState = protocol.IdleState
		}
	}

	return m
}

func (m Model) View() string {
	keys := commands.DefaultKeyMap

	var rows []string

	if !m.inRoom {
		row := key(keys.NewRoom) + separator1 + text("to create a new room") + separator2 +
			//key(keys.JoinRoom) + text(" "+keys.JoinRoom.Help().Desc) + separator2 +
			text("... or just ") + keyText("[paste]") + text(" the room id to join")
		rows = append(rows, row)
	}

	if m.inRoom {
		switch m.roomView { // Row 1
		case states.ActiveIssueView:
			row := text("Use ") + key(keys.PreviousCard) +
				text(" and ") + key(keys.NextCard) +
				text(" arrows to select card")
			rows = append(rows, row)

		case states.IssuesListView:
			row := "" // Keep empty line for alignment between views
			if !m.isDealer || (m.voteState != protocol.IdleState && m.voteState != protocol.FinishedState) {
				// Selecting issue is only available for dealer
				row = text("Use ") + key(keys.PreviousIssue) +
					text(" and ") + key(keys.NextIssue) +
					text(" arrows to select issue")
			}
			rows = append(rows, row)
		}
	}

	if m.inRoom { // Row 2 (optional, dealer-only)
		row := ""
		if m.voteState == protocol.VotingState && m.isDealer {
			row += keyHelp(keys.RevealVotes) + separator2
		}
		row += key(keys.SelectCard)

		switch m.roomView { // Row 1
		case states.ActiveIssueView:
			switch m.voteState {
			case protocol.VotingState:
				row += text(" Vote") + separator2 + keyHelp(keys.RevokeVote)
			case protocol.RevealedState:
				row += text(" to save estimation")
			default:
				row = ""
			}
		case states.IssuesListView:
			row += text(" Deal issue")
		}

		rows = append(rows, row)
	}

	{ // Row 3
		var row string

		if m.inRoom {
			row = key(keys.ToggleView)
			switch m.roomView {
			case states.ActiveIssueView:
				row += text(" Switch to issues list view")
			case states.IssuesListView:
				row += text(" Switch to room view")
			default:
				row += text(" Toggle room view")
			}
			row += separator2
		}

		row += key(keys.ToggleInput)
		if m.commandMode {
			row += text(" Switch to shortcuts mode")
		} else {
			row += text(" Switch to commands mode")
		}

		if m.inRoom {
			row += separator2 + keyHelp(keys.ExitRoom)
		}

		rows = append(rows, row)
	}

	return lipgloss.JoinVertical(lipgloss.Top, rows...)
}

func key(key bubblekey.Binding) string {
	s := fmt.Sprintf("[%s]", key.Help().Key)
	return keyText(s)
}

func keyText(text string) string {
	return keyStyle.Render(text)
}

func text(text string) string {
	return textStyle.Render(text)
}

func help(key bubblekey.Binding) string {
	return text(key.Help().Desc)
}

func keyHelp(k bubblekey.Binding) string {
	return key(k) + separator1 + help(k)
}
