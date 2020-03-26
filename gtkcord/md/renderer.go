package md

import (
	"io"
	"sync"

	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
)

// Render is a non-thread-safe TextBuffer renderer.
type Renderer struct {
	Buffer *gtk.TextBuffer

	tags TagState

	// runs afterwards sequentially
	setEmojis chan setEmoji
	setGroup  sync.WaitGroup
}

type setEmoji struct {
	pb    *gdk.Pixbuf
	line  int
	index int
}

func NewRenderer(buf *gtk.TextBuffer) *Renderer {
	tags, _ := buf.GetTagTable()

	return &Renderer{
		Buffer: buf,
		tags: TagState{
			table: tags,
		},
		setEmojis: make(chan setEmoji), // arbitrary 8
	}
}

func (r *Renderer) Render(_ io.Writer, source []byte, n ast.Node) error {
	ast.Walk(n, func(n ast.Node, enter bool) (ast.WalkStatus, error) {
		return r.renderNode(source, n, enter)
	})

	// Start a cleanup goroutine
	go func() {
		r.setGroup.Wait()
		close(r.setEmojis)
	}()

	go func() {
		// Emoji tag that slightly offsets the emoji vertically.
		emojiTag := r.tags.inlineEmojiTag()

		for set := range r.setEmojis {
			set := set

			// Start setting emojis in the background.
			semaphore.Async(func() {
				last := r.Buffer.GetIterAtLineIndex(set.line, set.index)
				fwdi := r.Buffer.GetIterAtLineIndex(set.line, set.index)
				fwdi.ForwardChar()

				r.Buffer.Delete(last, fwdi)
				r.Buffer.InsertPixbuf(last, set.pb)

				first := r.Buffer.GetIterAtLineIndex(set.line, set.index)
				r.Buffer.ApplyTag(emojiTag, first, last)
			})
		}
	}()

	return nil
}

// AddOptions is a noop.
func (r *Renderer) AddOptions(...renderer.Option) {}

func (r *Renderer) renderNode(source []byte, n ast.Node, enter bool) (ast.WalkStatus, error) {
	switch n := n.(type) {
	case *ast.Document:
		// noop

	case *ast.Paragraph:
		if !enter {
			r.insertWithTag([]byte{'\n'}, nil)
		}

	case *ast.Blockquote:
		r.tags.tagSet(AttrQuoted, enter)
		if enter {
			r.insertWithTag([]byte{'>', ' '}, nil)
		} else {
			r.insertWithTag([]byte{'\n'}, nil)
		}

	case *ast.FencedCodeBlock:
		// Insert a new line on both enter and exit:
		r.insertWithTag([]byte{'\n'}, nil)
		if enter {
			r.renderCodeBlock(n, source)
		}

	case *ast.AutoLink:
		if enter {
			url := n.URL(source)

			tag := r.tags.hyperlink(string(url))
			r.tags.injectTag(tag)

			r.insertWithTag(url, tag)
		}

	case *Inline:
		r.tags.tagSet(n.Attr, enter)

	case *Emoji: // TODO
		if enter {
			r.insertEmoji(n.EmojiURL())
		}

	case *Mention:
		if enter {
			switch {
			case n.Channel != nil:
				r.insertWithTag([]byte("#"+n.Channel.Name), r.tags.channel(n.Channel))

			case n.GuildUser != nil:
				var name = n.GuildUser.Username
				if n.GuildUser.Member != nil && n.GuildUser.Member.Nick != "" {
					name = n.GuildUser.Member.Nick
				}
				r.insertWithTag([]byte("@"+name), r.tags.guildUser(n.GuildUser))
			}
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
			if r.tags.Attr.Has(AttrQuoted) {
				r.insertWithTag([]byte{'>', ' '}, nil)
			}
		}
	}

	return ast.WalkContinue, nil
}

func (r *Renderer) insertWithTag(content []byte, tag *gtk.TextTag) {
	if tag == nil {
		tag = r.tags.tag
	}

	r.Buffer.InsertWithTag(r.Buffer.GetEndIter(), string(content), tag)
}
