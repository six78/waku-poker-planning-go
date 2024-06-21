package voteview

import (
	"strconv"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/six78/2-story-points-cli/pkg/protocol"
)

var (
	CommonVoteStyle = lipgloss.NewStyle().
			PaddingLeft(1).
			PaddingRight(1).
			Align(lipgloss.Center)
	NoVoteStyle     = CommonVoteStyle.Copy().Foreground(lipgloss.Color("#444444"))
	ReadyVoteStyle  = CommonVoteStyle.Copy().Foreground(lipgloss.Color("#5fd700"))
	LightVoteStyle  = CommonVoteStyle.Copy().Foreground(lipgloss.Color("#00d7ff"))
	MediumVoteStyle = CommonVoteStyle.Copy().Foreground(lipgloss.Color("#ffd787"))
	DangerVoteStyle = CommonVoteStyle.Copy().Foreground(lipgloss.Color("#ff005f"))
)

type VoteValueState int

const (
	voteValueDefault VoteValueState = iota
	voteValueEmpty
	voteValueInProgress
	voteValueX
	voteValueHidden
)

type Model struct {
	value protocol.VoteValue
	style *lipgloss.Style
	state VoteValueState

	applyStyle bool
	spinner    spinner.Model
}

func New(applyStyle bool) Model {
	s := spinner.New()
	s.Spinner = spinner.MiniDot

	return Model{
		style:      &CommonVoteStyle,
		state:      voteValueDefault,
		applyStyle: applyStyle,
		spinner:    s,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
	)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var spinnerCommand tea.Cmd
	if m.state == voteValueInProgress {
		m.spinner, spinnerCommand = m.spinner.Update(msg)
	}
	return m, spinnerCommand
}

func (m Model) View() string {
	var view string

	switch m.state {
	case voteValueDefault:
		view = string(m.value)
	case voteValueEmpty:
		view = ""
	case voteValueInProgress:
		view = m.spinner.View()
	case voteValueX:
		view = "X"
	case voteValueHidden:
		view = "✓"
	}

	if !m.applyStyle {
		return view
	}

	return m.style.Render(view)
}

func (m Model) Style() lipgloss.Style {

	return *m.style
}

func (m *Model) SetValue(value protocol.VoteValue) {
	m.value = value
}

func (m *Model) Reset() {
	m.state = voteValueDefault
	m.value = ""
	m.style = &CommonVoteStyle
}

func (m *Model) Show() {
	m.state = voteValueDefault
	m.style = VoteStyle(m.value)
}

func (m *Model) Hide() {
	m.state = voteValueHidden
	m.style = &ReadyVoteStyle
}

func (m *Model) Spin() {
	m.state = voteValueInProgress
}

func (m *Model) Cross() {
	m.state = voteValueX
	m.style = &NoVoteStyle
}

func (m *Model) Clear() {
	m.value = ""
	m.state = voteValueEmpty
	m.style = &CommonVoteStyle
}

func VoteStyle(vote protocol.VoteValue) *lipgloss.Style {
	voteNumber, err := strconv.Atoi(string(vote))
	if err != nil {
		return &CommonVoteStyle
	}
	if voteNumber >= 13 {
		return &DangerVoteStyle
	}
	if voteNumber >= 5 {
		return &MediumVoteStyle
	}
	return &LightVoteStyle
}
