package protocol

import "waku-poker-planning/config"

const Version byte = 1

type PlayerID string
type IssueID string

type Player struct {
	ID   PlayerID `json:"id"`
	Name string   `json:"name"`
}

type Issue struct {
	ID         IssueID                 `json:"id"`
	TitleOrURL string                  `json:"titleOrUrl"`
	Votes      map[PlayerID]VoteResult `json:"votes"`
	Result     *VoteValue              `json:"result"` // NOTE: keep pointer. Because "empty string means vote is not revealed"
}

type VoteState string

const (
	IdleState     VoteState = "idle"     // ActiveIssue == ""
	VotingState   VoteState = "voting"   // ActiveIssue != "", Revealed == false
	RevealedState VoteState = "revealed" // ActiveIssue != "", Revealed == true, Issues[ActiveIssue].Result == nil
	FinishedState VoteState = "finished" // ActiveIssue != "", Revealed == true, Issues[ActiveIssue].Result != nil
)

type State struct {
	Players     []Player   `json:"players"`
	Issues      IssuesList `json:"issues"`
	ActiveIssue IssueID    `json:"activeIssue"`
	Deck        Deck       `json:"-"`

	//VoteState VoteState `json:"voteState"` // FIXME: bool revealResults
	VotesRevealed bool `json:"votesRevealed"`
}

func (s *State) VoteState() VoteState {
	if s.ActiveIssue == "" {
		return IdleState
	}
	if !s.VotesRevealed {
		return VotingState
	}
	issue := s.Issues.Get(s.ActiveIssue)
	if issue == nil {
		config.Logger.Error("active issue not found when calculating vote state")
		return IdleState
	}
	if issue.Result == nil {
		return RevealedState
	}
	return FinishedState
}

type MessageType string

const (
	MessageTypeState        MessageType = "__state"
	MessageTypePlayerOnline MessageType = "__player_online"
	MessageTypePlayerVote   MessageType = "__player_vote"
)

type Message struct {
	Type      MessageType `json:"type"`
	Timestamp int64       `json:"updatedAt"` // WARNING: rename to Timestamp
}

type GameStateMessage struct {
	Message
	State State `json:"state"`
}

type PlayerOnlineMessage struct {
	Message
	Player Player `json:"player,omitempty"`
}

type PlayerVoteMessage struct {
	Message
	PlayerID   PlayerID   `json:"playerId"`
	Issue      IssueID    `json:"issue"`
	VoteResult VoteResult `json:"vote"`
}

type Deck []VoteValue
