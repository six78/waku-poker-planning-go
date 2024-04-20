package view

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pkg/errors"
	"golang.org/x/exp/slices"
	"strconv"
	"strings"
	"waku-poker-planning/game"
	"waku-poker-planning/protocol"
	"waku-poker-planning/view/commands"
	"waku-poker-planning/view/messages"
	"waku-poker-planning/view/states"
)

type Action string

const (
	Rename Action = "rename"
	New    Action = "new"
	Join   Action = "join"
	Vote   Action = "vote"
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
	Deal:   runDealAction,
	Add:    runAddAction,
	New:    runNewAction,
	Join:   runJoinAction,
	Reveal: runRevealAction,
	Finish: runFinishAction,
	Deck:   runDeckAction,
	Select: runSelectAction,
}

func processPlayerNameInput(m *model, playerName string) tea.Cmd {
	return func() tea.Msg {
		m.app.Game.RenamePlayer(playerName)
		return messages.AppStateFinishedMessage{State: states.InputPlayerName}
	}
}

func runRenameAction(m *model, args []string) tea.Cmd {
	return func() tea.Msg {
		if len(args) == 0 {
			err := errors.New("empty user")
			return messages.NewErrorMessage(err)
		}
		m.app.Game.RenamePlayer(args[0])
		return messages.NewErrorMessage(nil)
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

		err = m.app.Game.PublishVote(vote)
		return messages.NewErrorMessage(err)
	}
}

func runDealAction(m *model, args []string) tea.Cmd {
	return func() tea.Msg {
		if len(args) == 0 {
			err := errors.New("empty deal")
			return messages.NewErrorMessage(err)
		}
		_, err := m.app.Game.Deal(args[0])
		return messages.NewErrorMessage(err)
	}
}

func runAddAction(m *model, args []string) tea.Cmd {
	return func() tea.Msg {
		if len(args) == 0 {
			err := errors.New("empty issue")
			return messages.NewErrorMessage(err)
		}
		_, err := m.app.Game.AddIssue(args[0])
		return messages.NewErrorMessage(err)
	}
}

func runNewAction(m *model, args []string) tea.Cmd {
	m.state = states.CreatingRoom
	return commands.CreateNewRoom(m.app)
}

func runJoinAction(m *model, args []string) tea.Cmd {
	m.state = states.JoiningRoom
	return commands.JoinRoom(args[0], m.app)
}

func runRevealAction(m *model, args []string) tea.Cmd {
	return func() tea.Msg {
		err := m.app.Game.Reveal()
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
		err = m.app.Game.Finish(result)
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

		err = m.app.Game.SetDeck(deck)
		return messages.NewErrorMessage(err)
	}
}

func runSelectAction(m *model, args []string) tea.Cmd {
	return func() tea.Msg {
		if len(args) == 0 {
			return messages.NewErrorMessage(errors.New("no issue index given"))
		}

		index, err := strconv.Atoi(args[0])
		if err != nil {
			return messages.NewErrorMessage(fmt.Errorf("invalid issue index: %s", args[0]))
		}

		err = m.app.Game.SelectIssue(index)
		return messages.NewErrorMessage(err)
	}
}
