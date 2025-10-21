package main

import (
	"log"

	"github.com/Mohammad-Alipour/Gonsole/internal/editor"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	p := tea.NewProgram(editor.New(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

//main
