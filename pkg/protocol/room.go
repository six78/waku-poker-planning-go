package protocol

import (
	"crypto/rand"

	"github.com/mr-tron/base58"
	"github.com/pkg/errors"

	"github.com/six78/2-story-points-cli/internal/config"
)

type Room struct {
	Version      byte   `json:"version"`
	SymmetricKey []byte `json:"symmetricKey"`

	cachedRoomID *RoomID
}

type RoomID struct {
	string
}

func NewRoomID(roomID string) RoomID {
	return RoomID{roomID}
}

func (id RoomID) String() string {
	return id.string
}

func (id RoomID) Empty() bool {
	return id.string == ""
}

// RoomID: base58 encoded byte array:
// - byte 0: 	    version
// - byte 1..end: symmetric key
// Total expected length: 17 bytes

func (room *Room) Bytes() []byte {
	bytes := make([]byte, 0, 1+len(room.SymmetricKey))
	bytes = append(bytes, room.Version)
	bytes = append(bytes, room.SymmetricKey...)
	return bytes
}

func (room *Room) ToRoomID() RoomID {
	if room.cachedRoomID == nil {
		bytes := room.Bytes()
		roomID := NewRoomID(base58.Encode(bytes))
		room.cachedRoomID = &roomID
	}
	return *room.cachedRoomID
}

func (room *Room) VersionSupported() bool {
	return room.Version == Version
}

func ParseRoomID(input string) (*Room, error) {
	decoded, err := base58.Decode(input)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode room id")
	}

	if len(decoded) < 1 {
		return nil, errors.New("room id is too short")
	}
	roomID := NewRoomID(input)
	room := &Room{
		Version:      decoded[0],
		cachedRoomID: &roomID,
	}

	if room.VersionSupported() {
		room.SymmetricKey = decoded[1:]
	}

	return room, nil
}

func NewRoom() (*Room, error) {
	symmetricKey, err := generateSymmetricKey()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate symmetric key")
	}
	return &Room{
		Version:      Version,
		SymmetricKey: symmetricKey,
		cachedRoomID: nil,
	}, nil
}

func generateSymmetricKey() ([]byte, error) {
	key := make([]byte, config.SymmetricKeyLength)
	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}
	return key, nil
}
