package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mohammadalipour/gonsole/internal/editor"
)

func main() {
	prog := tea.NewProgram(editor.New())
	if err := prog.Start(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
