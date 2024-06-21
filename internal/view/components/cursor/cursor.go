package cursor

import (
	"math"

	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	vertical bool
	focused  bool
	position int
	min      int
	max      int
}

func New(vertical bool, focused bool) Model {
	return Model{
		vertical: vertical,
		focused:  focused,
		position: 0,
		min:      0,
		max:      0,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) Model {
	if !m.focused {
		return m
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyLeft:
			if !m.vertical {
				m.decrementCursor(m.position)
			}
		case tea.KeyRight:
			if !m.vertical {
				m.incrementCursor(m.position)
			}
		case tea.KeyUp:
			if m.vertical {
				m.decrementCursor(m.position)
			}
		case tea.KeyDown:
			if m.vertical {
				m.incrementCursor(m.position)
			}
		}
	}
	return m
}

func (m *Model) Match(position int) bool {
	return m.focused && m.position == position
}

func (m *Model) SetPosition(position int) {
	m.position = position
	m.adjustPosition()
}

func (m *Model) Vertical() bool {
	return m.vertical
}

func (m *Model) Position() int {
	return m.position
}

func (m *Model) Focused() bool {
	return m.focused
}

func (m *Model) SetFocus(focused bool) {
	m.focused = focused
}

func (m *Model) SetRange(min int, max int) {
	m.min = min
	m.max = max
	m.adjuistRange()
	m.adjustPosition()
}

func (m *Model) adjuistRange() {
	if m.max < m.min {
		m.max = m.min
	}
}

func (m *Model) adjustPosition() {
	if m.position > m.max {
		m.position = m.max
	}
	if m.position < m.min {
		m.position = m.min
	}
}

func (m *Model) Min() int {
	return m.min
}

func (m *Model) Max() int {
	return m.max

}

func (m *Model) decrementCursor(cursor int) {
	newPosition := math.Max(float64(cursor-1), float64(m.min))
	m.position = int(newPosition)
}

func (m *Model) incrementCursor(cursor int) {
	newPosition := math.Min(float64(cursor+1), float64(m.max))
	m.position = int(newPosition)
}
