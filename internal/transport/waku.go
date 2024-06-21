package transport

import (
	"context"
	"encoding/hex"
	"net"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
	"github.com/waku-org/go-waku/waku/v2/dnsdisc"
	"github.com/waku-org/go-waku/waku/v2/node"
	wp "github.com/waku-org/go-waku/waku/v2/payload"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	wakuenr "github.com/waku-org/go-waku/waku/v2/protocol/enr"
	"github.com/waku-org/go-waku/waku/v2/protocol/lightpush"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/protocol/subscription"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"

	"github.com/six78/2-story-points-cli/internal/config"
	pp "github.com/six78/2-story-points-cli/pkg/protocol"
)

var fleets = map[string]string{
	"wakuv2.prod":  "enrtree://ANEDLO25QVUGJOUTQFRYKWX6P4Z4GKVESBMHML7DZ6YK4LGS5FC5O@prod.wakuv2.nodes.status.im",
	"wakuv2.test":  "enrtree://AO47IDOLBKH72HIZZOXQP6NMRESAN7CHYWIBNXDXWRJRZWLODKII6@test.wakuv2.nodes.status.im",
	"waku.sandbox": "enrtree://AIRVQ5DDA4FFWLRBCHJWUWOO6X6S4ZTZ5B667LQ6AJU6PEYDLRD5O@sandbox.waku.nodes.status.im",
	"shards.test":  "enrtree://AMOJVZX4V6EXP7NTJPMAYJYST2QP6AJXYW76IU6VGJS7UVSNDYZG4@boot.test.shards.nodes.status.im",
}

type Node struct {
	waku   *node.WakuNode
	ctx    context.Context
	logger *zap.Logger

	pubsubTopic          string
	wakuConnectionStatus chan node.ConnStatus
	roomCache            ContentTopicCache
	lightMode            bool
	statusSubscribers    []ConnectionStatusSubscription
	connectionStatus     ConnectionStatus
}

func NewNode(ctx context.Context, logger *zap.Logger) *Node {
	return &Node{
		waku:                 nil,
		ctx:                  ctx,
		logger:               logger,
		pubsubTopic:          relay.DefaultWakuTopic,
		wakuConnectionStatus: nil,
		roomCache:            NewRoomCache(logger),
		lightMode:            config.WakuLightMode(),
	}
}

func (n *Node) Initialize() error {
	hostAddr, err := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	if err != nil {
		return errors.Wrap(err, "failed to resolve TCP address")
	}

	var discoveredNodes []dnsdisc.DiscoveredNode
	if config.WakuDnsDiscovery() {
		discoveredNodes, err = discoverNodes(n.ctx, n.logger.Named("dnsdiscovery"))
		if err != nil {
			return errors.Wrap(err, "failed to discover nodes")
		}
	}

	wakuConnectionStatus := make(chan node.ConnStatus)

	options := []node.WakuNodeOption{
		node.WithLogger(n.logger.Named("waku")),
		//node.WithDNS4Domain(),
		node.WithLogLevel(zap.DebugLevel),
		node.WithHostAddress(hostAddr),
		node.WithConnectionStatusChannel(wakuConnectionStatus),
	}

	if config.WakuDiscV5() {
		bootNodes := getBootNodes(discoveredNodes)
		options = append(options,
			node.WithDiscoveryV5(0, bootNodes, true),
			node.WithPeerExchange(),
		)
	}

	if config.WakuLightMode() {
		options = append(options,
			node.WithLightPush(),
			node.WithWakuFilterLightNode(),
		)
	} else {
		options = append(options,
			node.WithWakuRelay(),
		)
	}

	options = append(options, node.DefaultWakuNodeOptions...)

	wakuNode, err := node.New(options...)
	if err != nil {
		return errors.Wrap(err, "failed to create waku node")
	}

	n.waku = wakuNode
	n.wakuConnectionStatus = wakuConnectionStatus
	n.logger = n.logger.Named("waku")

	return nil
}

