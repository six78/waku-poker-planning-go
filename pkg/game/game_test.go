package game

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/six78/2-story-points-cli/internal/testcommon"
	"github.com/six78/2-story-points-cli/internal/testcommon/matchers"
	"github.com/six78/2-story-points-cli/internal/transport"
	mocktransport "github.com/six78/2-story-points-cli/internal/transport/mock"
	"github.com/six78/2-story-points-cli/pkg/protocol"
)

func TestGame(t *testing.T) {
	suite.Run(t, new(Suite))
}

type Suite struct {
	testcommon.Suite

	ctx       context.Context
	cancel    context.CancelFunc
	transport *mocktransport.MockService
	clock     clockwork.FakeClock
	dealer    *Game
}

func (s *Suite) newGame(extraOptions []Option) *Game {
	options := []Option{
		WithContext(s.ctx),
		WithTransport(s.transport),
		WithClock(s.clock),
		WithLogger(s.Logger),
		WithPlayerName(gofakeit.Username()),
		WithPublishStateLoop(false),
	}
	options = append(options, extraOptions...)

	g := NewGame(options)
	s.Require().NotNil(g)

	err := g.Initialize()
	s.Require().NoError(err)

	return g
}

func (s *Suite) SetupTest() {
	s.ctx, s.cancel = context.WithCancel(context.Background())

	ctrl := gomock.NewController(s.T())
	s.transport = mocktransport.NewMockService(ctrl)
	s.clock = clockwork.NewFakeClock()

	s.dealer = s.newGame([]Option{
		WithEnableSymmetricEncryption(true),
	})

	err := s.dealer.Initialize()
	s.Require().NoError(err)
}

func (s *Suite) TearDownTest() {
	s.cancel()
}

func (s *Suite) newStateMatcher() *matchers.StateMatcher {
	return matchers.NewStateMatcher(s.T(), nil)
}

func (s *Suite) expectSubscribeToMessages(room *protocol.Room) func(room *protocol.Room, payload []byte) {
	roomMatcher := matchers.NewRoomMatcher(room)

	subscription := &transport.MessagesSubscription{
		Ch:          make(chan []byte),
		Unsubscribe: func() {},
	}

	s.transport.EXPECT().SubscribeToMessages(roomMatcher).
		Return(subscription, nil).
		Times(1)

	return func(room *protocol.Room, payload []byte) {
		subscription.Ch <- payload
	}
}

func (s *Suite) TestStateSize() {
	const playersCount = 20
	const issuesCount = 30

	state := protocol.State{
		Players: make([]protocol.Player, 0, playersCount),
		Issues:  make(protocol.IssuesList, 0, issuesCount),
	}

	votes := make(map[protocol.PlayerID]protocol.VoteResult, playersCount)
	deck, deckFound := GetDeck(Fibonacci)
	s.Require().True(deckFound)

	state.Deck = deck

	for i := 0; i < playersCount; i++ {
		playerID, err := GeneratePlayerID()
		s.Require().NoError(err)

		state.Players = append(state.Players, protocol.Player{
			ID:   playerID,
			Name: fmt.Sprintf("player-%d", i),
		})

		vote := fmt.Sprintf("%d", i%len(deck))
		votes[playerID] = *protocol.NewVoteResult(protocol.VoteValue(vote))
	}

	for i := 0; i < issuesCount; i++ {
		issueID, err := GenerateIssueID()
		s.Require().NoError(err)

		state.Issues = append(state.Issues, &protocol.Issue{
			ID:         issueID,
			TitleOrURL: fmt.Sprintf("https://github.com/six78/waku-poker-planing/issues/%d", i),
			Votes:      votes, // same votes for each issue, whatever
			Result:     &deck[i%len(deck)],
		})
	}

	stateMessage, err := json.Marshal(state)
	s.Require().NoError(err)

	fmt.Println("state size", "bytes", len(stateMessage))
	s.Require().Less(len(stateMessage), 100*1024, "state size should be less than 100 kilobytes")
}

