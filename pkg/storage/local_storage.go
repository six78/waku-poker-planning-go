package storage

import (
	"encoding/json"
	"path"
	"sync"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/six78/2-story-points-cli/internal/config"
	"github.com/six78/2-story-points-cli/pkg/protocol"

	"github.com/shibukawa/configdir"
)

const (
	playerStorageFileName = "player.json"
	roomsDirectory        = "rooms"
)

var (
	ErrStorageUnmarshalFailed = errors.New("failed to unmarshal storage")
)

type LocalStorage struct {
	player playerStorage

	folder *configdir.Config
	mutex  *sync.RWMutex
}

type playerStorage struct {
	ID   protocol.PlayerID `json:"id"`
	Name string            `json:"name"`
}

type roomStorage struct {
	// TODO: PrivateKey string
	State *protocol.State `json:"state"`
}

func NewLocalStorage(localPath string) *LocalStorage {
	configDirs := configdir.New(config.VendorName, config.ApplicationName)
	configDirs.LocalPath = localPath

	return &LocalStorage{
		folder: queryFolder(&configDirs),
		mutex:  &sync.RWMutex{},
	}
}

func (s *LocalStorage) Initialize() error {
	if s.folder == nil {
		return errors.New("failed to find storage folder")
	}

	err := s.readPlayer()

	if errors.Is(err, ErrStorageUnmarshalFailed) {
		config.Logger.Error("failed to parse player storage, clearing storage", zap.Error(err))

		err = s.ResetPlayer()
		if err != nil {
			config.Logger.Error("failed to reset player storage", zap.Error(err))
		}
	}

	config.Logger.Info("storage initialized",
		zap.Any("player", s.player),
		zap.String("path", s.folder.Path),
		zap.Error(err),
	)

	return err
}

func (s *LocalStorage) readPlayer() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.folder.Exists(playerStorageFileName) {
		config.Logger.Info("no player storage found")
		return nil
	}

	data, err := s.folder.ReadFile(playerStorageFileName)
	if err != nil {
		return errors.Wrap(err, "failed to read player data")
	}

	err = json.Unmarshal(data, &s.player)
	if err != nil {
		return ErrStorageUnmarshalFailed
	}

	return nil
}

func (s *LocalStorage) savePlayerStorage() error {
	playerJson, err := json.Marshal(s.player)
	if err != nil {
		return errors.Wrap(err, "failed to marshal player storage")
	}

	err = s.folder.WriteFile(playerStorageFileName, playerJson)
	if err != nil {
		return errors.Wrap(err, "failed to save player storage")
	}

	return nil
}

func (s *LocalStorage) ResetPlayer() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.player.ID = ""
	s.player.Name = ""
	return s.savePlayerStorage()
}

func (s *LocalStorage) PlayerID() protocol.PlayerID {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.player.ID
}

func (s *LocalStorage) SetPlayerID(id protocol.PlayerID) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.player.ID = id
	return s.savePlayerStorage()
}

func (s *LocalStorage) PlayerName() string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.player.Name
}

func (s *LocalStorage) SetPlayerName(name string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.player.Name = name
	return s.savePlayerStorage()
}

func (s *LocalStorage) LoadRoomState(roomID protocol.RoomID) (*protocol.State, error) {
	filePath := roomFilePath(roomID)

	data, err := s.folder.ReadFile(filePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read room storage file")
	}

	var room roomStorage
	err = json.Unmarshal(data, &room)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal storage file")
	}

	return room.State, nil
}

func (s *LocalStorage) SaveRoomState(roomID protocol.RoomID, state *protocol.State) error {
	room := roomStorage{
		State: state,
	}

	roomJson, err := json.Marshal(room)
	if err != nil {
		return errors.Wrap(err, "failed to marshal room data")
	}

	filePath := roomFilePath(roomID)

	err = s.folder.WriteFile(filePath, roomJson)
	if err != nil {
		return errors.Wrap(err, "failed to write room storage")
	}

	return nil
}

func roomFilePath(roomID protocol.RoomID) string {
	return path.Join(roomsDirectory, roomID.String()+".json")
}

func queryFolder(configDirs *configdir.ConfigDir) *configdir.Config {
	configType := configdir.Global
	if configDirs.LocalPath != "" {
		configType = configdir.Local
	}

	folders := configDirs.QueryFolders(configType)
	if len(folders) == 0 {
		return nil
	}

	return folders[0]
}
