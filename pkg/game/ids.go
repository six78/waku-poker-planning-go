package game

import (
	"github.com/google/uuid"

	"github.com/six78/2-story-points-cli/pkg/protocol"
)

func GeneratePlayerID() (protocol.PlayerID, error) {
	playerUUID := uuid.New()
	return protocol.PlayerID(playerUUID.String()), nil
}

func GenerateIssueID() (protocol.IssueID, error) {
	itemUUID := uuid.New()
	return protocol.IssueID(itemUUID.String()), nil
}
