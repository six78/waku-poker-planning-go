package game

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"

	"github.com/six78/2-story-points-cli/internal/transport"
	"github.com/six78/2-story-points-cli/pkg/protocol"
	"github.com/six78/2-story-points-cli/pkg/storage"
)

var (
	ErrNoRoom = errors.New("no room")

	playerOnlineTimeout = 20 * time.Second
)

type StateSubscription chan *protocol.State

type Game struct {
	logger       *zap.Logger
	ctx          context.Context
	transport    transport.Service
	storage      storage.Service
	clock        clockwork.Clock
	exitRoom     chan struct{}
	messages     chan []byte
	features     FeatureFlags
	codeControls codeControlFlags

	isDealer bool
	player   *protocol.Player
	myVote   protocol.VoteResult // We save our vote to show it in UI

	room             *protocol.Room
	roomID           protocol.RoomID
	state            *protocol.State
	stateTimestamp   int64
	stateSubscribers []StateSubscription
	config           gameConfig
}

func NewGame(opts []Option) *Game {
	game := &Game{
		exitRoom:     nil,
		messages:     make(chan []byte, 42),
		features:     defaultFeatureFlags(),
		codeControls: defaultCodeControlFlags(),
		isDealer:     false,
		player:       nil,
		myVote: protocol.VoteResult{
			Value:     "",
			Timestamp: 0,
		},
		room:           nil,
		stateTimestamp: 0,
		config:         defaultConfig,
	}

	for _, opt := range opts {
		opt(game)
	}

	if game.ctx == nil {
		game.ctx = context.Background()
	}

	if game.logger == nil {
		game.logger = zap.NewNop()
	}

	if game.transport == nil {
		game.logger.Error("transport is required")
		return nil
	}

	if game.clock == nil {
		game.logger.Error("clock is required")
		return nil
	}

	return game
}

func (g *Game) Initialize() error {
	if g.HasStorage() {
		err := g.storage.Initialize()
		if err != nil {
			return errors.Wrap(err, "failed to create storage")
		}
	}

	player, err := g.loadPlayer(g.storage)
	if err != nil {
		return err
	}

	g.player = &protocol.Player{
		ID:     player.ID,
		Name:   player.Name,
		Online: true,
	}

	return nil
}

func (g *Game) LeaveRoom() {
	if g.room != nil {
		g.publishUserOnline(false)
	}

	if g.exitRoom != nil {
		close(g.exitRoom)
	}

	g.logger.Info("left room", zap.String("roomID", g.roomID.String()))

	g.exitRoom = nil
	g.isDealer = false
	g.room = nil
	g.roomID = protocol.NewRoomID("")
	g.state = nil
	g.stateTimestamp = 0
	g.notifyChangedState(false)
}

func (g *Game) Stop() {
	for _, subscriber := range g.stateSubscribers {
		close(subscriber)
	}
	g.stateSubscribers = nil
	g.LeaveRoom()
	// WARNING: wait for all routines to finish
}

func (g *Game) handleMessage(payload []byte) {
	g.logger.Debug("handling message", zap.String("payload", string(payload)))

	message := protocol.Message{}
	err := json.Unmarshal(payload, &message)
	if err != nil {
		g.logger.Error("failed to unmarshal message", zap.Error(err))
		return
	}
	logger := g.logger.With(zap.String("type", string(message.Type)))

	switch message.Type {
	case protocol.MessageTypeState:
		if !g.isDealer {
			g.handleStateMessage(payload)
		}

	case protocol.MessageTypePlayerOnline:
		if g.isDealer {
			g.handlePlayerOnlineMessage(payload)
		}

	case protocol.MessageTypePlayerOffline:
		if g.isDealer {
			g.handlePlayerOfflineMessage(payload)
		}

	case protocol.MessageTypePlayerVote:
		if g.isDealer {
			g.handlePlayerVoteMessage(payload)
		}

	default:
		logger.Warn("unsupported message type")
	}
}