func (n *Node) Start() error {
	if n.waku == nil {
		return errors.New("not initialized")
	}

	go n.watchConnectionStatus()

	err := n.waku.Start(n.ctx)
	if err != nil {
		return errors.Wrap(err, "failed to start waku node")
	}

	n.logger.Info("waku started", zap.String("peerID", n.waku.ID()))

	if config.WakuDiscV5() {
		n.logger.Debug("starting discoveryV5")
		err = n.waku.DiscV5().Start(context.Background())
		if err != nil {
			return errors.Wrap(err, "failed to start discoverV5")
		}
		n.logger.Debug("started discoveryV5")
	}

	if staticNodes := config.WakuStaticNodes(); len(staticNodes) != 0 {
		err = n.addStaticNodes(staticNodes)
		if err != nil {
			return errors.Wrap(err, "failed to add static nodes")
		}
	}

	n.logger.Info("waku node started")

	return nil
}

func getBootNodes(discoveredNodes []dnsdisc.DiscoveredNode) []*enode.Node {
	var bootNodes []*enode.Node
	for _, n := range discoveredNodes {
		if n.ENR != nil {
			bootNodes = append(bootNodes, n.ENR)
		}
	}
	return bootNodes
}

func (n *Node) Stop() {
	n.waku.Stop()
}

func parseEnrProtocols(v wakuenr.WakuEnrBitfield) string {
	var out []string
	if v&(1<<3) == 8 {
		out = append(out, "lightpush")
	}
	if v&(1<<2) == 4 {
		out = append(out, "filter")
	}
	if v&(1<<1) == 2 {
		out = append(out, "store")
	}
	if v&(1<<0) == 1 {
		out = append(out, "relay")
	}
	return strings.Join(out, ",")
}

func discoverNodes(ctx context.Context, logger *zap.Logger) ([]dnsdisc.DiscoveredNode, error) {
	enrTree, ok := fleets[config.Fleet()]
	if !ok {
		return nil, errors.Errorf("unknown fleet %s", config.Fleet())
	}

	// Otherwise run discovery
	var options []dnsdisc.DNSDiscoveryOption
	if nameserver := config.Nameserver(); nameserver != "" {
		options = append(options, dnsdisc.WithNameserver(nameserver))
	}

	discoveredNodes, err := dnsdisc.RetrieveNodes(ctx, enrTree, options...)
	if err != nil {
		return nil, err
	}

	logger.Debug("discovered nodes", zap.String("entree", enrTree))

	for _, d := range discoveredNodes {
		enrField := new(wakuenr.WakuEnrBitfield)
		err = d.ENR.Record().Load(enr.WithEntry(wakuenr.WakuENRField, &enrField))
		if err != nil {
			return nil, errors.Wrap(err, "failed to load waku enr field")
		}

		logger.Debug("discover node",
			zap.String("peerID", d.PeerID.String()),
			zap.Any("peerInfo", d.PeerInfo),
			zap.Any("protocols", parseEnrProtocols(*enrField)),
		)
	}

	return discoveredNodes, nil
}

func (n *Node) addStaticNodes(staticNodes []string) error {
	for _, staticNode := range staticNodes {
		n.logger.Info("connecting to a static store node",
			zap.String("address", staticNode),
		)
		addr, err := multiaddr.NewMultiaddr(staticNode)
		if err != nil {
			return errors.Wrap(err, "failed to parse multiaddr")
		}

		err = n.DialPeer(addr)
		if err != nil {
			return errors.Wrap(err, "failed to dial static peer")
		}
	}

	return nil
}

func (n *Node) DialPeer(address multiaddr.Multiaddr) error {
	const dialTimeout = 10 * time.Second

	ctx, cancel := context.WithTimeout(n.ctx, dialTimeout)
	defer cancel()

	return n.waku.DialPeerWithMultiAddress(ctx, address)
}

func (n *Node) PublishUnencryptedMessage(room *pp.Room, payload []byte) error {
	message, err := n.buildWakuMessage(room, payload)
	if err != nil {
		return errors.Wrap(err, "failed to build waku message")
	}
	return n.publishWakuMessage(message)
}

