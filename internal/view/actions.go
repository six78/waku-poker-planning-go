package view

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pkg/errors"
	"golang.org/x/exp/slices"

	"github.com/six78/2-story-points-cli/internal/view/commands"
	"github.com/six78/2-story-points-cli/internal/view/messages"
	"github.com/six78/2-story-points-cli/internal/view/states"
	"github.com/six78/2-story-points-cli/pkg/game"
	"github.com/six78/2-story-points-cli/pkg/protocol"
)

type Action string

const (
	Rename Action = "rename"
	New    Action = "new"
	Join   Action = "join"
	Exit   Action = "exit"
	Vote   Action = "vote"
	Unvote Action = "unvote"
	Deal   Action = "deal"
	Add    Action = "add"
	Reveal Action = "reveal"
	Finish Action = "finish"
	Deck   Action = "deck"
	Select Action = "select"
)

type actionFunc func(m *model, args []string) tea.Cmd

var actions = map[Action]actionFunc{
	Rename: runRenameAction,
	Vote:   runVoteAction,
	Unvote: runUnvoteAction,
	Deal:   runDealAction,
	Add:    runAddAction,
	New:    runNewAction,
	Join:   runJoinAction,
	Exit:   runExitAction,
	Reveal: runRevealAction,
	Finish: runFinishAction,
	Deck:   runDeckAction,
	Select: runSelectAction,
}

func processPlayerNameInput(m *model, playerName string) tea.Cmd {
	return func() tea.Msg {
		err := m.game.RenamePlayer(playerName)
		if err != nil {
			return messages.NewErrorMessage(err)
		}
		return messages.AppStateFinishedMessage{
			State: states.InputPlayerName,
		}
	}
}

func runRenameAction(m *model, args []string) tea.Cmd {
	return func() tea.Msg {
		if len(args) == 0 {
			err := errors.New("empty user")
			return messages.NewErrorMessage(err)
		}
		err := m.game.RenamePlayer(args[0])
		return messages.NewErrorMessage(err)
	}
}

func parseVote(input string) (protocol.VoteValue, error) {
	return protocol.VoteValue(input), nil
}

func runVoteAction(m *model, args []string) tea.Cmd {
	return func() tea.Msg {
		if len(args) == 0 {
			err := errors.New("empty vote")
			return messages.NewErrorMessage(err)
		}

		vote, err := parseVote(args[0])
		if err != nil {
			err = errors.Wrap(err, "failed to parse vote")
			return messages.NewErrorMessage(err)
		}

		return commands.PublishVote(m.game, vote)()
	}
}

func runUnvoteAction(m *model, args []string) tea.Cmd {
	return func() tea.Msg {
		return commands.PublishVote(m.game, "")()
	}
}

func runDealAction(m *model, args []string) tea.Cmd {
	return func() tea.Msg {
		if len(args) == 0 {
			err := errors.New("empty deal")
			return messages.NewErrorMessage(err)
		}
		// TODO: Find a better way of restoring empty spaces between args
		issue := strings.Join(args, " ")
		_, err := m.game.Deal(issue)
		return messages.NewErrorMessage(err)
	}
}

func runAddAction(m *model, args []string) tea.Cmd {
	return func() tea.Msg {
		if len(args) == 0 {
			err := errors.New("empty issue")
			return messages.NewErrorMessage(err)
		}
		return commands.AddIssue(m.game, args[0])()
	}
}

func runNewAction(m *model, args []string) tea.Cmd {
	return commands.CreateNewRoom(m.game)
}

func runJoinAction(m *model, args []string) tea.Cmd {
	return func() tea.Msg {
		if len(args) == 0 {
			err := errors.New("no room id argument provided")
			return messages.NewErrorMessage(err)
		}
		roomID := protocol.NewRoomID(args[0])
		return commands.JoinRoom(m.game, roomID)()
	}
}

func runExitAction(m *model, args []string) tea.Cmd {
	return func() tea.Msg {
		m.game.LeaveRoom()
		return messages.RoomJoin{
			RoomID:   m.game.RoomID(),
			IsDealer: m.game.IsDealer(),
		}
	}
}

func runRevealAction(m *model, args []string) tea.Cmd {
	return func() tea.Msg {
		err := m.game.Reveal()
		return messages.NewErrorMessage(err)
	}
}

func runFinishAction(m *model, args []string) tea.Cmd {
	return func() tea.Msg {
		if len(args) == 0 {
			err := errors.New("empty result")
			return messages.NewErrorMessage(err)
		}
		result, err := parseVote(args[0])
		if err != nil {
			return messages.NewErrorMessage(err)
		}
		if !slices.Contains(m.gameState.Deck, result) {
			err = errors.New("result not in deck")
			return messages.NewErrorMessage(err)
		}
		err = m.game.Finish(result)
		return messages.NewErrorMessage(err)
	}
}

func parseDeck(args []string) (protocol.Deck, error) {
	if len(args) == 0 {
		return nil, errors.New("deck can't be empty")
	}

	if len(args) == 1 {
		// attempt to parse deck by name
		deckName := strings.ToLower(args[0])
		deck, ok := game.GetDeck(deckName)
		if !ok {
			return nil, fmt.Errorf("unknown deck: '%s', available decks: %s",
				args[0], strings.Join(game.AvailableDecks(), ", "))
		}
		return deck, nil
	}

	deck := protocol.Deck{}
	cards := map[string]struct{}{}

	for _, card := range args {
		if _, ok := cards[card]; ok {
			return nil, fmt.Errorf("duplicate card: '%s'", card)
		}
		cards[card] = struct{}{}
		deck = append(deck, protocol.VoteValue(card))
	}

	return deck, nil
}

func runDeckAction(m *model, args []string) tea.Cmd {
	return func() tea.Msg {
		deck, err := parseDeck(args)
		if err != nil {
			return messages.NewErrorMessage(err)
		}

		err = m.game.SetDeck(deck)
		return messages.NewErrorMessage(err)
	}
}

func runSelectAction(m *model, args []string) tea.Cmd {
	return func() tea.Msg {
		if len(args) == 0 {
			err := errors.New("no issue index provided")
			return messages.NewErrorMessage(err)
		}

		index, err := strconv.Atoi(args[0])
		if err != nil {
			err = fmt.Errorf("invalid issue index: %s (%w)", args[0], err)
			return messages.NewErrorMessage(err)
		}

		return commands.SelectIssue(m.game, index)()
	}
}
