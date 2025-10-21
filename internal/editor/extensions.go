package editor

import (
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	extTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00BFFF")).
			Padding(1, 2)

	extItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#C0C0C0")).
			PaddingLeft(4)

	extActiveItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#0078D4")).
				Bold(true).
				PaddingLeft(4)

	extStatusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF88")).
			Padding(1, 2)

	extErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5555")).
			Padding(1, 2)

	searchInputStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#1e1e1e")).
				Foreground(lipgloss.Color("#ffffff")).
				Padding(0, 1).
				Margin(0, 1)
)

type Extension struct {
	Name        string
	Command     string
	Description string
	Installed   bool
}

type ExtensionsModel struct {
	items        []Extension
	selectedIdx  int   // index into filteredIdxs
	filteredIdxs []int // maps displayed rows -> items index
	status       string
	width        int
	height       int

	// search
	searchActive bool
	searchQuery  string
}

type InstallResultMsg struct {
	Index   int
	Success bool
	Output  string
}

func NewExtensionsModel() ExtensionsModel {
	exts := []Extension{
		{
			Name:        "Go Tools (gopls)",
			Command:     "gopls",
			Description: "Language Server for Go — autocompletion, diagnostics, navigation.",
		},
		{
			Name:        "Python (pyright)",
			Command:     "pyright",
			Description: "Type checking and language features for Python.",
		},
		{
			Name:        "TypeScript (typescript-language-server)",
			Command:     "typescript-language-server",
			Description: "Provides JS/TS autocompletion and hover info.",
		},
		{
			Name:        "HTML / CSS (vscode-langservers-extracted)",
			Command:     "vscode-html-languageserver",
			Description: "HTML/CSS/JS language servers (via vscode-langservers-extracted).",
		},
	}

	for i := range exts {
		exts[i].Installed = checkInstalled(exts[i].Command)
	}

	m := ExtensionsModel{
		items:  exts,
		status: "Use ↑/↓ to navigate, Enter to install, / or Ctrl+F to search, Esc to return",
	}
	m.rebuildFilter()
	return m
}

func checkInstalled(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func installExtensionCmd(index int, cmdName string) tea.Cmd {
	return func() tea.Msg {
		var install *exec.Cmd
		switch cmdName {
		case "gopls":
			install = exec.Command("bash", "-lc", "go install golang.org/x/tools/gopls@latest")
		case "pyright":
			install = exec.Command("bash", "-lc", "npm install -g pyright")
		case "typescript-language-server":
			install = exec.Command("bash", "-lc", "npm install -g typescript typescript-language-server")
		case "vscode-html-languageserver":
			install = exec.Command("bash", "-lc", "npm install -g vscode-langservers-extracted")
		default:
			return InstallResultMsg{
				Index:   index,
				Success: false,
				Output:  "unknown extension: " + cmdName,
			}
		}
		out, err := install.CombinedOutput()
		success := false
		if err == nil {
			success = checkInstalled(cmdName)
		}
		return InstallResultMsg{
			Index:   index,
			Success: success,
			Output:  string(out),
		}
	}
}

func (m *ExtensionsModel) rebuildFilter() {
	m.filteredIdxs = m.filteredIdxs[:0]
	q := strings.TrimSpace(strings.ToLower(m.searchQuery))
	for i := range m.items {
		if q == "" {
			m.filteredIdxs = append(m.filteredIdxs, i)
			continue
		}
		name := strings.ToLower(m.items[i].Name)
		desc := strings.ToLower(m.items[i].Description)
		if strings.Contains(name, q) || strings.Contains(desc, q) {
			m.filteredIdxs = append(m.filteredIdxs, i)
		}
	}
	if len(m.filteredIdxs) == 0 {
		m.selectedIdx = 0
	} else if m.selectedIdx >= len(m.filteredIdxs) {
		m.selectedIdx = len(m.filteredIdxs) - 1
	}
}

func (m ExtensionsModel) Init() tea.Cmd { return nil }

func (m ExtensionsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		k := msg.String()
		if m.searchActive {
			switch k {
			case "esc":
				m.searchActive = false
				m.rebuildFilter()
				return m, nil
			case "backspace":
				if len(m.searchQuery) > 0 {
					m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
					m.rebuildFilter()
				}
				return m, nil
			case "enter":
				if len(m.filteredIdxs) > 0 {
					itemIdx := m.filteredIdxs[m.selectedIdx]
					m.status = fmt.Sprintf("Installing %s ...", m.items[itemIdx].Name)
					return m, installExtensionCmd(itemIdx, m.items[itemIdx].Command)
				}
				return m, nil
			default:
				if len(k) == 1 {
					m.searchQuery += k
					m.rebuildFilter()
				}
				return m, nil
			}
		}

		switch k {
		case "up", "k":
			if m.selectedIdx > 0 {
				m.selectedIdx--
			}
			return m, nil
		case "down", "j":
			if m.selectedIdx < len(m.filteredIdxs)-1 {
				m.selectedIdx++
			}
			return m, nil
		case "/", "ctrl+f":
			m.searchActive = true
			m.searchQuery = ""
			m.rebuildFilter()
			return m, nil
		case "enter":
			if len(m.filteredIdxs) == 0 {
				m.status = "No extension selected"
				return m, nil
			}
			itemIdx := m.filteredIdxs[m.selectedIdx]
			if m.items[itemIdx].Installed {
				m.status = fmt.Sprintf("%s is already installed ✅", m.items[itemIdx].Name)
				return m, nil
			}
			m.status = fmt.Sprintf("Installing %s ...", m.items[itemIdx].Name)
			return m, installExtensionCmd(itemIdx, m.items[itemIdx].Command)
		case "esc":
			return m, nil
		}
		return m, nil

	case InstallResultMsg:
		idx := msg.Index
		if idx >= 0 && idx < len(m.items) {
			m.items[idx].Installed = msg.Success || checkInstalled(m.items[idx].Command)
			if msg.Success {
				m.status = fmt.Sprintf("✅ %s installed successfully", m.items[idx].Name)
			} else {
				out := strings.TrimSpace(msg.Output)
				if out == "" {
					m.status = fmt.Sprintf("❌ %s install failed (check console)", m.items[idx].Name)
				} else {

					if len(out) > 200 {
						out = out[:200] + "..."
					}
					m.status = fmt.Sprintf("❌ %s install failed: %s", m.items[idx].Name, out)
				}
			}
		}

		m.rebuildFilter()
		return m, nil
	}

	return m, nil
}

func (m ExtensionsModel) View() string {
	title := extTitleStyle.Render("🧩 Gonsole Extensions Manager")

	// search bar
	var searchBar string
	if m.searchActive {
		searchBar = searchInputStyle.Render(fmt.Sprintf("🔎 %s", m.searchQuery))
	} else {
		searchBar = searchInputStyle.Render("🔎  Press / or Ctrl+F to search")
	}

	var list strings.Builder
	if len(m.filteredIdxs) == 0 {
		list.WriteString(extItemStyle.Render(" (no results) ") + "\n")
	} else {
		for i, itemIdx := range m.filteredIdxs {
			ext := m.items[itemIdx]
			line := fmt.Sprintf("%s — %s", ext.Name, ext.Description)
			if ext.Installed {
				line += "  ✅"
			} else {
				line += "  ⏳ Not Installed"
			}
			if i == m.selectedIdx {
				list.WriteString(extActiveItemStyle.Render(line) + "\n")
			} else {
				list.WriteString(extItemStyle.Render(line) + "\n")
			}
		}
	}

	status := extStatusStyle.Render(m.status)
	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		searchBar,
		list.String(),
		status,
	)
}

//exten
