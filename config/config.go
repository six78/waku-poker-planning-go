package config

import (
	"flag"
	"fmt"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"strings"
	"time"
	"waku-poker-planning/protocol"
)

const OnlineMessagePeriod = 5 * time.Second
const StateMessagePeriod = 10 * time.Second
const logsDirectory = "logs"
const SymmetricKeyLength = 16

var fleet string
var playerName string
var initialAction string
var debug bool
var anonymous bool

var Logger *zap.Logger
var LogFilePath string

func SetupLogger() {
	LogFilePath = createLogFile()
	config := zap.NewDevelopmentConfig()
	config.OutputPaths = []string{LogFilePath}
	config.Development = false
	logger, err := config.Build()
	if err != nil {
		panic(err)
	}
	Logger = logger
}

func createLogFile() string {
	executableFilePath, err := os.Executable()
	if err != nil {
		panic(err)
	}

	executableDir := filepath.Dir(executableFilePath)
	name := fmt.Sprintf("waku-pp-%s.log", time.Now().Format(time.RFC3339))
	path := filepath.Join(executableDir, logsDirectory, name)

	if err := os.MkdirAll(filepath.Dir(path), 0770); err != nil {
		panic(err)
	}

	if _, err := os.Create(path); err != nil {
		panic(err)
	}

	return path
}

func ParseArguments() {
	flag.StringVar(&fleet, "fleet", "wakuv2.prod", "Waku fleet name")
	flag.StringVar(&playerName, "name", "", "Player name")
	flag.BoolVar(&debug, "debug", false, "Show debug info")
	flag.BoolVar(&anonymous, "anonymous", false, "Anonymouse mode")
	flag.Parse()

	initialAction = strings.Join(flag.Args(), " ")
}

func GeneratePlayerName() string {
	return fmt.Sprintf("player-%d", time.Now().Unix())
}

func Fleet() string {
	return fleet
}

func PlayerName() string {
	return playerName
}

func InitialAction() string {
	return initialAction
}

func Debug() bool {
	return debug
}

func Anonymous() bool {
	return anonymous
}

func GeneratePlayerID() (protocol.PlayerID, error) {
	playerUUID, err := uuid.NewUUID()
	if err != nil {
		return "", errors.Wrap(err, "failed to generate player UUID")
	}
	return protocol.PlayerID(playerUUID.String()), nil
}
