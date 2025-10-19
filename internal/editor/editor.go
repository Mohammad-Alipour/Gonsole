package editor

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mohammadalipour/gonsole/internal/syntax"
)

type Model struct {
	cursorX int
	cursorY int
	lines   []string
}

func New() Model {
	return Model{
		cursorX: 0,
		cursorY: 0,
		lines:   []string{""},
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			m.lines = append(m.lines[:m.cursorY+1],
				append([]string{""}, m.lines[m.cursorY+1:]...)...)
			m.cursorY++
			m.cursorX = 0
		case "backspace":
			if m.cursorX > 0 {
				line := m.lines[m.cursorY]
				m.lines[m.cursorY] = line[:m.cursorX-1] + line[m.cursorX:]
				m.cursorX--
			}
		case "left":
			if m.cursorX > 0 {
				m.cursorX--
			}
		case "right":
			if m.cursorX < len(m.lines[m.cursorY]) {
				m.cursorX++
			}
		case "up":
			if m.cursorY > 0 {
				m.cursorY--
				if m.cursorX > len(m.lines[m.cursorY]) {
					m.cursorX = len(m.lines[m.cursorY])
				}
			}
		case "down":
			if m.cursorY < len(m.lines)-1 {
				m.cursorY++
				if m.cursorX > len(m.lines[m.cursorY]) {
					m.cursorX = len(m.lines[m.cursorY])
				}
			}
		default:
			m.insertRune(msg.String())
		}
	}
	return m, nil
}

func (m *Model) insertRune(s string) {
	if len(s) == 1 {
		line := m.lines[m.cursorY]
		m.lines[m.cursorY] = line[:m.cursorX] + s + line[m.cursorX:]
		m.cursorX++
	}
}

func (m Model) View() string {
	code := ""
	for _, line := range m.lines {
		code += line + "\n"
	}

	colored, err := syntax.Highlight(code, "go")
	if err != nil {
		colored = code
	}

	return fmt.Sprintf("%s\n\n[ESC to exit]", colored)
}
