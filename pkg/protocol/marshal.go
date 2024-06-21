package protocol

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
)

func MarshalMessage(m *Message) ([]byte, error) {
	return json.Marshal(m)
}

func UnmarshalMessage(payload []byte) (*Message, error) {
	message := Message{}
	err := json.Unmarshal(payload, &message)
	return &message, err
}

func UnmarshalState(payload []byte) (*State, error) {
	state := GameStateMessage{}

	err := json.Unmarshal(payload, &state)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal message")
	}

	if state.Type != MessageTypeState {
		return nil, fmt.Errorf("message is not a state message, got: %s", state.Type)
	}

	return &state.State, err
}

func UnmarshalPlayerVote(payload []byte) (*PlayerVoteMessage, error) {
	vote := PlayerVoteMessage{}

	err := json.Unmarshal(payload, &vote)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal message")
	}

	if vote.Type != MessageTypePlayerVote {
		return nil, errors.New("message is not a player vote message")
	}

	return &vote, err
}