func (s *Suite) TestSimpleGame() {
	room, initialState, err := s.dealer.CreateNewRoom()
	s.Require().NoError(err)
	s.Require().NotNil(room)

	roomID := room.ToRoomID()
	roomMatcher := matchers.NewRoomMatcher(room)
	onlineMatcher := matchers.NewOnlineMatcher(s.T(), s.dealer.Player().ID)

	// Online state is sent periodically
	s.transport.EXPECT().PublishPublicMessage(roomMatcher, onlineMatcher).AnyTimes()

	s.expectSubscribeToMessages(room)

	// Join room
	stateMatcher := s.newStateMatcher()
	s.transport.EXPECT().PublishPublicMessage(roomMatcher, stateMatcher).
		Times(1)

	err = s.dealer.JoinRoom(roomID, initialState)
	s.Require().NoError(err)

	state := stateMatcher.Wait()
	s.Require().False(state.VotesRevealed)
	s.Require().Empty(state.ActiveIssue)
	s.Require().Len(state.Players, 1)
	s.Logger.Info("match on join room")

	// Deal first vote item

	firstItemText := gofakeit.LetterN(10)
	const dealerVote = protocol.VoteValue("1")

	var firstIssueID protocol.IssueID

	checkIssues := func(issuesList protocol.IssuesList) *protocol.Issue {
		s.Require().Len(issuesList, 1)
		item := issuesList.Get(firstIssueID)
		s.Require().NotNil(item)
		s.Require().Equal(firstItemText, item.TitleOrURL)
		return item
	}

	{ // Deal first vote item
		stateMatcher = s.newStateMatcher()

		s.transport.EXPECT().
			PublishPublicMessage(roomMatcher, stateMatcher).
			Times(1)

		firstIssueID, err = s.dealer.Deal(firstItemText)
		s.Require().NoError(err)

		state = stateMatcher.Wait()
		item := checkIssues(state.Issues)
		s.Require().Nil(item.Result)
		s.Require().Empty(item.Votes)
		s.Logger.Info("match on deal first item")
	}

	currentIssue := s.dealer.CurrentState().Issues.Get(s.dealer.CurrentState().ActiveIssue)
	s.Require().NotNil(currentIssue)
	s.Require().Equal(firstItemText, currentIssue.TitleOrURL)

	{ // Publish dealer vote
		voteMatcher := matchers.NewVoteMatcher(s.dealer.Player().ID, currentIssue.ID, dealerVote)
		s.transport.EXPECT().
			PublishPublicMessage(roomMatcher, voteMatcher).
			Times(1)

		stateMatcher = s.newStateMatcher()
		s.transport.EXPECT().
			PublishPublicMessage(roomMatcher, stateMatcher).
			Times(1)

		err = s.dealer.PublishVote(dealerVote)
		s.Require().NoError(err)

		state = stateMatcher.Wait()
		item := checkIssues(state.Issues)
		s.Require().NotNil(item)
		s.Require().Nil(item.Result)
		s.Require().Len(item.Votes, 1)

		vote, ok := item.Votes[s.dealer.Player().ID]
		s.Require().True(ok)
		s.Require().Empty(vote.Value)
		s.Require().Greater(vote.Timestamp, int64(0))
	}

	{ // Reveal votes
		stateMatcher = s.newStateMatcher()
		s.transport.EXPECT().
			PublishPublicMessage(roomMatcher, stateMatcher).
			Times(1)

		err = s.dealer.Reveal()
		s.Require().NoError(err)

		state = stateMatcher.Wait()
		item := checkIssues(state.Issues)
		s.Require().Nil(item.Result)
		s.Require().Len(item.Votes, 1)

		vote, ok := item.Votes[s.dealer.Player().ID]
		s.Require().True(ok)
		s.Require().NotNil(vote)
		s.Require().Equal(dealerVote, vote.Value)
		s.Require().Greater(vote.Timestamp, int64(0))
	}

	const votingResult = protocol.VoteValue("1")

	{ // Finish voting
		stateMatcher = s.newStateMatcher()
		s.transport.EXPECT().
			PublishPublicMessage(roomMatcher, stateMatcher).
			Times(1)

		err = s.dealer.Finish(votingResult)
		s.Require().NoError(err)

		state = stateMatcher.Wait()
		item := checkIssues(state.Issues)
		s.Require().NotNil(item.Result)
		s.Require().Equal(votingResult, *item.Result)
		s.Require().Len(item.Votes, 1)

		vote, ok := item.Votes[s.dealer.Player().ID]
		s.Require().True(ok)
		s.Require().Equal(dealerVote, vote.Value)
		s.Require().Greater(vote.Timestamp, int64(0))
	}

	const secondItemText = "b"
	var secondIssueID protocol.IssueID

	checkIssues = func(issues protocol.IssuesList) *protocol.Issue {
		s.Require().Len(issues, 2)

		item := issues.Get(firstIssueID)
		s.Require().NotNil(item)
		s.Require().Equal(firstItemText, item.TitleOrURL)

		item = issues.Get(secondIssueID)
		s.Require().NotNil(item)
		s.Require().Equal(secondItemText, item.TitleOrURL)

		return item
	}

	{ // Deal another issue
		stateMatcher = s.newStateMatcher()
		s.transport.EXPECT().
			PublishPublicMessage(roomMatcher, stateMatcher).
			Times(1)

		secondIssueID, err = s.dealer.Deal(secondItemText)
		s.Require().NoError(err)

		state = stateMatcher.Wait()
		item := checkIssues(state.Issues)
		s.Require().Nil(item.Result)
		s.Require().Empty(item.Votes)
	}
}

