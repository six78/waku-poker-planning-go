package view

import (
	tea "github.com/charmbracelet/bubbletea"
	"go.uber.org/zap"

	"github.com/six78/2-story-points-cli/internal/config"
	"github.com/six78/2-story-points-cli/internal/transport"
	"github.com/six78/2-story-points-cli/pkg/game"
)

func Run(game *game.Game, transport transport.Service) int {
	m := initialModel(game, transport)
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		config.Logger.Error("error running program", zap.Error(err))
		return 1
	}
	return 0
}
