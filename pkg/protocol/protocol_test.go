package protocol

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/require"
)

func TestRoomID(t *testing.T) {
	sent, err := NewRoom()
	require.NoError(t, err)

	roomID := sent.ToRoomID()
	require.NotEmpty(t, roomID)

	received, err := ParseRoomID(roomID.String())
	require.NoError(t, err)
	require.NotEmpty(t, received)

	require.True(t, reflect.DeepEqual(sent, received))

	require.Equal(t, sent.Version, received.Version)
	require.Equal(t, sent.SymmetricKey, received.SymmetricKey)
}

func TestOnlineTimestampMigrationBackward(t *testing.T) {
	now := time.Now()

	player := Player{
		ID:              PlayerID(gofakeit.LetterN(5)),
		Name:            gofakeit.Username(),
		Online:          true,
		OnlineTimestamp: now,
	}

	payload, err := json.Marshal(player)
	require.NoError(t, err)

	var playerReceived Player
	err = json.Unmarshal(payload, &playerReceived)
	require.NoError(t, err)

	playerReceived.ApplyDeprecatedPatchOnReceive()

	require.Equal(t, player.ID, playerReceived.ID)
	require.Equal(t, player.Name, playerReceived.Name)
	require.Equal(t, now.UnixMilli(), playerReceived.OnlineTimestamp.UnixMilli())
	require.Equal(t, now.UnixMilli(), playerReceived.OnlineTimestampMilliseconds)
}

func TestOnlineTimestampMigrationForward(t *testing.T) {
	now := time.Now()

	player := Player{
		ID:                          PlayerID(gofakeit.LetterN(5)),
		Name:                        gofakeit.Username(),
		Online:                      true,
		OnlineTimestampMilliseconds: now.UnixMilli(),
	}

	player.ApplyDeprecatedPatchOnSend()

	payload, err := json.Marshal(player)
	require.NoError(t, err)

	var playerReceived Player
	err = json.Unmarshal(payload, &playerReceived)
	require.NoError(t, err)

	require.Equal(t, player.ID, playerReceived.ID)
	require.Equal(t, player.Name, playerReceived.Name)
	require.Equal(t, now.UnixMilli(), playerReceived.OnlineTimestamp.UnixMilli())
}