func (s *Suite) TestPublishMessageWithNoRoom() {
	game := s.newGame(nil)
	err := game.publishMessage(nil)
	s.Require().ErrorIs(err, ErrNoRoom)
}

func (s *Suite) TestPublishUnsupportedMessage() {
	var err error

	game := s.newGame(nil)
	game.room, err = protocol.NewRoom()
	s.Require().NoError(err)

	err = game.publishMessage(make(chan int))
	s.Require().Error(err)
}

func (s *Suite) TestPublishMessage() {
	testCases := []struct {
		name       string
		encryption bool
	}{
		{
			name:       "encryption message",
			encryption: true,
		},
		{
			name:       "unencrypted message",
			encryption: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Create controller inside subtest
			ctrl := gomock.NewController(s.T())
			s.transport = mocktransport.NewMockService(ctrl)

			game := s.newGame([]Option{
				WithEnableSymmetricEncryption(tc.encryption),
			})

			var err error
			game.room, err = protocol.NewRoom()
			s.Require().NoError(err)

			roomMatcher := matchers.NewRoomMatcher(game.room)
			payload, jsonPayload := s.FakePayload()

			if tc.encryption {
				s.transport.EXPECT().
					PublishPublicMessage(roomMatcher, gomock.Eq(jsonPayload)).
					Times(1)
			} else {
				s.transport.EXPECT().
					PublishUnencryptedMessage(roomMatcher, gomock.Eq(jsonPayload)).
					Times(1)
			}

			err = game.publishMessage(payload)
			s.Require().NoError(err)
		})
	}
}

func (s *Suite) TestOnlineState() {
	/*
		2. Create and join room
		3. mock time = 0
		4. Alice sends online message
		5. Dealer updates online timestamp
		6. mock time = 20
		7. Dealer checks online initialState, mark as offline
	*/

	playerID, err := GeneratePlayerID()
	s.Require().NoError(err)

	player := protocol.Player{
		ID:   playerID,
		Name: gofakeit.Username(),
	}

	s.dealer = s.newGame([]Option{
		WithPlayerName("dealer"),
		WithEnablePublishOnlineState(false), // FIXME: Add a separate test for self publishing
	})

	s.Logger.Debug("<<< test info",
		zap.Any("player", player),
		zap.Any("dealer", s.dealer.Player()),
	)

	room, initialState, err := s.dealer.CreateNewRoom()
	s.Require().NoError(err)
	s.Require().NotNil(room)

	roomID := room.ToRoomID()
	roomMatcher := matchers.NewRoomMatcher(room)

	s.expectSubscribeToMessages(room)

	//s.transport.EXPECT().
	//	PublishPublicMessage(roomMatcher,
	//		matchers.NewOnlineMatcher(s.T(), s.dealer.Player().ID)).
	//	AnyTimes()

	stateMatcher := s.newStateMatcher()
	s.transport.EXPECT().
		PublishPublicMessage(roomMatcher, stateMatcher).
		Times(1)

	err = s.dealer.JoinRoom(roomID, initialState)
	s.Require().NoError(err)

	_ = stateMatcher.Wait()

	// Player joins the room
	playerOnlineMessage, err := json.Marshal(&protocol.PlayerOnlineMessage{
		Message: protocol.Message{
			Type:      protocol.MessageTypePlayerOnline,
			Timestamp: s.clock.Now().UnixMilli(),
		},
		Player: player,
	})
	s.Require().NoError(err)

	stateMatcher = s.newStateMatcher()
	s.transport.EXPECT().
		PublishPublicMessage(roomMatcher, stateMatcher).
		Times(1)

	s.dealer.handlePlayerOnlineMessage(playerOnlineMessage)

	// Ensure new player joined
	state := stateMatcher.Wait()
	s.Require().Len(state.Players, 2)

	p, ok := state.Players.Get(player.ID)
	s.Require().True(ok)
	s.Require().True(p.Online)
	s.Require().Equal(s.clock.Now().UnixMilli(), p.OnlineTimestampMilliseconds)

	s.transport.EXPECT().
		PublishPublicMessage(roomMatcher, stateMatcher).
		Times(1)

	// Advance time, make sure player is marked as offline
	lastSeenAt := p.OnlineTimestampMilliseconds
	s.clock.Advance(playerOnlineTimeout)

	state = stateMatcher.Wait()
	s.Require().Len(state.Players, 2)

	p, ok = state.Players.Get(player.ID)
	s.Require().True(ok)
	s.Require().False(p.Online)
	s.Require().Equal(lastSeenAt, p.OnlineTimestampMilliseconds)
}
