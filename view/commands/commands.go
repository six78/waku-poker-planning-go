package commands

import (
	"errors"
	tea "github.com/charmbracelet/bubbletea"
	"waku-poker-planning/app"
	"waku-poker-planning/view/messages"
	"waku-poker-planning/view/states"
)

// Any command here must:
// 	1. Get App as argument
// 	2. Return tea.Cmd

func InitializeApp(a *app.App) tea.Cmd {
	return func() tea.Msg {
		err := a.Initialize()
		if err != nil {
			return messages.FatalErrorMessage{Err: err}
		}
		return messages.AppStateMessage{FinishedState: states.Initializing}
	}
}

func WaitForWakuPeers(a *app.App) tea.Cmd {
	return func() tea.Msg {
		ok := a.WaitForPeersConnected()
		if !ok {
			return messages.FatalErrorMessage{
				Err: errors.New("failed to connect to peers"),
			}
		}
		return messages.AppStateMessage{FinishedState: states.WaitingForPeers}
	}
}

func CreateNewRoom(a *app.App) tea.Cmd {
	return func() tea.Msg {
		err := a.Game.CreateNewRoom()
		return messages.AppStateMessage{
			FinishedState: states.CreatingRoom,
			ErrorMessage: messages.ErrorMessage{
				Err: err,
			},
		}
	}
}

func JoinRoom(roomID string, a *app.App) tea.Cmd {
	return func() tea.Msg {
		err := a.Game.JoinRoom(roomID)
		return messages.AppStateMessage{
			FinishedState: states.JoiningRoom,
			ErrorMessage: messages.ErrorMessage{
				Err: err,
			},
		}
	}
}

func WaitForGameState(app *app.App) tea.Cmd {
	return func() tea.Msg {
		state, more, err := app.WaitForGameState()
		if err != nil {
			return messages.FatalErrorMessage{err}
		}
		if !more {
			return nil
		}
		return messages.GameStateMessage{State: state}
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
