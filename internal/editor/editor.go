package editor

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/alecthomas/chroma/formatters"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/creack/pty"
)

var (
	headerStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#0078D4")).
			Foreground(lipgloss.Color("#ffffff")).
			Bold(true).
			Padding(0, 1)

	sidebarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#C0C0C0")).
			Background(lipgloss.Color("#1e1e1e")).
			Padding(1, 1)

	activeFileStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00BFFF")).
			Bold(true)

	cursorStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#005F87")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true)

	lineNumStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#5f87af"))

	editorBgStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#000000")).
			Foreground(lipgloss.Color("#e0e0e0")).
			Padding(1, 2)

	statusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#2D2D2D")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 1)

	terminalStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#0b0b0b")).
			Foreground(lipgloss.Color("#00ff88")).
			Padding(0, 1)

	searchBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#0078D4")).
			Foreground(lipgloss.Color("#ffffff")).
			Padding(0, 1).
			Bold(true)

	highlightStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#005A9E")).
			Foreground(lipgloss.Color("#ffffff"))
)

type TerminalOutputMsg string

type snapshot struct {
	lines    []string
	cursorX  int
	cursorY  int
	filepath string
}

type Model struct {
	cursorX, cursorY int
	lines            []string
	file             string
	lang             string
	status           string
	files            []string
	dir              string
	selectedIdx      int
	mode             string // "editor" or "sidebar"
	scrollTop        int
	visibleRows      int
	width, height    int

	// Undo / Redo
	history     []snapshot
	redoHistory []snapshot

	// Terminal
	showTerminal bool
	termViewport viewport.Model
	termBuffer   string
	ptyFile      *os.File
	ptyCmd       *exec.Cmd

	// Search
	searchActive  bool
	searchQuery   string
	searchResults []int
	searchIndex   int

	// Extensions
	showExtensions bool
	extModel       ExtensionsModel
}

func New() Model {
	m := Model{
		lines:       []string{""},
		status:      "New file",
		mode:        "editor",
		scrollTop:   0,
		visibleRows: 25,
		extModel:    NewExtensionsModel(),
	}

	if len(os.Args) > 1 {
		file := os.Args[1]
		m.file = file
		m.detectLang(file)
		m.loadFile(file)
		m.loadDir(filepath.Dir(file))
	} else {
		m.lang = "plaintext"
		m.loadDir(".")
	}

	m.termViewport = viewport.New(10, 10)
	m.termViewport.Style = terminalStyle

	if f, cmd, err := startPty(); err == nil {
		m.ptyFile = f
		m.ptyCmd = cmd
	} else {
		m.status = fmt.Sprintf("pty error: %v", err)
	}

	m.saveSnapshot()
	return m
}

func startPty() (*os.File, *exec.Cmd, error) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "bash"
	}
	cmd := exec.Command(shell)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setctty: true, Setsid: true}
	f, err := pty.Start(cmd)
	if err != nil {
		return nil, nil, err
	}
	return f, cmd, nil
}

func (m *Model) readPtyOnce() tea.Cmd {
	return func() tea.Msg {
		if m.ptyFile == nil {
			return TerminalOutputMsg("")
		}
		buf := make([]byte, 2048)
		n, err := m.ptyFile.Read(buf)
		if err != nil {
			return TerminalOutputMsg("")
		}
		return TerminalOutputMsg(string(buf[:n]))
	}
}

func (m *Model) loadDir(path string) {
	m.dir = path
	items, err := os.ReadDir(path)
	if err != nil {
		m.files = []string{"<cannot read dir>"}
		return
	}
	m.files = nil
	for _, f := range items {
		if f.IsDir() {
			m.files = append(m.files, "📁 "+f.Name()+"/")
		} else {
			m.files = append(m.files, "📄 "+f.Name())
		}
	}
}

func (m *Model) detectLang(path string) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		m.lang = "go"
	case ".py":
		m.lang = "python"
	case ".js":
		m.lang = "javascript"
	case ".html":
		m.lang = "html"
	default:
		m.lang = "plaintext"
	}
}

