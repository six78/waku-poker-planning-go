package game

import (
	"encoding/json"

	"go.uber.org/zap"
	"golang.org/x/exp/slices"

	"github.com/six78/2-story-points-cli/pkg/protocol"
)

func (g *Game) handleStateMessage(payload []byte) {
	var message protocol.GameStateMessage
	err := json.Unmarshal(payload, &message)
	if err != nil {
		g.logger.Error("failed to unmarshal message", zap.Error(err))
		return
	}

	g.logger.Info("state message received", zap.Any("state", message.State))

	if g.state != nil && message.State.ActiveIssue != g.state.ActiveIssue {
		// Voting finished or new issue dealt. Reset our vote.
		g.resetMyVote()
	}

	g.state = &message.State
	g.state.Deck, _ = GetDeck(Fibonacci) // FIXME: remove hardcoded deck
	g.notifyChangedState(false)
}

func (g *Game) handlePlayerOnlineMessage(payload []byte) {
	var message protocol.PlayerOnlineMessage
	err := json.Unmarshal(payload, &message)
	if err != nil {
		g.logger.Error("failed to unmarshal message", zap.Error(err))
		return
	}

	g.logger.Info("player online message received", zap.Any("player", message.Player))
	message.Player.ApplyDeprecatedPatchOnReceive()

	// TODO: Store player pointers in a map

	index := g.playerIndex(message.Player.ID)
	if index < 0 {
		message.Player.Online = true
		message.Player.OnlineTimestampMilliseconds = g.timestamp()
		g.state.Players = append(g.state.Players, message.Player)
		g.notifyChangedState(true)
		g.logger.Info("player joined", zap.Any("player", message.Player))
		return
	}

	playerChanged := !g.state.Players[index].Online ||
		g.state.Players[index].Name != message.Player.Name

	g.state.Players[index].OnlineTimestampMilliseconds = g.timestamp()

	if !playerChanged {
		return
	}

	g.state.Players[index].Online = true
	g.state.Players[index].Name = message.Player.Name
	g.notifyChangedState(true)
}

func (g *Game) handlePlayerOfflineMessage(payload []byte) {
	var message protocol.PlayerOfflineMessage
	err := json.Unmarshal(payload, &message)
	if err != nil {
		g.logger.Error("failed to unmarshal message", zap.Error(err))
		return
	}

	g.logger.Info("player is offline", zap.Any("player", message.Player))
	index := g.playerIndex(message.Player.ID)
	if index < 0 {
		return
	}

	g.state.Players[index].Online = false
	g.notifyChangedState(true)
}

func (g *Game) handlePlayerVoteMessage(payload []byte) {
	var message protocol.PlayerVoteMessage
	err := json.Unmarshal(payload, &message)

	if err != nil {
		g.logger.Error("failed to unmarshal message", zap.Error(err))
		return
	}

	logger := g.logger.With(zap.Any("playerID", message.PlayerID))
	logger.Info("player vote message received")

	if g.state.VoteState() != protocol.VotingState {
		g.logger.Warn("player vote ignored as not in voting state")
		return
	}

	if message.VoteResult.Value != "" && !slices.Contains(g.state.Deck, message.VoteResult.Value) {
		logger.Warn("player vote ignored as not found in deck",
			zap.Any("vote", message.VoteResult),
			zap.Any("deck", g.state.Deck))
		return
	}

	if g.state.ActiveIssue != message.Issue {
		logger.Warn("player vote ignored as not for the current vote item",
			zap.Any("voteFor", message.Issue),
			zap.Any("currentVoteItemID", g.state.ActiveIssue),
		)
		return
	}

	item := g.state.Issues.Get(message.Issue)
	if item == nil {
		g.logger.Error("vote item not found", zap.Any("voteFor", message.Issue))
		return
	}

	currentVote, voteExist := item.Votes[message.PlayerID]
	if voteExist && currentVote.Timestamp >= message.Timestamp {
		logger.Warn("player vote ignored as outdated",
			zap.Any("currentVote", currentVote),
			zap.Any("receivedVote", message.VoteResult),
		)
		return
	}

	logger.Info("player vote accepted",
		zap.Any("voteFor", message.Issue),
		zap.Any("voteResult", message.VoteResult.Value),
		zap.Any("timestamp", message.Timestamp),
	)

	if message.VoteResult.Value == "" {
		delete(item.Votes, message.PlayerID)
	} else {
		item.Votes[message.PlayerID] = message.VoteResult
	}

	g.notifyChangedState(true)
}