func (g *Game) SubscribeToStateChanges() StateSubscription {
	channel := make(StateSubscription, 10)
	g.stateSubscribers = append(g.stateSubscribers, channel)
	return channel
}

func (g *Game) CurrentState() *protocol.State {
	return g.state
}

func (g *Game) notifyChangedState(publish bool) {
	state := g.hiddenCurrentState()

	g.logger.Debug("notifying state change",
		zap.Bool("publish", publish),
		zap.Int("subscribers", len(g.stateSubscribers)),
		zap.Any("state", state),
	)

	for _, subscriber := range g.stateSubscribers {
		subscriber <- state
	}

	if publish {
		go g.publishState(state)
	}
}

func (g *Game) publishOnlineState() {
	g.publishUserOnline(true)
	for {
		select {
		case <-g.clock.After(g.config.OnlineMessagePeriod):
			g.publishUserOnline(true)
		case <-g.exitRoom:
			return
		case <-g.ctx.Done():
			return
		}
	}
}

func (g *Game) publishStateLoop() {
	logger := g.logger.With(zap.String("source", "state publish loop"))
	logger.Debug("started")
	for {
		select {
		case <-g.clock.After(g.config.StateMessagePeriod):
			logger.Debug("tick")
			g.notifyChangedState(true)
		case <-g.exitRoom:
			logger.Debug("finished: room left")
			return
		case <-g.ctx.Done():
			logger.Debug("finished: ctx done")
			return
		}
	}
}

func (g *Game) watchPlayersStateLoop() {
	g.logger.Debug("check users state loop")
	ticker := g.clock.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-g.exitRoom:
			return
		case <-g.ctx.Done():
			return
		case <-ticker.Chan():
			if g.state == nil {
				continue
			}
			stateChanged := false
			now := g.clock.Now()
			for i, player := range g.state.Players {
				if !player.Online {
					continue
				}
				if now.Sub(player.OnlineTime()) <= playerOnlineTimeout {
					continue
				}
				g.logger.Info("marking user as offline",
					zap.Any("name", player.Name),
					zap.Any("lastSeenAt", player.OnlineTimestampMilliseconds),
					zap.Any("now", now),
				)
				g.state.Players[i].Online = false
				stateChanged = true
			}
			if stateChanged {
				g.notifyChangedState(true)
			}
		}
	}
}

func (g *Game) processIncomingMessages(sub *transport.MessagesSubscription) {
	if sub.Unsubscribe != nil {
		defer sub.Unsubscribe()
	}
	for {
		select {
		case payload, more := <-sub.Ch:
			if !more {
				return
			}
			g.handleMessage(payload)
		case <-g.exitRoom:
			return
		case <-g.ctx.Done():
			return
		}
	}
}

func (g *Game) loopPublishedMessages() {
	for {
		select {
		case <-g.exitRoom:
			return
		case <-g.ctx.Done():
			return
		case payload := <-g.messages:
			g.handleMessage(payload)
		}
	}
}

func (g *Game) publishMessage(message any) error {
	if g.room == nil {
		return ErrNoRoom
	}

	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}

	if g.config.EnableSymmetricEncryption {
		err = g.transport.PublishPublicMessage(g.room, payload)
	} else {
		err = g.transport.PublishUnencryptedMessage(g.room, payload)
	}

	// Loop message to ourselves
	if g.isDealer {
		g.messages <- payload
	}

	return err
}

func (g *Game) publishUserOnline(online bool) {
	timestamp := g.timestamp()

	g.logger.Debug("publishing online state",
		zap.Bool("online", online),
		zap.Int64("timestamp", timestamp),
	)

	var message interface{}

	player := *g.player
	player.ApplyDeprecatedPatchOnSend()

	if online {
		message = protocol.PlayerOnlineMessage{
			Player: player,
			Message: protocol.Message{
				Type:      protocol.MessageTypePlayerOnline,
				Timestamp: timestamp,
			},
		}
	} else {
		message = protocol.PlayerOfflineMessage{
			Player: player,
			Message: protocol.Message{
				Type:      protocol.MessageTypePlayerOffline,
				Timestamp: timestamp,
			},
		}
	}

	err := g.publishMessage(message)
	if err != nil {
		g.logger.Error("failed to publish online state", zap.Error(err))
	}
}

