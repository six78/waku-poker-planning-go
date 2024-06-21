package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/shibukawa/configdir"
	"go.uber.org/zap"
)

const OnlineMessagePeriod = 5 * time.Second
const StateMessagePeriod = 30 * time.Second
const logsDirectory = "logs"
const SymmetricKeyLength = 32
const EnableSymmetricEncryption = true

const VendorName = "six78"
const ApplicationName = "2sp"

const UserColor = lipgloss.Color("#7D56F4")
const ForegroundShadeColor = lipgloss.Color("#555555")

var fleet string
var nameserver string
var playerName string
var initialAction string
var debug bool
var anonymous bool
var wakuStaticNodes StaticWakuNodes
var wakuLightMode bool
var wakuDiscV5 bool
var wakuDnsDiscovery bool

var Logger *zap.Logger
var LogFilePath string

type StaticWakuNodes []string

func (n *StaticWakuNodes) String() string {
	return strings.Join(*n, ",")
}

func (n *StaticWakuNodes) Set(value string) error {
	*n = append(*n, value)
	return nil
}

func SetupLogger() {
	var c zap.Config
	if debug {
		c = zap.NewDevelopmentConfig()
	} else {
		c = zap.NewProductionConfig()
	}

	LogFilePath = createLogFile()
	c.OutputPaths = []string{LogFilePath}
	c.Development = false
	logger, err := c.Build()
	if err != nil {
		panic(err)
	}
	Logger = logger
}

func createLogFile() string {
	//executableFilePath, err := os.Executable()
	//if err != nil {
	//	panic(err)
	//}

	//s.configDirs.QueryFolderContainsFile(playerFileName).Path)

	name := fmt.Sprintf("waku-pp-%s.log", time.Now().UTC().Format(time.RFC3339))
	name = strings.Replace(name, ":", "-", -1)
	//executableDir := filepath.Dir(executableFilePath)

	configDirs := configdir.New(VendorName, ApplicationName)
	folders := configDirs.QueryFolders(configdir.Global)
	path := filepath.Join(folders[0].Path, logsDirectory, name)
	//err = folders[0].WriteFile(path, []byte(playerUUID))

	if err := os.MkdirAll(filepath.Dir(path), 0770); err != nil {
		panic(err)
	}

	if _, err := os.Create(path); err != nil {
		panic(err)
	}

	return path
}

func ParseArguments() {
	flag.StringVar(&playerName, "name", "", "Player name")
	flag.BoolVar(&debug, "debug", false, "Show debug info")
	flag.BoolVar(&anonymous, "anonymous", false, "Anonymous mode")
	flag.StringVar(&fleet, "waku.fleet", "shards.test", "Waku fleet name")
	flag.StringVar(&nameserver, "waku.nameserver", "", "Waku nameserver")
	flag.Var(&wakuStaticNodes, "waku.staticnode", "Waku static node multiaddress")
	flag.BoolVar(&wakuLightMode, "waku.lightmode", false, "Waku lightpush/filter mode")
	flag.BoolVar(&wakuDiscV5, "waku.discv5", true, "Enable DiscV5 discovery")
	flag.BoolVar(&wakuDnsDiscovery, "waku.dnsdiscovery", true, "Enable DNS discovery")
	flag.Parse()

	//// NOTE: wakuv2.test ENRtree returns invalid URLs, define static nodes
	//if fleet == "wakuv2.test" {
	//	fleet = ""
	//	wakuStaticNodes = append(wakuStaticNodes, "/dns4/node-01.do-ams3.wakuv2.test.status.im/tcp/30303/p2p/16Uiu2HAmPLe7Mzm8TsYUubgCAW1aJoeFScxrLj8ppHFivPo97bUZ")
	//	wakuStaticNodes = append(wakuStaticNodes, "/dns4/node-01.gc-us-central1-a.wakuv2.test.status.im/tcp/30303/p2p/16Uiu2HAmJb2e28qLXxT5kZxVUUoJt72EMzNGXB47Rxx5hw3q4YjS")
	//	wakuStaticNodes = append(wakuStaticNodes, "/dns4/node-01.ac-cn-hongkong-c.wakuv2.test.status.im/tcp/8000/wss/p2p/16Uiu2HAkvWiyFsgRhuJEb9JfjYxEkoHLgnUQmr1N5mKWnYjxYRVm")
	//}

	initialAction = strings.Join(flag.Args(), " ")
}

func GeneratePlayerName() string {
	return fmt.Sprintf("player-%d", time.Now().Unix())
}

func Fleet() string {
	return fleet
}

func Nameserver() string {
	return nameserver
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

func WakuStaticNodes() []string {
	return wakuStaticNodes
}

func WakuLightMode() bool {
	return wakuLightMode
}

func WakuDiscV5() bool {
	return wakuDiscV5
}

func WakuDnsDiscovery() bool {
	return wakuDnsDiscovery
}
