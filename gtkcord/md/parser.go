package md

import (
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

func BlockParsers() []util.PrioritizedValue {
	return []util.PrioritizedValue{
		util.Prioritized(blockquote{}, 500),
		util.Prioritized(fenced{}, 800), // code blocks
		util.Prioritized(paragraph{}, 1000),
	}
}

func InlineParsers() []util.PrioritizedValue {
	return []util.PrioritizedValue{
		util.Prioritized(emoji{}, 200),
		util.Prioritized(inline{}, 300),
		util.Prioritized(mention{}, 400),
		util.Prioritized(autolink{}, 500),
	}
}

// matchInline function to parse a pair of bytes (chars)
func matchInline(r text.Reader, open, close byte) []byte {
	line, _ := r.PeekLine()

	start := 0
	for ; start < len(line) && line[start] != open; start++ {
	}

	stop := start
	for ; stop < len(line) && line[stop] != close; stop++ {
	}

	// Advance total distance:
	r.Advance(stop)

	stop++ // add the '>'

	// This would be true if there's no closure.
	if stop == len(line) {
		return nil
	}

	r.Advance(1)

	return line[start:stop]
}
