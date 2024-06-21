package cursor

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/suite"
)

func TestCursor(t *testing.T) {
	suite.Run(t, new(Suite))
}

type Suite struct {
	suite.Suite
}

func (s *Suite) TestDecrementCursor() {
	testCases := []struct {
		given    int
		expected int
	}{
		{
			given:    0,
			expected: 0,
		},
		{
			given:    1,
			expected: 0,
		},
		{
			given:    2,
			expected: 1,
		},
	}

	for _, tc := range testCases {
		test := func(*testing.T) {
			model := New(true, true)
			model.SetRange(0, 2)
			model.decrementCursor(tc.given)
			s.Require().Equal(tc.expected, model.Position())
		}
		testName := fmt.Sprintf("test decrement %d", tc.given)
		s.T().Run(testName, test)
	}
}

func (s *Suite) TestIncrementCursor() {
	testCases := []struct {
		given    int
		expected int
	}{
		{
			given:    0,
			expected: 1,
		},
		{
			given:    1,
			expected: 2,
		},
		{
			given:    2,
			expected: 2,
		},
	}

	for _, tc := range testCases {
		test := func(*testing.T) {
			model := New(true, true)
			model.SetRange(0, 2)
			model.incrementCursor(tc.given)
			s.Require().Equal(tc.expected, model.Position())
		}
		testName := fmt.Sprintf("test increment %d", tc.given)
		s.T().Run(testName, test)
	}
}

func (s *Suite) TestNew() {
	model := New(true, true)
	s.Require().True(model.Vertical())
	s.Require().True(model.Focused())

	model = New(true, false)
	s.Require().True(model.Vertical())
	s.Require().False(model.Focused())

	model = New(false, false)
	s.Require().False(model.Vertical())
	s.Require().False(model.Focused())
}

func (s *Suite) TestUpdate() {
	// Setup helpers
	keyRight := tea.KeyMsg{Type: tea.KeyRight}
	keyLeft := tea.KeyMsg{Type: tea.KeyLeft}
	keyUp := tea.KeyMsg{Type: tea.KeyUp}
	keyDown := tea.KeyMsg{Type: tea.KeyDown}
	allKeys := []tea.Msg{keyLeft, keyRight, keyUp, keyDown}

	testCases := []struct {
		name         string
		vertical     bool
		incrementKey tea.KeyType
		decrementKey tea.KeyType
	}{
		{
			name:         "horizontal cursor",
			vertical:     false,
			decrementKey: tea.KeyLeft,
			incrementKey: tea.KeyRight,
		},
		{
			name:         "vertical cursor",
			vertical:     true,
			decrementKey: tea.KeyUp,
			incrementKey: tea.KeyDown,
		},
	}

	for _, tc := range testCases {
		test := func(t *testing.T) {

			model := New(tc.vertical, false)

			cmd := model.Init()
			s.Require().Nil(cmd)

			model.SetRange(0, 2)
			model.SetPosition(1)
			s.Require().Equal(1, model.Position())
			s.Require().Equal(0, model.Min())
			s.Require().Equal(2, model.Max())

			// No focus - no reaction to any button
			for _, key := range allKeys {
				model = model.Update(key)
				s.Require().Equal(1, model.Position())
			}

			// Get focus
			model.SetFocus(true)
			s.Require().True(model.Focused())

			decrementMessage := tea.KeyMsg{Type: tc.decrementKey}
			incrementMessage := tea.KeyMsg{Type: tc.incrementKey}

			model = model.Update(decrementMessage)
			s.Require().Equal(0, model.Position())

			model = model.Update(decrementMessage)
			s.Require().Equal(0, model.Position())

			model = model.Update(incrementMessage)
			s.Require().Equal(1, model.Position())

			model = model.Update(incrementMessage)
			s.Require().Equal(2, model.Position())

			model = model.Update(incrementMessage)
			s.Require().Equal(2, model.Position())

			model = model.Update(decrementMessage)
			s.Require().Equal(1, model.Position())
		}

		s.T().Run(tc.name, test)
	}
}

func (s *Suite) TestAdjust() {
	model := New(true, true)
	model.SetRange(0, 2)

	// Expect position to adjust to min
	model.SetPosition(1)
	model.SetRange(2, 3)
	s.Require().Equal(2, model.Min())
	s.Require().Equal(3, model.Max())
	s.Require().Equal(2, model.Position())

	// Expect position to adjust to max
	model.SetPosition(3)
	model.SetRange(1, 2)
	s.Require().Equal(1, model.Min())
	s.Require().Equal(2, model.Max())
	s.Require().Equal(2, model.Position())

	// Expect max to adjust to min
	model.SetRange(3, 1)
	s.Require().Equal(3, model.Min())
	s.Require().Equal(3, model.Max())
}

func (s *Suite) TestMatch() {
	model := New(true, true)
	model.SetRange(0, 2)
	model.SetPosition(1)

	model.SetFocus(true)
	for _, v := range []int{0, 1, 2} {
		s.Require().Equal(v == 1, model.Match(v))
	}

	model.SetFocus(false)
	for _, v := range []int{0, 1, 2} {
		s.Require().False(model.Match(v))
	}
}