func (m *Model) loadFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		m.status = fmt.Sprintf("cannot open %s: %v", path, err)
		return
	}
	m.lines = strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	m.status = fmt.Sprintf("Opened %s [%s]", path, m.lang)
	m.saveSnapshot()
}

func (m *Model) saveFile() {
	if m.file == "" {
		m.file = "untitled.txt"
	}
	content := strings.Join(m.lines, "\n")
	_ = os.WriteFile(m.file, []byte(content), 0644)
	m.status = fmt.Sprintf("Saved %s [%s]", m.file, m.lang)
}

func (m *Model) saveSnapshot() {
	copyLines := append([]string{}, m.lines...)
	m.history = append(m.history, snapshot{
		lines:    copyLines,
		cursorX:  m.cursorX,
		cursorY:  m.cursorY,
		filepath: m.file,
	})
	if len(m.history) > 200 {
		m.history = m.history[len(m.history)-200:]
	}
	m.redoHistory = nil
}

func (m *Model) undo() {
	if len(m.history) <= 1 {
		return
	}
	current := m.history[len(m.history)-1]
	m.redoHistory = append(m.redoHistory, current)
	m.history = m.history[:len(m.history)-1]
	last := m.history[len(m.history)-1]
	m.lines = append([]string{}, last.lines...)
	m.cursorX = last.cursorX
	m.cursorY = last.cursorY
}

func (m *Model) redo() {
	if len(m.redoHistory) == 0 {
		return
	}
	next := m.redoHistory[len(m.redoHistory)-1]
	m.redoHistory = m.redoHistory[:len(m.redoHistory)-1]
	m.history = append(m.history, next)
	m.lines = append([]string{}, next.lines...)
	m.cursorX = next.cursorX
	m.cursorY = next.cursorY
}

func (m *Model) updateSearchResults() {
	m.searchResults = nil
	if m.searchQuery == "" {
		return
	}
	for i, line := range m.lines {
		if strings.Contains(strings.ToLower(line), strings.ToLower(m.searchQuery)) {
			m.searchResults = append(m.searchResults, i)
		}
	}
	m.searchIndex = 0
}

func (m *Model) highlightCode(code string) string {
	lexer := lexers.Get(m.lang)
	if lexer == nil {
		lexer = lexers.Analyse(code)
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}
	style := styles.Get("dracula")
	if style == nil {
		style = styles.Fallback
	}
	formatter := formatters.TTY
	iterator, _ := lexer.Tokenise(nil, code)
	var buf bytes.Buffer
	if err := formatter.Format(&buf, style, iterator); err != nil {
		return code
	}
	return buf.String()
}

