package playervoteview

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/six78/2-story-points-cli/internal/view/components/voteview"
	"github.com/six78/2-story-points-cli/internal/view/messages"
	"github.com/six78/2-story-points-cli/pkg/protocol"
)

type Model struct {
	voteView voteview.Model
	playerID protocol.PlayerID
}

func New(playerID protocol.PlayerID) Model {
	return Model{
		voteView: voteview.New(false),
		playerID: playerID,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.voteView.Init(),
	)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case messages.GameStateMessage:
		state := msg.State

		if state.VoteState() == protocol.IdleState {
			m.voteView.Clear()
			break
		}

		activeIssue := msg.State.Issues.Get(msg.State.ActiveIssue)
		if activeIssue == nil {
			m.voteView.Clear()
			break
		}

		vote, ok := activeIssue.Votes[m.playerID]
		if !ok {
			if state.VoteState() == protocol.VotingState {
				m.voteView.Spin()
			} else {
				m.voteView.Cross()
			}
			break
		}

		if vote.Value == "" {
			m.voteView.Hide()
			break
		}

		m.voteView.SetValue(vote.Value)
		m.voteView.Show()
	}

	var cmd tea.Cmd
	m.voteView, cmd = m.voteView.Update(msg)

	return m, cmd
}

func (m Model) View() string {
	return m.voteView.View()
}

// Style returns a standard style for given vote value.
// We don't render the style in View, as this component is frequently used in a table cell.
func (m Model) Style() lipgloss.Style {
	return m.voteView.Style()
}
