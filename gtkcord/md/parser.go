package md

import (
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/util"
)

func BlockParsers() []util.PrioritizedValue {
	return []util.PrioritizedValue{
		util.Prioritized(fenced{}, 500), // code blocks
		util.Prioritized(blockquote{}, 800),
		util.Prioritized(paragraph{}, 1000),
	}
}

func InlineParsers() []util.PrioritizedValue {
	return []util.PrioritizedValue{
		util.Prioritized(parser.NewCodeSpanParser(), 100),
		util.Prioritized(autolink{}, 300),
		util.Prioritized(parser.NewEmphasisParser(), 500),
	}
}