func (m Model) toggleExtensions() Model {
	m.showExtensions = !m.showExtensions
	return m
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.showExtensions {
		updated, cmd := m.extModel.Update(msg)
		if nm, ok := updated.(ExtensionsModel); ok {
			m.extModel = nm
		}
		if km, ok := msg.(tea.KeyMsg); ok && km.String() == "esc" {
			m.showExtensions = false
		}
		return m, cmd
	}

	switch msg := msg.(type) {
	case TerminalOutputMsg:
		if s := string(msg); s != "" {
			m.termBuffer += s
			m.termViewport.SetContent(m.termBuffer)
			m.termViewport.GotoBottom()
		}
		return m, m.readPtyOnce()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		termH := 10
		if m.showTerminal {
			termH = max(6, m.height/4)
		}
		m.visibleRows = m.height - termH - 4
		if m.visibleRows < 5 {
			m.visibleRows = 5
		}
		m.termViewport.Width = m.width
		m.termViewport.Height = termH
		return m, nil

	case tea.KeyMsg:
		k := msg.String()

		if m.searchActive {
			switch k {
			case "esc":
				m.searchActive = false
				return m, nil
			case "enter":
				if len(m.searchResults) > 0 {
					m.searchIndex = (m.searchIndex + 1) % len(m.searchResults)
					m.cursorY = m.searchResults[m.searchIndex]
					if m.cursorY < m.scrollTop {
						m.scrollTop = m.cursorY
					} else if m.cursorY >= m.scrollTop+m.visibleRows {
						m.scrollTop = m.cursorY - m.visibleRows + 1
					}
				}
				return m, nil
			case "backspace":
				if len(m.searchQuery) > 0 {
					m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
					m.updateSearchResults()
				}
				return m, nil
			default:
				if len(k) == 1 {
					m.searchQuery += k
					m.updateSearchResults()
				}
				return m, nil
			}
		}

		if m.showTerminal {
			switch k {
			case "ctrl+t", "esc":
				m.showTerminal = false
				return m, nil
			case "enter":
				if m.ptyFile != nil {
					_, _ = m.ptyFile.Write([]byte{'\r'})
				}
				return m, nil
			case "backspace":
				if m.ptyFile != nil {
					_, _ = m.ptyFile.Write([]byte{0x7f})
				}
				return m, nil
			default:
				if m.ptyFile != nil {
					_, _ = m.ptyFile.Write([]byte(k))
				}
				return m, nil
			}
		}

		switch k {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "ctrl+t":
			m.showTerminal = !m.showTerminal
			if m.showTerminal {
				return m, m.readPtyOnce()
			}
			return m, nil
		case "ctrl+s":
			m.saveFile()
			return m, nil
		case "ctrl+f":
			m.searchActive = true
			m.searchQuery = ""
			m.searchResults = nil
			m.searchIndex = 0
			return m, nil
		case "ctrl+z":
			m.undo()
			return m, nil
		case "ctrl+y":
			m.redo()
			return m, nil
		case "ctrl+e":
			m.showExtensions = true
			return m, nil
		case "tab":
			if m.mode == "editor" {
				m.mode = "sidebar"
			} else {
				m.mode = "editor"
			}
			return m, nil
		case "up":
			if m.mode == "editor" && m.cursorY > 0 {
				m.cursorY--
			}
			if m.cursorY < m.scrollTop {
				m.scrollTop = m.cursorY
				if m.scrollTop < 0 {
					m.scrollTop = 0
				}
			}
			if m.mode == "sidebar" && m.selectedIdx > 0 {
				m.selectedIdx--
			}
			return m, nil
		case "down":
			if m.mode == "editor" && m.cursorY < len(m.lines)-1 {
				m.cursorY++
			}
			if m.cursorY >= m.scrollTop+m.visibleRows {
				m.scrollTop++
			}
			if m.mode == "sidebar" && m.selectedIdx < len(m.files)-1 {
				m.selectedIdx++
			}
			return m, nil
		case "enter":
			if m.mode == "editor" {
				m.lines = append(m.lines[:m.cursorY+1],
					append([]string{""}, m.lines[m.cursorY+1:]...)...)
				m.cursorY++
				m.cursorX = 0
				m.saveSnapshot()
			} else if m.mode == "sidebar" {
				item := m.files[m.selectedIdx]
				clean := strings.TrimPrefix(item, "📄 ")
				clean = strings.TrimPrefix(clean, "📁 ")
				if strings.HasSuffix(clean, "/") {
					m.loadDir(filepath.Join(m.dir, strings.TrimSuffix(clean, "/")))
					m.selectedIdx = 0
				} else {
					m.file = filepath.Join(m.dir, clean)
					m.detectLang(m.file)
					m.loadFile(m.file)
					m.mode = "editor"
				}
			}
			return m, nil
		case "left":
			if m.cursorX > 0 {
				m.cursorX--
			}
			return m, nil
		case "right":
			if m.cursorX < len(m.lines[m.cursorY]) {
				m.cursorX++
			}
			return m, nil
		case "backspace":
			if m.cursorX > 0 {
				line := m.lines[m.cursorY]
				m.lines[m.cursorY] = line[:m.cursorX-1] + line[m.cursorX:]
				m.cursorX--
			} else if m.cursorY > 0 {
				prev := m.lines[m.cursorY-1]
				curr := m.lines[m.cursorY]
				m.lines = append(m.lines[:m.cursorY-1],
					append([]string{prev + curr}, m.lines[m.cursorY+1:]...)...)
				m.cursorY--
				m.cursorX = len(prev)
			}
			m.saveSnapshot()
			return m, nil
		default:
			// printable insertion
			if len(k) == 1 && m.mode == "editor" {
				line := m.lines[m.cursorY]
				m.lines[m.cursorY] = line[:m.cursorX] + k + line[m.cursorX:]
				m.cursorX++
				m.saveSnapshot()
			}
			return m, nil
		}
	}

	return m, nil
}