/*
	NOTE: Waku built-in encryption was a simple start, but it has a few disadvantages:
		1. It's fixed to 32-bytes key size
		   This makes RoomID too big even with a single SymmetricKey: "NrhbXYhn49Zo7LeKLQGQVRSjoBSLhLD6zSXKwqb3Podf"
		2. Because of this we have to pass pp.Room to this waku package.
		   I'm not sure if this is a good architecture decision.
*/

func (n *Node) encryptPublicPayload(room *pp.Room, message *pb.WakuMessage) error {
	keyInfo := &wp.KeyInfo{
		Kind:   wp.Symmetric,
		SymKey: room.SymmetricKey,
		// PrivKey: Set a privkey if the message requires a signature
	}

	return wp.EncodeWakuMessage(message, keyInfo)
}

func (n *Node) PublishPublicMessage(room *pp.Room, payload []byte) error {
	message, err := n.buildWakuMessage(room, payload)
	if err != nil {
		return errors.Wrap(err, "failed to build waku message")
	}

	err = n.encryptPublicPayload(room, message)
	if err != nil {
		return errors.Wrap(err, "failed to encrypt message")
	}

	return n.publishWakuMessage(message)
}

func (n *Node) PublishPrivateMessage(room *pp.Room, payload []byte) error {
	n.logger.Error("PublishPrivateMessage not implemented")
	return errors.New("PublishPrivateMessage not implemented")
}

func (n *Node) buildWakuMessage(room *pp.Room, payload []byte) (*pb.WakuMessage, error) {
	version := uint32(0)
	if config.EnableSymmetricEncryption {
		version = 1
	}

	contentTopic, err := n.roomCache.Get(room)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build content topic")
	}

	return &pb.WakuMessage{
		Payload:      payload,
		Version:      &version,
		ContentTopic: contentTopic,
		Timestamp:    utils.GetUnixEpoch(),
	}, nil
}

func (n *Node) publishWakuMessage(message *pb.WakuMessage) error {
	var err error
	var messageID []byte

	if n.lightMode {
		publishOptions := []lightpush.Option{
			lightpush.WithPubSubTopic(n.pubsubTopic),
		}
		messageID, err = n.waku.Lightpush().Publish(n.ctx, message, publishOptions...)
	} else {
		publishOptions := []relay.PublishOption{
			relay.WithPubSubTopic(n.pubsubTopic),
		}
		messageID, err = n.waku.Relay().Publish(n.ctx, message, publishOptions...)
	}

	if err != nil {
		n.logger.Error("failed to publish message", zap.Error(err))
		return errors.Wrap(err, "failed to publish message")
	}

	n.logger.Info("message sent",
		zap.String("messageID", hex.EncodeToString(messageID)))

	return nil
}

func (n *Node) watchConnectionStatus() {
	for {
		select {
		case <-n.ctx.Done():
			return
		case connStatus, more := <-n.wakuConnectionStatus:
			n.logger.Debug("waku connection status",
				zap.Bool("isOnline", connStatus.IsOnline),
				zap.Any("peersCount", len(connStatus.Peers)),
			)
			if !more {
				return
			}
			n.notifyConnectionStatus(&connStatus)
		}
	}
}

