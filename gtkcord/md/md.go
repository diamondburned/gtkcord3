package md

import (
	"bytes"
	"regexp"
	"sync"

	"github.com/alecthomas/chroma"
)

const mdRegex = `(?m)` +
	`(?:^\x60\x60\x60 *(\w*)([\s\S]*?)\n?\x60\x60\x60$)` +
	`|((?:(?:^|\n)^>\s+.*)+)\n` +
	`|(?:(?:^|\n)(?:[>*+-]|\d+\.)\s+.*)+` +
	`|(?:\x60([^\x60].*?)\x60)` +
	`|(__|\*\*\*|\*\*|[_*]|~~|\|\|)` +
	`|(https?:\/\S+(?:\.|:)\S+)`

var HighlightStyle = "solarized-dark"

var (
	style    = (*chroma.Style)(nil)
	regex    = regexp.MustCompile(mdRegex)
	fmtter   = Formatter{}
	css      = map[chroma.TokenType]string{}
	lexerMap = sync.Map{}
)

func Parse(md []byte) []byte {
	s := statePool.Get().(*mdState)
	defer statePool.Put(s)

	s.submatch(regex, md)

	for i := 0; i < len(s.matches); i++ {
		s.prev = md[s.last:s.matches[i][0].from]
		s.last = s.getLastIndex(i)
		s.chunk = s.chunk[:0] // reset chunk

		switch {
		case len(s.matches[i][2].str) > 0:
			// codeblock
			s.chunk = s.renderCodeBlock(
				s.matches[i][1].str,
				s.matches[i][2].str,
			)
		case len(s.matches[i][3].str) > 0:
			// blockquotes, greentext
			s.chunk = renderBlockquote(s.matches[i][3].str)
		case len(s.matches[i][4].str) > 0:
			// inline code
			s.chunk = bytes.Join([][]byte{codeSpan[0], s.matches[i][4].str, codeSpan[1]}, nil)
		case len(s.matches[i][5].str) > 0:
			// inline stuff
			s.chunk = s.tag(s.matches[i][5].str)
		case len(s.matches[i][6].str) > 0:
			// URLs
			s.chunk = s.matches[i][6].str
		case bytes.Count(s.prev, []byte(`\`))%2 != 0:
			// escaped, print raw
			s.chunk = escape(s.matches[i][0].str)
		default:
			s.chunk = escape(s.matches[i][0].str)
		}

		s.output = append(s.output, escape(s.prev)...)
		s.output = append(s.output, s.chunk...)
	}

	s.output = append(s.output, md[s.last:]...)

	// Flush:
	for len(s.context) > 0 {
		s.output = append(s.output, s.tag(s.context[len(s.context)-1])...)
	}

	return s.output
}

func renderBlockquote(body []byte) []byte {
	return bytes.Join([][]byte{quoteSpan[0], escape(body), quoteSpan[1]}, nil)
}

func escape(thing []byte) []byte {
	// escaped := thing[:0]
	escaped := make([]byte, 0, len(thing))

	for i := 0; i < len(thing); i++ {
		switch thing[i] {
		case '&':
			escaped = append(escaped, escapes[0]...)
		case '\'':
			escaped = append(escaped, escapes[1]...)
		case '<':
			escaped = append(escaped, escapes[2]...)
		case '>':
			escaped = append(escaped, escapes[3]...)
		default:
			escaped = append(escaped, thing[i])
		}
	}

	return escaped
}