func (g *Game) PublishVote(vote protocol.VoteValue) error {
	if g.state.VoteState() != protocol.VotingState {
		return errors.New("no voting in progress")
	}
	if vote != "" && !slices.Contains(g.state.Deck, vote) {
		return fmt.Errorf("invalid vote")
	}
	g.logger.Debug("publishing vote", zap.Any("vote", vote))
	g.myVote = *protocol.NewVoteResult(vote)
	err := g.publishMessage(protocol.PlayerVoteMessage{
		Message: protocol.Message{
			Type:      protocol.MessageTypePlayerVote,
			Timestamp: g.timestamp(),
		},
		PlayerID:   g.player.ID,
		Issue:      g.state.ActiveIssue,
		VoteResult: g.myVote,
	})
	if err != nil {
		g.logger.Error("failed to publish vote", zap.Error(err))
		return err
	}
	return nil
}

func (g *Game) RetrieveVote() error {
	return g.PublishVote("")
}

func (g *Game) publishState(state *protocol.State) {
	if !g.isDealer {
		g.logger.Warn("only dealer can publish state")
		return
	}

	if state == nil {
		g.logger.Error("no state to publish")
		return
	}

	if g.HasStorage() && g.IsDealer() {
		err := g.storage.SaveRoomState(g.RoomID(), state)
		if err != nil {
			g.logger.Error("failed to save room state", zap.Error(err))
		}
	}

	g.logger.Debug("publishing state")
	err := g.publishMessage(protocol.GameStateMessage{
		Message: protocol.Message{
			Type:      protocol.MessageTypeState,
			Timestamp: g.timestamp(),
		},
		State: *state,
	})
	if err != nil {
		g.logger.Error("failed to publish state", zap.Error(err))
	}
}

func (g *Game) timestamp() int64 {
	return g.clock.Now().UnixMilli()
}

func (g *Game) Deal(input string) (protocol.IssueID, error) {
	if !g.isDealer {
		return "", errors.New("only dealer can deal")
	}

	if g.state.VoteState() == protocol.RevealedState {
		return "", errors.New("finish current vote to deal another issue")
	}

	issueID, err := g.addIssue(input)
	if err != nil {
		return "", errors.Wrap(err, "failed to add issue")
	}

	err = g.SelectIssue(len(g.state.Issues) - 1)

	return issueID, err
}

func (g *Game) CreateNewRoom() (*protocol.Room, *protocol.State, error) {
	room, err := protocol.NewRoom()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create a new room")
	}

	deckName := Fibonacci // FIXME: Remove hardcoded deck
	deck, deckFound := GetDeck(deckName)
	if !deckFound {
		return nil, nil, errors.Wrap(err, fmt.Sprintf("unknown deck '%s'", deckName))
	}

	state := &protocol.State{
		Players:     []protocol.Player{*g.player},
		Deck:        deck,
		ActiveIssue: "",
		Issues:      make([]*protocol.Issue, 0),
		Timestamp:   g.timestamp(),
	}

	return room, state, nil
}

func (g *Game) JoinRoom(roomID protocol.RoomID, state *protocol.State) error {
	if g.RoomID() == roomID {
		return errors.New("already in this room")
	}
	if g.room != nil {
		return errors.New("exit current room to join another one")
	}
	if roomID.Empty() {
		return errors.New("empty room ID")
	}

	room, err := protocol.ParseRoomID(roomID.String())
	if err != nil {
		return errors.Wrap(err, "failed to parse room ID")
	}
	if !room.VersionSupported() {
		return errors.Wrap(err, fmt.Sprintf("this room has unsupported version %d", room.Version))
	}

	if state == nil && g.HasStorage() {
		state = g.loadStateFromStorage(roomID)
	}

	g.exitRoom = make(chan struct{})
	g.isDealer = state != nil
	g.room = room
	g.roomID = roomID
	g.state = state
	g.stateTimestamp = 0
	if g.isDealer {
		g.state.Deck, _ = GetDeck(Fibonacci) // FIXME: remove hardcoded deck
	}

	g.resetMyVote()

	err = g.startRoutines()
	if err != nil {
		return errors.Wrap(err, "failed to start routines")
	}

	g.notifyChangedState(g.isDealer)

	if state == nil {
		g.logger.Info("joined room", zap.Any("roomID", roomID))
	} else {
		g.stateTimestamp = g.timestamp()
		g.logger.Info("loaded room", zap.Any("roomID", roomID), zap.Bool("isDealer", g.isDealer))
	}

	return nil
}