func (n *Node) SubscribeToMessages(room *pp.Room) (*MessagesSubscription, error) {
	n.logger.Debug("subscribing to room")

	contentTopic, err := n.roomCache.Get(room)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build content topic")
	}

	contentFilter := protocol.NewContentFilter(relay.DefaultWakuTopic, contentTopic)

	var in chan *protocol.Envelope
	var unsubscribe func()

	if n.lightMode {
		var subs []*subscription.SubscriptionDetails
		subs, err = n.waku.FilterLightnode().Subscribe(n.ctx, contentFilter)

		unsubscribe = func() {
			response, err := n.waku.FilterLightnode().Unsubscribe(n.ctx, contentFilter)
			if err != nil {
				n.logger.Warn("failed to unsubscribe from lightnode", zap.Error(err))
			}
			for _, err := range response.Errors() {
				n.logger.Warn("lightnode unsubscribe response error", zap.Error(err.Err))
			}
		}

		if err != nil {
			n.logger.Error("failed to subscribe to content topic", zap.Bool("lightMode", n.lightMode), zap.Error(err))
			return nil, errors.Wrap(err, "failed to subscribe to content topic")
		}

		if len(subs) != 1 {
			if len(subs) > 0 {
				unsubscribe()
			}
			err = errors.Errorf("unexpected number of subscriptions: %d", len(subs))
			n.logger.Error("failed to subscribe to content topic", zap.Error(err))
			return nil, err
		}

		in = subs[0].C

	} else {
		var subs []*relay.Subscription
		subs, err = n.waku.Relay().Subscribe(n.ctx, contentFilter)

		unsubscribe = func() {
			err := n.waku.Relay().Unsubscribe(n.ctx, contentFilter)
			if err != nil {
				n.logger.Warn("failed to unsubscribe from relay", zap.Error(err))
			}
			// WARNING: Why 0 peers after this?
			// FIXME: Open a go-waku PR. This unregister should be called in WakuRelay.RemoveTopicValidator
			err = n.waku.Relay().PubSub().UnregisterTopicValidator(contentFilter.PubsubTopic)
			if err != nil {
				n.logger.Warn("failed to unregister topic validator")
			}
		}

		if err != nil {
			n.logger.Error("failed to subscribe to content topic", zap.Bool("lightMode", n.lightMode), zap.Error(err))
			return nil, errors.Wrap(err, "failed to subscribe to content topic")
		}

		if len(subs) != 1 {
			if len(subs) > 0 {
				unsubscribe()
			}
			err = errors.Errorf("unexpected number of subscriptions: %d", len(subs))
			n.logger.Error("failed to subscribe to content topic", zap.Error(err))
			return nil, err
		}

		in = subs[0].Ch
	}

	leaveRoom := make(chan struct{})
	sub := &MessagesSubscription{
		Ch: make(chan []byte, 10),
		Unsubscribe: func() {
			close(leaveRoom)
		},
	}

	go func() {
		defer func() {
			unsubscribe()
			close(sub.Ch)
			n.logger.Debug("subscription channel closed")
		}()

		for {
			select {
			case <-leaveRoom:
				return
			case value := <-in:
				n.logger.Info("waku message received (relay)",
					zap.String("payload", string(value.Message().Payload)),
				)
				payload, err := decryptMessage(room, value.Message())
				if err != nil {
					n.logger.Warn("failed to decrypt message payload")
				}

				sub.Ch <- payload
			}
		}
	}()

	return sub, nil
}

func decryptMessage(room *pp.Room, message *pb.WakuMessage) ([]byte, error) {
	// NOTE: waku automatically decide to decrypt or not based on message.Version (0/1)
	//if !config.EnableSymmetricEncryption {
	//	return payload, nil
	//}

	keyInfo := &wp.KeyInfo{
		Kind:   wp.Symmetric,
		SymKey: room.SymmetricKey,
	}

	err := wp.DecodeWakuMessage(message, keyInfo)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode waku message")
	}

	return message.Payload, nil
}

func (n *Node) ConnectionStatus() ConnectionStatus {
	return n.connectionStatus
}

func (n *Node) SubscribeToConnectionStatus() ConnectionStatusSubscription {
	channel := make(ConnectionStatusSubscription, 10)
	n.statusSubscribers = append(n.statusSubscribers, channel)
	return channel
}

func (n *Node) notifyConnectionStatus(s *node.ConnStatus) {
	status := ConnectionStatus{
		IsOnline:   s.IsOnline,
		HasHistory: s.HasHistory,
		PeersCount: len(s.Peers),
	}

	n.connectionStatus = status

	for _, subscriber := range n.statusSubscribers {
		subscriber <- status
	}
}
