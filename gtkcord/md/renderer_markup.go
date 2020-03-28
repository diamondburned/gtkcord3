package md

import (
	"html"
	"io"

	"github.com/yuin/goldmark/ast"
)

type MarkupRenderer struct {
	attr Attribute
}

func NewMarkupRenderer() *MarkupRenderer {
	return &MarkupRenderer{}
}

func (r *MarkupRenderer) Render(w io.Writer, source []byte, n ast.Node) error {
	ast.Walk(n, func(n ast.Node, enter bool) (ast.WalkStatus, error) {
		r.switchNode(w, n, source, enter)
		return ast.WalkContinue, nil
	})

	// Close if needed:
	r.closeAttr(w)

	return nil
}

func (r *MarkupRenderer) switchNode(w io.Writer, n ast.Node, source []byte, enter bool) {
	switch n := n.(type) {
	case *ast.Document:
		// noop

	case *ast.Paragraph:
		if !enter {
			w.Write([]byte{'\n'})
		}

	case *ast.Blockquote:
		w.Write([]byte{'\n'})
		if enter {
			w.Write([]byte{'>', ' '})
		} else {
			w.Write([]byte{'\n'})
		}

	case *ast.FencedCodeBlock:
		w.Write([]byte{'\n'})
		if enter {
			// Temporarily close last tag:
			old := r.attr
			r.closeAttr(w)
			r.attr = 0

			// Write an opening tag:
			r.setAttr(w, AttrMonospace, true)

			// Write the code body
			for i := 0; i < n.Lines().Len(); i++ {
				line := n.Lines().At(i)
				writeEscape(w, line.Value(source))
			}

			// Close the tag:
			r.closeAttr(w)

			// Restore the tag:
			r.attr = 0
			r.setAttr(w, old, true)
		}

	case *ast.AutoLink:
		if enter {
			w.Write([]byte(`<a href="`))
			writeEscape(w, n.URL(source))
			w.Write([]byte(`">`))
			writeEscape(w, n.URL(source))
			w.Write([]byte(`</a>`))
		}

	case *Inline:
		r.setAttr(w, n.Attr, enter)

	case *Emoji:
		if enter {
			// Only write :emojiName:
			writeEscape(w, []byte(":"+string(n.Name)+":"))
		}

	case *Mention:
		if enter {
			switch {
			case n.Channel != nil:
				writeEscape(w, []byte("#"+n.Channel.Name))
			case n.GuildUser != nil:
				writeEscape(w, []byte("@"+n.GuildUser.Username))
			}
		}

	case *ast.String:
		if !enter {
			break
		}

		writeEscape(w, n.Value)

	case *ast.Text:
		if !enter {
			break
		}

		writeEscape(w, n.Segment.Value(source))

		switch {
		case n.HardLineBreak():
			w.Write([]byte{'\n'})
			fallthrough
		case n.SoftLineBreak():
			w.Write([]byte{'\n'})

			// Check blockquote:
			if r.attr.Has(AttrQuoted) {
				w.Write([]byte{'>', ' '})
			}
		}
	}
}

func (r *MarkupRenderer) setAttr(w io.Writer, attr Attribute, enter bool) {
	// close the original span if there's one
	r.closeAttr(w)

	// add/remove to tag
	if enter {
		r.attr.Add(attr)
	} else {
		r.attr.Remove(attr)
	}

	// generate a new span if needed
	if r.attr != 0 {
		w.Write([]byte(`<span ` + r.attr.Markup() + `>`))
	}
}

// func (r *MarkupRenderer) writeAttr(w io.Writer, attr string) {
// 	r.closeAttr(w)

// 	r.lastAttr = attr
// 	w.Write([]byte("<span " + attr + ">"))
// }

func (r *MarkupRenderer) closeAttr(w io.Writer) {
	if r.attr != 0 {
		w.Write([]byte("</span>"))
	}
}

func writeEscape(w io.Writer, unescaped []byte) {
	w.Write([]byte(html.EscapeString(string(unescaped))))
}
