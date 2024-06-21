package storage

import (
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/shibukawa/configdir"
	"github.com/stretchr/testify/suite"

	"github.com/six78/2-story-points-cli/internal/config"
	"github.com/six78/2-story-points-cli/internal/testcommon"
	"github.com/six78/2-story-points-cli/pkg/protocol"
)

func TestLocalStorage(t *testing.T) {
	suite.Run(t, &Suite{})
}

type Suite struct {
	testcommon.Suite
	storage  *LocalStorage
	tempPath string
}

func (s *Suite) SetupTest() {
	var err error
	s.tempPath = s.T().TempDir()
	s.storage = NewLocalStorage(s.tempPath)
	s.Require().NotNil(s.storage)
	err = s.storage.Initialize()
	s.Require().NoError(err)
}

func (s *Suite) TestLocalPath() {
	localPath := s.T().TempDir()

	storage := NewLocalStorage(localPath)
	s.Require().NotNil(storage)

	err := storage.Initialize()
	s.Require().NoError(err)

	s.Require().NotNil(storage)
	s.Require().NotNil(storage.folder)
	s.Require().Equal(localPath, storage.folder.Path)
}

func (s *Suite) TestGlobalPath() {
	configDirs := configdir.New(config.VendorName, config.ApplicationName)
	folders := configDirs.QueryFolders(configdir.Global)
	s.Require().NotEmpty(folders)

	folder := folders[0]
	s.Require().NotNil(folder)

	storage := NewLocalStorage("")
	err := s.storage.Initialize()
	s.Require().NoError(err)
	s.Require().NotNil(storage)
	s.Require().NotNil(storage.folder)
	s.Require().Equal(folder.Path, storage.folder.Path)
}

func (s *Suite) TestPlayerStorage() {
	s.Require().Empty(s.storage.PlayerID())
	s.Require().Empty(s.storage.PlayerName())

	id := protocol.PlayerID(gofakeit.LetterN(5))
	err := s.storage.SetPlayerID(id)
	s.Require().NoError(err)
	s.Require().Equal(id, s.storage.PlayerID())
	s.Require().Empty(s.storage.PlayerName())

	name := gofakeit.LetterN(6)
	err = s.storage.SetPlayerName(name)
	s.Require().NoError(err)
	s.Require().Equal(id, s.storage.PlayerID())
	s.Require().Equal(name, s.storage.PlayerName())
}

func (s *Suite) TestRoomStorage() {
	roomID := protocol.NewRoomID(gofakeit.LetterN(5))
	state, err := s.storage.LoadRoomState(roomID)
	s.Require().Error(err)
	s.Require().Nil(state)

	state = &protocol.State{}
	err = gofakeit.Struct(state)
	s.Require().NoError(err)

	err = s.storage.SaveRoomState(roomID, state)
	s.Require().NoError(err)

	resetPlayersTimestamps := func(state *protocol.State) {
		for i := range state.Players {
			state.Players[i].OnlineTimestampMilliseconds = 0
		}
	}

	// Reset fields that are not saved
	loadedState, err := s.storage.LoadRoomState(roomID)
	resetPlayersTimestamps(state)
	resetPlayersTimestamps(loadedState)
	loadedState.Deck = state.Deck
	loadedState.Timestamp = state.Timestamp

	s.Require().NoError(err)
	s.Require().Equal(state, loadedState)
}

func (s *Suite) TestResetPlayer() {
	id := protocol.PlayerID(gofakeit.LetterN(5))
	name := gofakeit.LetterN(6)

	err := s.storage.SetPlayerID(id)
	s.Require().NoError(err)
	s.Require().Equal(id, s.storage.PlayerID())

	err = s.storage.SetPlayerName(name)
	s.Require().NoError(err)
	s.Require().Equal(name, s.storage.PlayerName())

	err = s.storage.ResetPlayer()
	s.Require().NoError(err)
	s.Require().Empty(s.storage.PlayerID())
	s.Require().Empty(s.storage.PlayerName())
}

func (s *Suite) TestResetPlayerOnUnmarshalFailure() {
	// Set up a valid player storage
	id := protocol.PlayerID(gofakeit.LetterN(5))
	name := gofakeit.LetterN(6)

	err := s.storage.SetPlayerID(id)
	s.Require().NoError(err)
	err = s.storage.SetPlayerName(name)
	s.Require().NoError(err)

	// Check that the player storage is valid
	s.Require().Equal(id, s.storage.PlayerID())
	s.Require().Equal(name, s.storage.PlayerName())

	// Write invalid JSON to the player storage file
	err = s.storage.folder.WriteFile(playerStorageFileName, []byte("{invalid json"))
	s.Require().NoError(err)

	// Create a new storage (with same path) to ensure that the player storage was reset
	newStorage := NewLocalStorage(s.tempPath)
	err = s.storage.Initialize()
	s.Require().NoError(err)
	s.Require().NotNil(newStorage)
	s.Require().Empty(newStorage.PlayerID())
	s.Require().Empty(newStorage.PlayerName())
}