func (g *Game) startRoutines() error {
	sub, err := g.transport.SubscribeToMessages(g.room)
	if err != nil {
		return errors.Wrap(err, "failed to subscribe to messages")
	}

	go g.loopPublishedMessages()
	go g.processIncomingMessages(sub)

	if g.codeControls.EnablePublishOnlineState {
		go g.publishOnlineState()
	}

	if !g.isDealer {
		return nil
	}

	if g.config.PublishStateLoopEnabled {
		go g.publishStateLoop()
	}
	go g.watchPlayersStateLoop()

	return nil
}

func (g *Game) IsDealer() bool {
	return g.isDealer
}

func (g *Game) Room() protocol.Room {
	return *g.room
}

func (g *Game) RoomID() protocol.RoomID {
	return g.roomID
}

func (g *Game) Player() protocol.Player {
	return *g.player
}

func (g *Game) MyVote() protocol.VoteResult {
	return g.myVote
}

func (g *Game) RenamePlayer(name string) error {
	if g.HasStorage() {
		err := g.storage.SetPlayerName(name)
		if err != nil {
			return errors.Wrap(err, "failed to save player name")
		}
	}

	g.player.Name = name
	g.publishUserOnline(true)
	return nil
}

func (g *Game) Reveal() error {
	if !g.isDealer {
		return errors.New("only dealer can reveal cards")
	}

	if g.state.VoteState() != protocol.VotingState {
		return errors.New("cannot reveal when voting is not in progress")
	}

	g.state.VotesRevealed = true
	g.notifyChangedState(true)
	return nil
}

func (g *Game) hiddenCurrentState() *protocol.State {
	if g.state == nil {
		return nil
	}

	// Create a deep copy of the state
	hiddenState := *g.state

	if hiddenState.VoteState() != protocol.VotingState {
		return &hiddenState
	}

	hiddenState.Issues = make([]*protocol.Issue, 0, len(g.state.Issues))
	for _, item := range g.state.Issues {
		copiedItem := *item
		copiedItem.Votes = make(map[protocol.PlayerID]protocol.VoteResult, len(item.Votes))
		for playerID, vote := range item.Votes {
			if item.ID == g.state.ActiveIssue {
				copiedItem.Votes[playerID] = vote.Hidden()
			} else {
				copiedItem.Votes[playerID] = vote
			}
		}
		hiddenState.Issues = append(hiddenState.Issues, &copiedItem)
	}

	return &hiddenState
}

func (g *Game) SetDeck(deck protocol.Deck) error {
	if !g.features.EnableDeckSelection {
		return errors.New("deck selection is disabled")
	}
	if !g.isDealer {
		return errors.New("only dealer can set deck")
	}
	if g.state.VoteState() != protocol.IdleState && g.state.VoteState() != protocol.FinishedState {
		return errors.New("cannot set deck when voting is in progress")
	}
	g.state.Deck = deck
	g.notifyChangedState(true)
	return nil
}

func (g *Game) Finish(result protocol.VoteValue) error {
	if !g.isDealer {
		return errors.New("only dealer can finish")
	}
	if g.state.VoteState() != protocol.RevealedState {
		return errors.New("cannot finish when voting is not revealed")
	}
	if !slices.Contains(g.state.Deck, result) {
		return errors.New("result is not in the deck")
	}

	item := g.state.Issues.Get(g.state.ActiveIssue)
	if item == nil {
		return errors.New("vote item not found in the vote list")
	}

	item.Result = &result
	g.state.ActiveIssue = g.state.Issues.GetNextIssueToDeal(g.state.ActiveIssue)
	g.state.VotesRevealed = false
	g.resetMyVote()
	g.notifyChangedState(true)

	return nil
}

