package editor

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	kwStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#569CD6")) // blue
	strStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#CE9178")) // orange
	numStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#B5CEA8")) // greenish
	comStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#6A9955")) // comment green
	typeStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#4EC9B0")) // teal
	funcStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#DCDCAA")) // yellow
	constStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#C586C0")) // purple
)

func HighlightSyntax(line, lang string) string {
	switch lang {
	case "go":
		return highlightGo(line)
	case "python":
		return highlightPython(line)
	case "javascript":
		return highlightJS(line)
	case "html":
		return highlightHTML(line)
	default:
		return line
	}
}

func highlightGo(line string) string {
	// Comments
	if strings.HasPrefix(strings.TrimSpace(line), "//") {
		return comStyle.Render(line)
	}

	// Strings
	line = highlightRegex(line, `"[^"]*"`, strStyle)
	line = highlightRegex(line, "`[^`]*`", strStyle)

	// Numbers
	line = highlightRegex(line, `\b\d+(\.\d+)?\b`, numStyle)

	// Keywords
	keywords := []string{
		"package", "import", "func", "return", "if", "else", "for", "range",
		"var", "const", "type", "struct", "interface", "map", "chan", "go",
		"select", "case", "break", "continue", "default", "switch", "defer",
	}
	line = highlightKeywords(line, keywords, kwStyle)

	// Builtin types
	builtins := []string{
		"string", "int", "int64", "float64", "bool", "byte", "rune", "error",
	}
	line = highlightKeywords(line, builtins, typeStyle)

	// Function names
	line = highlightRegex(line, `\b[A-Za-z_][A-Za-z0-9_]*\s*\(`, funcStyle)

	return line
}

// Python
func highlightPython(line string) string {
	if strings.HasPrefix(strings.TrimSpace(line), "#") {
		return comStyle.Render(line)
	}
	line = highlightRegex(line, `"(?:[^"\\]|\\.)*"`, strStyle)
	line = highlightRegex(line, `'(?:[^'\\]|\\.)*'`, strStyle)
	line = highlightRegex(line, `\b\d+(\.\d+)?\b`, numStyle)

	keywords := []string{
		"def", "return", "if", "elif", "else", "for", "while", "import", "from",
		"class", "self", "in", "is", "and", "or", "not", "True", "False", "None",
	}
	line = highlightKeywords(line, keywords, kwStyle)
	line = highlightRegex(line, `\b[A-Za-z_][A-Za-z0-9_]*\s*\(`, funcStyle)
	return line
}

func highlightJS(line string) string {
	line = highlightRegex(line, `"(?:[^"\\]|\\.)*"`, strStyle)
	line = highlightRegex(line, `'(?:[^'\\]|\\.)*'`, strStyle)
	line = highlightRegex(line, "`(?:[^`\\]|\\.)*`", strStyle)

	if strings.HasPrefix(strings.TrimSpace(line), "//") {
		return comStyle.Render(line)
	}
	line = highlightRegex(line, `\b\d+(\.\d+)?\b`, numStyle)

	keywords := []string{
		"let", "const", "var", "function", "return", "if", "else", "for", "while",
		"class", "new", "try", "catch", "finally", "throw", "switch", "case",
		"break", "continue", "import", "export", "default", "async", "await",
	}
	line = highlightKeywords(line, keywords, kwStyle)
	line = highlightRegex(line, `\b[A-Za-z_][A-Za-z0-9_]*\s*\(`, funcStyle)
	return line
}

func highlightHTML(line string) string {
	line = highlightRegex(line, `<!--.*?-->`, comStyle)
	line = highlightRegex(line, `"(?:[^"\\]|\\.)*"`, strStyle)
	line = highlightRegex(line, `<\/?[A-Za-z0-9]+`, kwStyle)
	line = highlightRegex(line, `\b[A-Za-z-]+(?==)`, typeStyle)
	line = highlightRegex(line, `>`, kwStyle)
	return line
}

func highlightKeywords(line string, keywords []string, style lipgloss.Style) string {
	for _, kw := range keywords {
		re := regexp.MustCompile(`\b` + kw + `\b`)
		line = re.ReplaceAllStringFunc(line, func(s string) string {
			return style.Render(s)
		})
	}
	return line
}

func highlightRegex(line, pattern string, style lipgloss.Style) string {
	re := regexp.MustCompile(pattern)
	return re.ReplaceAllStringFunc(line, func(s string) string {
		return style.Render(s)
	})
}
