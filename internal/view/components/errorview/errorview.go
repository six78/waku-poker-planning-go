package errorview

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/six78/2-story-points-cli/internal/view/messages"
)

const color = lipgloss.Color("#d78700")

type Model struct {
	errorMessage string
	style        lipgloss.Style
}

func New() Model {
	return Model{
		errorMessage: "",
		style:        lipgloss.NewStyle().Foreground(color),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) Model {
	switch msg := msg.(type) {
	case messages.ErrorMessage:
		if msg.Err == nil {
			m.errorMessage = ""
		} else {
			m.errorMessage = msg.Err.Error()
		}
	}
	return m
}

func (m Model) View() string {
	return m.style.Render(m.errorMessage)
}
