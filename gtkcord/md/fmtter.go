package md

import (
	"html"
	"io"
	"strings"

	"github.com/alecthomas/chroma"
)

// Formatter that generates Pango markup.
type Formatter struct {
	highlightRanges [][2]int
}

func (f *Formatter) Reset() {
	f.highlightRanges = f.highlightRanges[:0]
}

func (f *Formatter) Format(w io.Writer, iterator chroma.Iterator) error {
	tokens := iterator.Tokens()
	lines := chroma.SplitTokensIntoLines(tokens)
	highlightIndex := 0

	w.Write([]byte(
		`<span font_family="monospace" font_size="smaller" ` +
			f.styleAttr(chroma.Background) + ">"))

	highlightIndex = 0
	for index, tokens := range lines {
		// 1-based line number.
		line := 1 + index
		highlight, next := f.shouldHighlight(highlightIndex, line)
		if next {
			highlightIndex++
		}

		if highlight {
			w.Write([]byte("<span " + f.styleAttr(chroma.LineHighlight) + ">"))
		}

		for _, token := range tokens {
			html := html.EscapeString(token.String())
			attr := f.styleAttr(token.Type)
			if attr != "" {
				html = "<span " + attr + ">" + html + "</span>"
			}
			w.Write([]byte(html))
		}
		if highlight {
			w.Write([]byte("</span>"))
		}
	}

	w.Write([]byte("</span>"))

	return nil
}

func (f *Formatter) shouldHighlight(highlightIndex, line int) (bool, bool) {
	next := false
	for highlightIndex < len(f.highlightRanges) && line > f.highlightRanges[highlightIndex][1] {
		highlightIndex++
		next = true
	}
	if highlightIndex < len(f.highlightRanges) {
		hrange := f.highlightRanges[highlightIndex]
		if line >= hrange[0] && line <= hrange[1] {
			return true, next
		}
	}
	return false, next
}

func (f *Formatter) styleAttr(tt chroma.TokenType, extraCSS ...string) string {
	if _, ok := css[tt]; !ok {
		tt = tt.SubCategory()
	}
	if _, ok := css[tt]; !ok {
		tt = tt.Category()
	}
	if _, ok := css[tt]; !ok {
		return ""
	}

	return strings.Join(append([]string{css[tt]}, extraCSS...), " ")
}

func styleToCSS(style *chroma.Style) map[chroma.TokenType]string {
	classes := map[chroma.TokenType]string{}
	bg := style.Get(chroma.Background)
	for t := range chroma.StandardTypes {
		var entry = style.Get(t)
		if t != chroma.Background {
			entry = entry.Sub(bg)
		}
		if entry.IsZero() {
			continue
		}
		classes[t] = styleEntryToCSS(entry)
	}
	return classes
}

// styleEntryToCSS converts a chroma.StyleEntry to CSS attributes.
func styleEntryToCSS(e chroma.StyleEntry) string {
	styles := []string{}
	if e.Colour.IsSet() {
		styles = append(styles, `fgcolor="`+e.Colour.String()+`"`)
	}
	if e.Bold == chroma.Yes {
		styles = append(styles, `weight="bold"`)
	}
	if e.Italic == chroma.Yes {
		styles = append(styles, `style="italic"`)
	}
	if e.Underline == chroma.Yes {
		styles = append(styles, `underline="single"`)
	}
	return strings.Join(styles, " ")
}
