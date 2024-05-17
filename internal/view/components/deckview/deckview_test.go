package deckview

import (
	"2sp/internal/testcommon"
	"2sp/internal/view/cursor"
	"2sp/internal/view/messages"
	"2sp/pkg/game"
	protocol2 "2sp/pkg/protocol"
	"github.com/brianvoe/gofakeit/v6"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/suite"
	"testing"
)

func TestDeckView(t *testing.T) {
	suite.Run(t, new(Suite))
}

type Suite struct {
	testcommon.Suite
}

func (s *Suite) TestRenderCard() {
	testCases := []struct {
		value    protocol2.VoteValue
		cursor   bool
		voted    bool
		expected string
	}{
		{protocol2.VoteValue("1"), false, false, "     \n╭───╮\n│ 1 │\n╰───╯\n     "},
		{protocol2.VoteValue("2"), true, false, "     \n╭───╮\n│ 2 │\n╰───╯\n  ^  "},
		{protocol2.VoteValue("3"), true, true, "╭───╮\n│ 3 │\n╰───╯\n     \n  ^  "},
		{protocol2.VoteValue("4"), false, true, "╭───╮\n│ 4 │\n╰───╯\n     "},
	}

	for _, tc := range testCases {
		result := renderCard(tc.value, tc.cursor, false, tc.voted)
		s.Require().Equal(tc.expected, result)
	}
}

func (s *Suite) TestRenderDeck() {
	model := Model{
		deck:         game.CreateDeck([]string{"1", "2", "3"}),
		voteState:    protocol2.VotingState,
		myVote:       protocol2.VoteValue("2"),
		focused:      true,
		isDealer:     false,
		commandMode:  false,
		voteCursor:   cursor.New(false, false),
		finishCursor: cursor.New(false, false),
	}

	model.voteCursor.SetRange(0, len(model.deck)-1)
	model.voteCursor.SetPosition(2)
	model.voteCursor.SetFocus(true)
	model.finishCursor.SetRange(0, len(model.deck)-1)
	model.finishCursor.SetPosition(0)
	model.finishCursor.SetFocus(false)

	result := model.View()
	expected := `
      ╭───╮       
╭───╮ │ 2 │ ╭───╮ 
│ 1 │ ╰───╯ │ 3 │ 
╰───╯       ╰───╯ 
              ^   
`
	// Remove leading and trailing newlines
	expected = expected[1 : len(expected)-1]

	s.Require().Equal(expected, result)
}

func (s *Suite) TestModelInit() {
	testCases := []struct {
		name string
	}{{
		name: "new model focused",
	}, {
		name: "new model blurred",
	},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			model := New()
			cmd := model.Init()
			s.Require().Nil(cmd)
			s.Require().False(model.focused)
			s.Require().Zero(model.VoteCursor())
			s.Require().Zero(model.FinishCursor())
		})
	}
}

func (s *Suite) TestFocus() {
	model := New()
	model.Focus()
	s.Require().True(model.focused)
	model.Blur()
	s.Require().False(model.focused)
}

func (s *Suite) TestUpdate() {
	issueID := protocol2.IssueID(gofakeit.Letter())
	deck := make(protocol2.Deck, 3)
	gofakeit.Slice(deck)

	model := New()
	_ = model.Init()

	model = model.Update(messages.GameStateMessage{
		State: nil,
	})

	s.Require().Equal(protocol2.Deck{}, model.deck)
	s.Require().Equal(protocol2.IdleState, model.voteState)

	model = model.Update(messages.GameStateMessage{
		State: &protocol2.State{
			Deck:          deck,
			ActiveIssue:   issueID,
			VotesRevealed: false,
		},
	})

	s.Require().Equal(deck, model.deck)
	s.Require().Equal(protocol2.VotingState, model.voteState)

	model = model.Update(messages.RoomJoin{IsDealer: true})
	s.Require().True(model.isDealer)

	model = model.Update(messages.RoomJoin{IsDealer: false})
	s.Require().False(model.isDealer)

	model = model.Update(messages.CommandModeChange{CommandMode: true})
	s.Require().True(model.commandMode)

	model = model.Update(messages.CommandModeChange{CommandMode: false})
	s.Require().False(model.commandMode)

	value := protocol2.VoteValue(gofakeit.Letter())
	model = model.Update(messages.MyVote{Result: protocol2.VoteResult{
		Value: value,
	}})
	s.Require().Equal(value, model.myVote)

}

