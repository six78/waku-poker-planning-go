package transport

import (
	"context"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	pp "github.com/six78/2-story-points-cli/pkg/protocol"
)

func TestEncryptionSuite(t *testing.T) {
	suite.Run(t, new(EncryptionSuite))
}

type EncryptionSuite struct {
	suite.Suite
	node   *Node
	cancel func()
}

func (s *EncryptionSuite) SetupSuite() {
	var ctx context.Context
	ctx, s.cancel = context.WithCancel(context.Background())

	logger, err := zap.NewDevelopment()
	s.Require().NoError(err)

	// Skip initialization, for this test we only need roomCache and logger
	s.node = NewNode(ctx, logger)
}

func (s *EncryptionSuite) TearDownSuite() {
	s.cancel()
}

func (s *EncryptionSuite) TestPublicEncryption() {
	room, err := pp.NewRoom()
	s.Require().NoError(err)

	payload := make([]byte, 100)
	gofakeit.Slice(payload)

	message, err := s.node.buildWakuMessage(room, payload)
	s.Require().NoError(err)

	err = s.node.encryptPublicPayload(room, message)
	s.Require().NoError(err)

	decryptedPayload, err := decryptMessage(room, message)
	s.Require().NoError(err)

	s.Require().Equal(payload, decryptedPayload)
}
