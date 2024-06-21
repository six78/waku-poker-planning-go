package wakustatusview

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/six78/2-story-points-cli/internal/transport"
	"github.com/six78/2-story-points-cli/internal/view/messages"
)

var (
	okStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#00E676"))
	warnStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFEA00"))
	dangerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5722"))
)

type Model struct {
	status transport.ConnectionStatus
}

func New() Model {
	return Model{}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) Model {
	switch msg := msg.(type) {
	case messages.ConnectionStatus:
		m.status = msg.Status
	}
	return m
}

func (m Model) View() string {
	marker := "●"
	if m.status.PeersCount > 3 {
		marker = okStyle.Render(marker)
	} else if m.status.PeersCount > 0 {
		marker = warnStyle.Render(marker)
	} else {
		marker = dangerStyle.Render(marker)
	}

	text := fmt.Sprintf(" Waku connection status: %d peer(s)", m.status.PeersCount)

	return lipgloss.JoinHorizontal(lipgloss.Left, marker, text)
}