func (s *Suite) TestCursor() {
	issueID := protocol2.IssueID(gofakeit.Letter())
	deck := make(protocol2.Deck, 3)
	gofakeit.Slice(deck)

	model := New()
	_ = model.Init()
	model.Focus()

	s.Require().True(model.focused)
	s.Require().False(model.commandMode)

	// Switch to VotingState as dealer
	model = model.Update(messages.RoomJoin{
		IsDealer: true,
	})
	model = model.Update(messages.GameStateMessage{
		State: &protocol2.State{
			Deck:          deck,
			ActiveIssue:   issueID,
			VotesRevealed: false,
		},
	})
	s.Require().True(model.isDealer)
	s.Require().Equal(protocol2.VotingState, model.voteState)
	s.Require().True(model.voteCursor.Focused())
	s.Require().False(model.finishCursor.Focused())

	// Ensure initial positions
	s.Require().Equal(0, model.VoteCursor())
	s.Require().Equal(0, model.FinishCursor())

	// Setup helpers
	keyRight := tea.KeyMsg{Type: tea.KeyRight}
	keyLeft := tea.KeyMsg{Type: tea.KeyLeft}

	inc := func(value int, condition bool) int {
		if condition {
			return value + 1
		}
		return value
	}

	testCursors := func(vote bool, finish bool) {
		initialVote := model.VoteCursor()
		initialFinish := model.FinishCursor()

		// Require not to be at the end of the deck
		s.Require().Less(initialVote, len(deck)-1)
		s.Require().Less(initialFinish, len(deck)-1)

		model = model.Update(keyRight)
		s.Require().Equal(inc(initialVote, vote), model.VoteCursor())
		s.Require().Equal(inc(initialFinish, finish), model.FinishCursor())

		model = model.Update(keyLeft)
		s.Require().Equal(initialVote, model.VoteCursor())
		s.Require().Equal(initialFinish, model.FinishCursor())
	}

	// Check deck cursor movement, ensure finish cursor stays in place
	testCursors(true, false)

	// Switch to revealed state
	model = model.Update(messages.GameStateMessage{
		State: &protocol2.State{
			Deck:          deck,
			ActiveIssue:   issueID,
			VotesRevealed: true,
			Issues: protocol2.IssuesList{
				&protocol2.Issue{ID: issueID},
			},
		},
	})
	s.Require().Equal(protocol2.RevealedState, model.voteState)
	s.Require().True(model.finishCursor.Focused())
	s.Require().False(model.voteCursor.Focused())

	// Move finish cursor to the middle to check no movement in both directions later
	model = model.Update(keyRight)
	s.Require().Equal(1, model.FinishCursor())

	// Check finish cursor movement, ensure deck cursor stays in place
	testCursors(false, true)

	// Switch to non-dealer
	model = model.Update(messages.RoomJoin{IsDealer: false})
	s.Require().Equal(protocol2.RevealedState, model.voteState)

	// Check both cursors stay in place
	testCursors(false, false)

	// Disable focus
	model.Blur()
	testCursors(false, false)

	// Enable focus, enable command mode
	model.Focus()
	model = model.Update(messages.CommandModeChange{CommandMode: true})
	testCursors(false, false)

	// Change vote state to not voting or revealed
	model = model.Update(messages.GameStateMessage{
		State: &protocol2.State{
			Deck: deck,
		},
	})
	testCursors(false, false)
}

func (s *Suite) TestBorderStyle() {
	style := cardBorderStyle(false, false)
	s.Require().Equal(&defaultBorderStyle, style)

	style = cardBorderStyle(true, false)
	s.Require().Equal(&votedBorderStyle, style)

	style = cardBorderStyle(false, true)
	s.Require().Equal(&highlightBorderStyle, style)

	style = cardBorderStyle(true, true)
	s.Require().Equal(&highlightBorderStyle, style)
}