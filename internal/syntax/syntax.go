package syntax

import (
	"bytes"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
)

func Highlight(code, lang string) (string, error) {
	var buf bytes.Buffer

	lexer := lexers.Get(lang)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	style := styles.Get("dracula")
	if style == nil {
		style = styles.Fallback
	}

	formatter := formatters.TTY8(true)
	iter, err := lexer.Tokenise(nil, code)
	if err != nil {
		return code, err
	}

	err = formatter.Format(&buf, style, iter)
	if err != nil {
		return code, err
	}

	return buf.String(), nil
}
