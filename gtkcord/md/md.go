package md

import (
	"bytes"
	"regexp"
	"strings"
	"sync"

	"github.com/alecthomas/chroma"
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/state"
	"github.com/gotk3/gotk3/gtk"
)

var regexes = []string{
	// codeblock
	`(?:^\x60\x60\x60 *(\w*)([\s\S]*?)\n?\x60\x60\x60$)`,
	// blockquote
	`((?:(?:^|\n)^>\s+.*)+)\n`,
	// Bullet points, but there's no capture group (disabled)
	`(?:(?:^|\n)(?:[>*+-]|\d+\.)\s+.*)+`,
	// This is actually inline code
	`(?:\x60([^\x60].*?)\x60)`,
	// Inline markup stuff
	`(__|\*\*\*|\*\*|[_*]|~~|\|\|)`,
	// Hyperlinks
	`(https?:\/\S+(?:\.|:)\S+)`,
	// User mentions
	`(?:<@!?(\d+)>)`,
	// Role mentions
	`(?:<@&(\d+)>)`,
	// Channel mentions
	`(?:<#(\d+)>)`,
	// Emojis
	`(<(a?):.*:(\d+)>)`,
}

var HighlightStyle = "solarized-dark"

var (
	style    = (*chroma.Style)(nil)
	regex    = regexp.MustCompile(`(?m)` + strings.Join(regexes, "|"))
	fmtter   = Formatter{}
	css      = map[chroma.TokenType]string{}
	lexerMap = sync.Map{}
)

func Parse(md []byte, buf *gtk.TextBuffer) {
	ParseMessage(nil, nil, md, buf)
}

func ParseMessage(state *state.State, m *discord.Message, md []byte, buf *gtk.TextBuffer) {
	iter := buf.GetEndIter()

	s := statePool.Get().(*mdState)
	defer statePool.Put(s)

	s.submatch(regex, md)

	var tree func(i int)
	if s == nil || m == nil {
		tree = s.switchTree(iter, buf)
	} else {
		tree = s.switchTreeMessage(iter, buf, state, m)
	}

	for i := 0; i < len(s.matches); i++ {
		s.prev = md[s.last:s.matches[i][0].from]
		s.last = s.getLastIndex(i)
		s.chunk = s.chunk[:0] // reset chunk

		tree(i)

		if b := append(escape(s.prev), s.chunk...); len(b) > 0 {
			buf.InsertMarkup(iter, string(b))
		}
	}

	buf.InsertMarkup(iter, string(md[s.last:]))

	// Flush:
	for len(s.context) > 0 {
		buf.InsertMarkup(iter, string(s.tag(s.context[len(s.context)-1])))
	}
}

func renderBlockquote(body []byte) []byte {
	return bytes.Join([][]byte{quoteSpan[0], escape(body), quoteSpan[1]}, nil)
}

func escape(thing []byte) []byte {
	if len(thing) == 0 {
		return nil
	}

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
