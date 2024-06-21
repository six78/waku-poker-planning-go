package transport

import (
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
	waku "github.com/waku-org/go-waku/waku/v2/protocol"
	"go.uber.org/zap"

	"github.com/six78/2-story-points-cli/internal/config"
	"github.com/six78/2-story-points-cli/pkg/protocol"
)

type ContentTopicCache struct {
	logger       *zap.Logger
	roomID       *protocol.RoomID
	contentTopic string
	err          error
	hits         int
}

func NewRoomCache(logger *zap.Logger) ContentTopicCache {
	return ContentTopicCache{
		logger:       logger.Named("TopicCache"),
		roomID:       nil,
		contentTopic: "",
		hits:         0,
	}
}

func (r *ContentTopicCache) Get(room *protocol.Room) (string, error) {
	roomID := room.ToRoomID()
	if r.roomID != nil && *r.roomID == roomID {
		r.hits++
		return r.contentTopic, r.err
	}

	r.roomID = &roomID
	r.contentTopic, r.err = r.roomContentTopic(room)
	r.hits = 0

	if r.err != nil {
		r.logger.Error("failed to calculate content topic", zap.Error(r.err))
	} else {
		r.logger.Debug("new content topic", zap.String("contentTopic", r.contentTopic))
	}

	return r.contentTopic, r.err
}

func (r *ContentTopicCache) roomContentTopic(room *protocol.Room) (string, error) {
	version := strconv.Itoa(int(protocol.Version))
	hash := crypto.Keccak256(room.Bytes())
	contentTopicName := hexutil.Encode(hash[:4])[2:]

	// FIXME: Change vendor name to application name here?
	// TODO: Switch to protobuf
	contentTopic, err := waku.NewContentTopic(config.VendorName, version, contentTopicName, "json") // WARNING: "six78" is not the name of the app

	if err != nil {
		return "", errors.Wrap(err, "failed to create content topic")
	}

	return contentTopic.String(), nil
}
