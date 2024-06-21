package commands

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pkg/errors"

	"github.com/six78/2-story-points-cli/internal/transport"
	"github.com/six78/2-story-points-cli/internal/view/messages"
	"github.com/six78/2-story-points-cli/internal/view/states"
	"github.com/six78/2-story-points-cli/pkg/game"
	"github.com/six78/2-story-points-cli/pkg/protocol"
)

func InitializeApp(game *game.Game, transport transport.Service) tea.Cmd {
	return func() tea.Msg {
		err := transport.Initialize()
		if err != nil {
			return messages.FatalErrorMessage{
				Err: errors.Wrap(err, "failed to initialize transport"),
			}
		}

		err = transport.Start()
		if err != nil {
			return messages.FatalErrorMessage{
				Err: errors.Wrap(err, "failed to start transport"),
			}
		}

		err = game.Initialize()
		if err != nil {
			return messages.FatalErrorMessage{
				Err: errors.Wrap(err, "failed to initialize game"),
			}
		}

		return messages.AppStateFinishedMessage{State: states.Initializing}
	}
}

func CreateNewRoom(game *game.Game) tea.Cmd {
	return func() tea.Msg {
		room, initialState, err := game.CreateNewRoom()
		if err != nil {
			return messages.NewErrorMessage(err)
		}

		roomID := room.ToRoomID()

		err = game.JoinRoom(roomID, initialState)
		if err != nil {
			return messages.NewErrorMessage(err)
		}

		return messages.RoomJoin{
			RoomID:   game.RoomID(),
			IsDealer: game.IsDealer(),
		}
	}
}

func JoinRoom(game *game.Game, roomID protocol.RoomID) tea.Cmd {
	return func() tea.Msg {
		err := game.JoinRoom(roomID, nil)
		if err != nil {
			return messages.NewErrorMessage(err)
		}
		return messages.RoomJoin{
			RoomID:   game.RoomID(),
			IsDealer: game.IsDealer(),
		}
	}
}

func ToggleRoomView(currentRoomView states.RoomView) tea.Cmd {
	return func() tea.Msg {
		var nextRoomView states.RoomView
		switch currentRoomView {
		case states.ActiveIssueView:
			nextRoomView = states.IssuesListView
		case states.IssuesListView:
			nextRoomView = states.ActiveIssueView
		}
		return messages.RoomViewChange{RoomView: nextRoomView}
	}
}

func PublishVote(game *game.Game, vote protocol.VoteValue) tea.Cmd {
	return func() tea.Msg {
		err := game.PublishVote(vote)
		if err != nil {
			return messages.NewErrorMessage(err)
		}
		// TODO: Send err=nil ErrorMessage here
		return messages.MyVote{
			Result: game.MyVote(),
		}
	}
}

func FinishVoting(game *game.Game, result protocol.VoteValue) tea.Cmd {
	return func() tea.Msg {
		err := game.Finish(result)
		return messages.NewErrorMessage(err)
	}
}

func AddIssue(game *game.Game, urlOrTitle string) tea.Cmd {
	return func() tea.Msg {
		_, err := game.AddIssue(urlOrTitle)
		return messages.NewErrorMessage(err)
	}
}

func SelectIssue(game *game.Game, index int) tea.Cmd {
	return func() tea.Msg {
		err := game.SelectIssue(index)
		return messages.NewErrorMessage(err)
	}
}

func QuitApp(game *game.Game) tea.Cmd {
	return func() tea.Msg {
		if game != nil {
			game.LeaveRoom()
		}
		return tea.Quit()
	}
}

func DelayMessage(timeout time.Duration, msg tea.Msg, restart chan struct{}) tea.Cmd {
	return func() tea.Msg {
		for {
			select {
			case <-time.After(timeout):
				return msg
			case <-restart:
			}
		}
	}
}
