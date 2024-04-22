package editor

import (
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbletea"
)

var Content string
var Quit bool

type errMsg error

type Model struct {
	textarea textarea.Model
	err      error
}

func InitialModel() Model {
	ti := textarea.New()
	ti.SetHeight(10)
	ti.SetWidth(45)
	ti.Placeholder = "Enter your markdown comment here..."
	ti.Focus()
	return Model{
		textarea: ti,
		err:      nil,
	}
}

func (m Model) Init() tea.Cmd {
	return textarea.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if m.textarea.Height() > msg.Height {
			m.textarea.SetWidth(int(float32(msg.Width) * .95))
		}
		if m.textarea.Width() > msg.Width {
			m.textarea.SetWidth(int(float32(msg.Height) * .95))
		}
	case tea.KeyMsg:
		switch msg.Type.String() {
		case "enter":
			if !m.textarea.Focused() {
				Content = m.textarea.Value()
				return m, tea.Quit
			}
		case "esc":
			if m.textarea.Focused() {
				m.textarea.Blur()
			}
		case "ctrl+c":
			Quit = true
			return m, tea.Quit
		default:
			if !m.textarea.Focused() {
				cmd = m.textarea.Focus()
				cmds = append(cmds, cmd)
			}
		}
	case errMsg:
		m.err = msg
		return m, nil
	}
	m.textarea, cmd = m.textarea.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	return m.textarea.View()
}