func (g *Game) resetMyVote() {
	g.myVote = protocol.VoteResult{
		Value:     "",
		Timestamp: 0,
	}
}

func (g *Game) AddIssue(titleOrURL string) (protocol.IssueID, error) {
	if !g.isDealer {
		return "", errors.New("only dealer can add issues")
	}
	issueID, err := g.addIssue(titleOrURL)
	if err != nil {
		return "", err
	}
	g.notifyChangedState(true)
	return issueID, nil
}

func (g *Game) addIssue(titleOrURL string) (protocol.IssueID, error) {
	issueID, err := GenerateIssueID()
	if err != nil {
		return "", errors.New("failed to generate UUID")
	}

	issueExist := slices.ContainsFunc(g.state.Issues, func(item *protocol.Issue) bool {
		return item.TitleOrURL == titleOrURL
	})
	if issueExist {
		return "", errors.New("issue already exists")
	}

	g.logger.Debug("adding issue", zap.String("titleOrUrl", titleOrURL))
	issue := protocol.Issue{
		ID:         issueID,
		TitleOrURL: titleOrURL,
		Votes:      make(map[protocol.PlayerID]protocol.VoteResult),
		Result:     nil,
	}

	g.state.Issues = append(g.state.Issues, &issue)
	return issue.ID, nil
}

func (g *Game) SelectIssue(index int) error {
	if !g.isDealer {
		return errors.New("only dealer can deal")
	}

	if g.state.VoteState() == protocol.RevealedState {
		return errors.New("cannot deal when voting is in progress")
	}

	if index < 0 || index >= len(g.state.Issues) {
		return errors.New("invalid issue index")
	}

	g.state.Issues[index].Result = nil
	g.state.Issues[index].Votes = make(map[protocol.PlayerID]protocol.VoteResult)
	g.state.ActiveIssue = g.state.Issues[index].ID
	g.notifyChangedState(true)

	return nil
}

func (g *Game) playerIndex(playerID protocol.PlayerID) int {
	return slices.IndexFunc(g.state.Players, func(player protocol.Player) bool {
		return player.ID == playerID
	})
}

func (g *Game) loadPlayer(s storage.Service) (*protocol.Player, error) {
	var err error
	var player protocol.Player

	// Load ID
	if !nilStorage(s) {
		player.ID = s.PlayerID()
	}

	if player.ID == "" {
		player.ID, err = GeneratePlayerID()
		if err != nil {
			return nil, errors.Wrap(err, "failed to generate player ID")
		}

		if !nilStorage(s) {
			err = s.SetPlayerID(player.ID)
			if err != nil {
				return nil, errors.Wrap(err, "failed to save player ID")
			}
		}
	}

	// Load Name
	if g.config.PlayerName != "" {
		player.Name = g.config.PlayerName
	} else if !nilStorage(s) {
		player.Name = s.PlayerName()
	}

	return &player, nil
}

func nilStorage(s storage.Service) bool {
	return s == nil || reflect.ValueOf(s).IsNil()
}

func (g *Game) HasStorage() bool {
	return !nilStorage(g.storage)
}

func (g *Game) loadStateFromStorage(roomID protocol.RoomID) *protocol.State {
	if !g.HasStorage() {
		return nil
	}
	state, err := g.storage.LoadRoomState(roomID)
	if err != nil {
		g.logger.Info("room not found in storage", zap.Error(err))
		return nil
	}
	g.logger.Info("loaded room from storage", zap.Any("roomID", roomID))

	// Mark players as offline if they haven't been seen for a while
	now := g.clock.Now()
	for i := range state.Players {
		online := now.Sub(state.Players[i].OnlineTime()) < playerOnlineTimeout
		state.Players[i].Online = online
	}

	return state
}
