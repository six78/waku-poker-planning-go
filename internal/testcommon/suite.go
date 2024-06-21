package testcommon

import (
	"encoding/json"
	"reflect"

	"github.com/brianvoe/gofakeit/v6"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/six78/2-story-points-cli/internal/config"
)

type Suite struct {
	suite.Suite
	Logger *zap.Logger
}

func (s *Suite) SetupSuite() {
	logger, err := zap.NewDevelopment()
	s.Require().NoError(err)
	s.Logger = logger
	config.Logger = logger
}

func (s *Suite) TearDownSuite() {
	_ = config.Logger.Sync()
}

func (s *Suite) SplitBatch(batch tea.Cmd) []tea.Cmd {
	s.Require().Equal(reflect.Func, reflect.TypeOf(batch).Kind())

	result := batch()
	s.Require().NotNil(result)

	batchMessage := result.(tea.BatchMsg)
	s.Require().NotNil(batchMessage)

	return batchMessage
}

func (s *Suite) FakePayload() ([]byte, []byte) {
	payload := make([]byte, 10)
	gofakeit.Slice(&payload)

	jsonPayload, err := json.Marshal(payload)
	s.Require().NoError(err)

	return payload, jsonPayload
}