func (m Model) renderEditor() string {
	start := m.scrollTop
	end := m.scrollTop + m.visibleRows
	if end > len(m.lines) {
		end = len(m.lines)
	}
	var builder strings.Builder
	for i := start; i < end; i++ {
		lineNum := fmt.Sprintf("%4d ", i+1)
		line := m.lines[i]

		if m.searchQuery != "" {
			lower := strings.ToLower(line)
			q := strings.ToLower(m.searchQuery)
			if strings.Contains(lower, q) {

				var b strings.Builder
				idx := 0
				for {
					pos := strings.Index(strings.ToLower(line[idx:]), q)
					if pos == -1 {
						b.WriteString(line[idx:])
						break
					}
					b.WriteString(line[idx : idx+pos])
					b.WriteString(highlightStyle.Render(line[idx+pos : idx+pos+len(q)]))
					idx = idx + pos + len(q)
				}
				line = b.String()
			}
		}

		h := m.highlightCode(line)

		if i == m.cursorY {
			orig := m.lines[i]
			pos := m.cursorX
			if pos > len(orig) {
				pos = len(orig)
			}
			left := orig[:pos]
			right := ""
			if pos < len(orig) {
				right = orig[pos:]
			}
			leftH := m.highlightCode(left)
			cursorChar := " "
			if pos < len(orig) {
				cursorChar = string(orig[pos])
			}
			rightH := m.highlightCode(right)
			cursorRendered := cursorStyle.Render(cursorChar)
			h = leftH + cursorRendered + rightH
		}

		builder.WriteString(lineNumStyle.Render(lineNum) + h + "\n")
	}
	return editorBgStyle.Width(max(20, m.width-30)).Render(builder.String())
}

func (m Model) renderSidebar() string {
	var out string
	for i, f := range m.files {
		if i == m.selectedIdx && m.mode == "sidebar" {
			out += activeFileStyle.Render("→ " + f + "\n")
		} else {
			out += sidebarStyle.Render("  " + f + "\n")
		}
	}
	return sidebarStyle.Width(30).Height(m.height - 4).Render(out)
}

func (m Model) View() string {
	if m.showExtensions {
		return m.extModel.View()
	}

	sidebar := m.renderSidebar()
	editorView := m.renderEditor()
	content := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, editorView)
	header := headerStyle.Width(m.width).Render(fmt.Sprintf(" Gonsole — %s ", filepath.Base(m.file)))
	status := statusBarStyle.Width(m.width).Render(fmt.Sprintf(
		"📁 %s | 🧩 Ctrl+E Extensions | 🧠 %s | Ln %d, Col %d | Ctrl+S Save | Ctrl+F Search | Ctrl+Z Undo | Ctrl+T Terminal",
		filepath.Base(m.file), m.lang, m.cursorY+1, m.cursorX+1,
	))

	if m.searchActive {
		searchBar := searchBarStyle.Width(m.width).
			Render(fmt.Sprintf("🔍 Find: %s  (%d/%d)", m.searchQuery, m.searchIndex+1, len(m.searchResults)))
		return lipgloss.JoinVertical(lipgloss.Left, header, searchBar, content, status)
	}

	if m.showTerminal {
		m.termViewport.SetContent(m.termBuffer)
		m.termViewport.GotoBottom()
		return lipgloss.JoinVertical(lipgloss.Left, header, content, status, m.termViewport.View())
	}
	return lipgloss.JoinVertical(lipgloss.Left, header, content, status)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
