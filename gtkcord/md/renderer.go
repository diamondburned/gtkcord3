package md

import (
	"io"

	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/ningen/v2/md"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
)

const (
	InlineEmojiSize = 22
	LargeEmojiSize  = 48
)

func EmojiURL(emojiID string, animated bool) string {
	const EmojiBaseURL = "https://cdn.discordapp.com/emojis/"

	if animated {
		return EmojiBaseURL + emojiID + ".gif?v=1"
	}

	return EmojiBaseURL + emojiID + ".png?v=1"
}

// Render is a non-thread-safe TextBuffer renderer.
type Renderer struct {
	View   *gtk.TextView
	Buffer *gtk.TextBuffer

	end  *gtk.TextIter
	tags TagState
}

func NewRenderer(tv *gtk.TextView) *Renderer {
	buf := tv.Buffer()
	tags := buf.TagTable()

	return &Renderer{
		View:   tv,
		Buffer: buf,
		tags: TagState{
			table: tags,
		},
	}
}

func (r *Renderer) Render(_ io.Writer, source []byte, n ast.Node) error {
	r.Buffer.SetText("")

	ast.Walk(n, func(n ast.Node, enter bool) (ast.WalkStatus, error) {
		return r.renderNode(source, n, enter)
	})

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
		r.tags.tagSet(md.AttrQuoted, enter)
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

	case *ast.Link:
		if enter {
			tag := r.tags.hyperlink(string(n.Destination))
			r.tags.injectTag(tag)
			// Shitty hack to hijack hyperlink into tag, since markdown is trash.
			r.tags.tag = tag
			r.insertWithTag(n.Title, nil) // use replaced tag
		} else {
			// Reset above tag state.
			r.tags.tagAdd(0)
		}

	case *ast.AutoLink:
		if enter {
			url := n.URL(source)
			tag := r.tags.hyperlink(string(url))
			r.tags.injectTag(tag)
			r.insertWithTag(url, tag)
		}

	case *md.Inline:
		r.tags.tagSet(n.Attr, enter)

	case *md.Emoji:
		if enter {
			r.insertEmoji(n)
		}

	case *md.Mention:
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
			if r.tags.Attr.Has(md.AttrQuoted) {
				r.insertWithTag([]byte{'>', ' '}, nil)
			}
		}
	}

	return ast.WalkContinue, nil
}

func (r *Renderer) endIter() *gtk.TextIter {
	if r.end == nil {
		r.end = r.Buffer.EndIter()
	}
	return r.end
}

func (r *Renderer) insertWithTag(content []byte, tag *gtk.TextTag) {
	if tag == nil {
		tag = r.tags.tag
	}

	end := r.endIter()

	var startIx int
	if tag != nil {
		startIx = end.Offset()
	}

	r.Buffer.Insert(end, string(content))

	if tag != nil {
		r.Buffer.ApplyTag(tag, r.Buffer.IterAtOffset(startIx), end)
	}
}

func (r *Renderer) insertEmoji(e *md.Emoji) {
	// TODO
	sz := InlineEmojiSize
	if e.Large {
		sz = LargeEmojiSize
	}

	anchor := r.Buffer.CreateChildAnchor(r.endIter())

	img := gtk.NewImage()
	img.SetTooltipText(e.Name)
	img.SetSizeRequest(sz, 10)           // 10 is the minimum height
	img.SetObjectProperty("yalign", 1.0) // ???
	img.SetFromIconName("image-missing", 0)
	img.SetPixelSize(sz)
	img.Show()

	r.View.AddChildAtAnchor(img, anchor)
	// Ensure the end iterator is updated.
	r.end.ForwardToEnd()

	url := e.EmojiURL()
	cache.SetImageURLScaled(img, url, sz, sz)
}
