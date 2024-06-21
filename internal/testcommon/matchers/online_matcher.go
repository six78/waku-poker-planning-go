package matchers

import (
	"encoding/json"
	"testing"

	"go.uber.org/zap"

	"github.com/six78/2-story-points-cli/internal/config"
	"github.com/six78/2-story-points-cli/pkg/protocol"
)

type OnlineMatcher struct {
	MessageMatcher
	playerID protocol.PlayerID
}

func NewOnlineMatcher(t *testing.T, playerID protocol.PlayerID) *OnlineMatcher {
	return &OnlineMatcher{
		MessageMatcher: *NewMessageMatcher(t),
		playerID:       playerID,
	}
}

func (m *OnlineMatcher) Matches(x interface{}) bool {
	if !m.MessageMatcher.Matches(x) {
		return false
	}

	if m.message.Type != protocol.MessageTypePlayerOnline {
		return false
	}

	var onlineMessage protocol.PlayerOnlineMessage
	err := json.Unmarshal(m.payload, &onlineMessage)
	if err != nil {
		return false
	}

	if onlineMessage.Player.ID != m.playerID {
		return false
	}

	config.Logger.Debug("<<< OnlineMatcher.Matches",
		zap.Any("onlineMessage", onlineMessage),
	)
	m.triggered <- onlineMessage
	return true
}

func (m *OnlineMatcher) String() string {
	return "is user online protocol message"
}

func (m *OnlineMatcher) Wait() protocol.PlayerOnlineMessage {
	return m.MessageMatcher.Wait().(protocol.PlayerOnlineMessage)
}
