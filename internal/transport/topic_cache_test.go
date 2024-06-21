package transport

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/six78/2-story-points-cli/pkg/protocol"
)

func TestContentTopicCache(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	cache := NewRoomCache(logger)

	room1, err := protocol.NewRoom()
	require.NoError(t, err)

	room1ContentTopic, err := cache.roomContentTopic(room1)
	require.NoError(t, err)

	// First call to Get
	contentTopic1, err := cache.Get(room1)
	require.NoError(t, err)
	require.Equal(t, room1ContentTopic, contentTopic1)
	require.Equal(t, 0, cache.hits)

	// Second call to Get, hit cache
	for i := range [3]int{} {
		contentTopic2, err2 := cache.Get(room1)
		require.NoError(t, err2)
		require.Equal(t, room1ContentTopic, contentTopic2)
		require.Equal(t, i+1, cache.hits)
	}

	room2, err := protocol.NewRoom()
	require.NoError(t, err)

	room2ContentTopic, err := cache.roomContentTopic(room2)
	require.NoError(t, err)

	contentTopic2, err := cache.Get(room2)
	require.NoError(t, err)
	require.Equal(t, room2ContentTopic, contentTopic2)
	require.Equal(t, 0, cache.hits)
}
