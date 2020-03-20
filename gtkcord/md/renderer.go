package md

import (
	"io"

	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/yuin/goldmark/ast"
)

// Render is a non-thread-safe TextBuffer renderer.
type Renderer struct {
	Buffer *gtk.TextBuffer
	State  *ningen.State

	tags TagState

	// runs afterwards sequentially
	setEmojis chan setEmoji
}

type setEmoji struct {
	pb    *gdk.Pixbuf
	line  int
	index int
}

func NewRenderer(buf *gtk.TextBuffer, state *ningen.State) *Renderer {
	tags, _ := buf.GetTagTable()

	return &Renderer{
		Buffer: buf,
		State:  state,
		tags: TagState{
			table: tags,
			// deal with nil tags
		},
	}
}

func (r *Renderer) Render(_ io.Writer, source []byte, n ast.Node) error {
	return ast.Walk(n, func(n ast.Node, enter bool) (ast.WalkStatus, error) {
		return r.renderNode(source, n, enter)
	})
}

func (r *Renderer) renderNode(source []byte, n ast.Node, enter bool) (ast.WalkStatus, error) {
	switch n := n.(type) {
	case *ast.Blockquote:
		r.tags.tagSet(AttrQuoted, enter)
		if enter {
			r.insertWithTag([]byte("> "), nil)
		} else {
			r.insertWithTag([]byte("\n"), nil)
		}

	case *ast.FencedCodeBlock:
		// TODO

	case *ast.Paragraph:
		if !enter {
			r.insertWithTag([]byte("\n"), nil)
		}

	// TODO: a better autolink
	case *ast.AutoLink:
		if enter {
			url := n.URL(source)

			tag := r.tags.hyperlink(string(url))
			r.tags.injectTag(tag)

			r.insertWithTag(url, tag)
		}

	case *ast.CodeSpan:
		r.tags.tagSet(AttrMonospace, enter)

	case *ast.Emphasis:
		if n.Level == 2 {
			r.tags.tagSet(AttrBold, enter)
		} else {
			r.tags.tagSet(AttrItalics, enter)
		}

	case *ast.String:
		if !enter {
			break
		}

		r.insertWithTag(n.Value, nil)

	case *ast.Text:
		if !enter {
			break
		}

		segment := n.Segment
		r.insertWithTag(segment.Value(source), nil)

		switch {
		case n.HardLineBreak():
			r.insertWithTag([]byte("\n"), nil)
			fallthrough
		case n.SoftLineBreak():
			r.insertWithTag([]byte("\n"), nil)

			// Check if blockquote prefix:
			if r.tags.attr.Has(AttrQuoted) {
				r.insertWithTag([]byte("> "), nil)
			}
		}

		// Iterate
	}

	return ast.WalkContinue, nil
}

func (r *Renderer) insertWithTag(content []byte, tag *gtk.TextTag) {
	if tag == nil {
		tag = r.tags.tag
	}

	r.Buffer.InsertWithTag(r.Buffer.GetEndIter(), string(content), tag)
}
