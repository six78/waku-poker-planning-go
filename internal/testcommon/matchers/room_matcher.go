package matchers

import (
	"fmt"

	"github.com/six78/2-story-points-cli/pkg/protocol"
)

type RoomMatcher struct {
	room *protocol.Room
}

func NewRoomMatcher(room *protocol.Room) *RoomMatcher {
	return &RoomMatcher{
		room: room,
	}
}

func (m *RoomMatcher) Matches(x interface{}) bool {
	room, ok := x.(*protocol.Room)
	return ok && m.room.ToRoomID() == room.ToRoomID()
}

func (m *RoomMatcher) String() string {
	return fmt.Sprintf("is equal to room %s", m.room.ToRoomID())
}
