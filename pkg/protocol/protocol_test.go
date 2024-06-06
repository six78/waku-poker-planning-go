package protocol

import (
	"encoding/json"
	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
	"time"
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

func TestOnlineTimestampMigration(t *testing.T) {
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

	playerReceived.ApplyDeprecatedPatch()

	require.Equal(t, player.ID, playerReceived.ID)
	require.Equal(t, player.Name, playerReceived.Name)
	require.Equal(t, player.OnlineTimestamp.UnixMilli(), playerReceived.OnlineTimestamp.UnixMilli())
	require.Equal(t, player.OnlineTimestamp.UnixMilli(), playerReceived.OnlineTimestampMilliseconds)
}
