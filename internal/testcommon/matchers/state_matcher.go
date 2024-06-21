package matchers

import (
	"encoding/json"
	"testing"

	"github.com/six78/2-story-points-cli/pkg/protocol"
)

type Callback func(state *protocol.State) bool

type StateMatcher struct {
	Matcher
	MessageMatcher
	cb    Callback
	state protocol.State
}

func NewStateMatcher(t *testing.T, cb Callback) *StateMatcher {
	return &StateMatcher{
		Matcher: *NewMatcher(t),
		cb:      cb,
	}
}

func (m *StateMatcher) Matches(x interface{}) bool {
	if !m.MessageMatcher.Matches(x) {
		return false
	}

	if m.message.Type != protocol.MessageTypeState {
		return false
	}

	var stateMessage protocol.GameStateMessage
	err := json.Unmarshal(m.payload, &stateMessage)
	if err != nil {
		return false
	}

	m.state = stateMessage.State
	m.triggered <- stateMessage.State

	if m.cb == nil {
		return true
	}

	return m.cb(&stateMessage.State)
}

func (m *StateMatcher) String() string {
	return "is state message matching custom condition"
}

func (m *StateMatcher) State() protocol.State {
	return m.state
}

func (m *StateMatcher) Wait() protocol.State {
	return m.Matcher.Wait().(protocol.State)
}
