package view

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"go.uber.org/zap"

	"github.com/six78/2-story-points-cli/internal/config"
	"github.com/six78/2-story-points-cli/internal/view/messages"
	"github.com/six78/2-story-points-cli/internal/view/states"
)

// Any command here must:
// 	1. Get App as argument
// 	2. Return tea.Cmd

// FIXME: Move to ./commands/

func ProcessUserInput(m *model) tea.Cmd {
	defer m.input.Reset()
	return ProcessInput(m)
}

func ProcessInput(m *model) tea.Cmd {
	if m.state == states.InputPlayerName {
		defer m.input.Reset()
		return processPlayerNameInput(m, m.input.Value())
	}

	if m.state == states.Playing {
		defer m.input.Reset()
		return ProcessAction(m, m.input.Value())
	}

	return nil
}

func ProcessAction(m *model, action string) tea.Cmd {
	defer func() {
		config.Logger.Debug("user action processed",
			zap.Any("state", m.state),
		)
	}()

	args := strings.Fields(action)
	if len(args) == 0 {
		return nil
	}

	commandRoot := Action(args[0])
	commandFn, ok := actions[commandRoot]

	if !ok {
		return func() tea.Msg {
			err := fmt.Errorf("unknown action: %s", commandRoot)
			return messages.NewErrorMessage(err)
		}
	}

	return commandFn(m, args[1:])
}
