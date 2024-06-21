package userinput

import (
	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/six78/2-story-points-cli/internal/config"
	"github.com/six78/2-story-points-cli/internal/view/messages"
	"github.com/six78/2-story-points-cli/internal/view/states"
)

var style = lipgloss.NewStyle().Foreground(config.UserColor)

type Model struct {
	input       textinput.Model
	commandMode bool
}

func New(commandMode bool) Model {
	input := textinput.New()
	input.Placeholder = "Type a command..."
	input.Prompt = "┃ "
	input.Cursor.SetMode(cursor.CursorBlink)
	input.Cursor.Style = style
	if commandMode {
		input.Focus()
	}

	return Model{
		input:       input,
		commandMode: commandMode,
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case messages.AppStateMessage:
		switch msg.State {
		case states.Playing:
			m.input.Placeholder = "Type a command..."
		case states.InputPlayerName:
			cmd = m.input.Focus()
			cmds = append(cmds, cmd)
			m.input.Placeholder = "Type your name..."
		default:
		}
	case messages.AppStateFinishedMessage:
		switch msg.State {
		case states.InputPlayerName:
			if !m.commandMode {
				m.input.Blur()
			}
		default:
		}
	case messages.CommandModeChange:
		m.commandMode = msg.CommandMode
		if m.commandMode {
			cmd = m.input.Focus()
			cmds = append(cmds, cmd)
		} else {
			m.input.Blur()
		}
	}

	m.input, cmd = m.input.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	return m.input.View()
}

func (m *Model) SetValue(s string) {
	m.input.SetValue(s)
}

func (m *Model) Focused() bool {
	return m.input.Focused()
}

func (m *Model) Reset() {
	m.input.Reset()
}

func (m *Model) Value() string {
	return m.input.Value()
}
