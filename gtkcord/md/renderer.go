package md

import (
	"io"

	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/diamondburned/ningen/md"
	"github.com/gotk3/gotk3/gtk"
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

	tags TagState
}

func NewRenderer(tv *gtk.TextView) *Renderer {
	buf, _ := tv.GetBuffer()
	tags, _ := buf.GetTagTable()

	return &Renderer{
		View:   tv,
		Buffer: buf,
		tags: TagState{
			table: tags,
		},
	}
}

func (r *Renderer) Render(_ io.Writer, source []byte, n ast.Node) error {
	r.Buffer.Delete(r.Buffer.GetStartIter(), r.Buffer.GetEndIter())

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

func (r *Renderer) insertWithTag(content []byte, tag *gtk.TextTag) {
	if tag == nil {
		tag = r.tags.tag
	}

	r.Buffer.InsertWithTag(r.Buffer.GetEndIter(), string(content), tag)
}

func (r *Renderer) insertEmoji(e *md.Emoji) {
	// TODO
	var sz = InlineEmojiSize
	if e.Large {
		sz = LargeEmojiSize
	}

	anchor := r.Buffer.CreateChildAnchor(r.Buffer.GetEndIter())

	img, _ := gtk.ImageNew()
	img.Show()
	img.SetTooltipText(e.Name)
	img.SetSizeRequest(sz, 10) // 10 is the minimum height
	img.SetProperty("yalign", 1.0)
	gtkutils.ImageSetIcon(img, "image-missing", sz)

	r.View.AddChildAtAnchor(img, anchor)

	url := e.EmojiURL()

	go func() {
		if err := cache.SetImageScaled(url, img, sz, sz); err != nil {
			log.Errorln("Markdown: Failed to GET "+url+":", err)
		}
	}()
}
