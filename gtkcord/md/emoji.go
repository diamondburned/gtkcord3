package md

import (
	"regexp"

	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/icons"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

const (
	InlineEmojiSize = 22
	LargeEmojiSize  = 48
)

func EmojiURL(emojiID string, animated bool) string {
	const EmojiBaseURL = "https://cdn.discordapp.com/emojis/"

	if animated {
		return EmojiBaseURL + emojiID + ".gif"
	}

	return EmojiBaseURL + emojiID + ".png"
}

type Emoji struct {
	ast.BaseInline

	ID   string
	Name string
	GIF  bool

	Large bool // TODO
}

var KindEmoji = ast.NewNodeKind("Emoji")

// Kind implements Node.Kind.
func (e *Emoji) Kind() ast.NodeKind {
	return KindEmoji
}

// Dump implements Node.Dump
func (e *Emoji) Dump(source []byte, level int) {
	ast.DumpHelper(e, source, level, nil, nil)
}

func (e Emoji) EmojiURL() string {
	return EmojiURL(string(e.ID), e.GIF)
}

type emoji struct{}

var emojiRegex = regexp.MustCompile(`<(a?):(.+?):(\d+)>`)

func (emoji) Trigger() []byte {
	// return []byte("http")
	return []byte{'<'}
}

func (emoji) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	match := matchInline(block, '<', '>')
	if match == nil {
		return nil
	}

	var matches = emojiRegex.FindSubmatch(match)
	if len(matches) != 4 {
		return nil
	}

	var emoji = &Emoji{
		BaseInline: ast.BaseInline{},

		GIF:  string(matches[1]) == "a",
		Name: string(matches[2]),
		ID:   string(matches[3]),
	}

	return emoji
}

func (s *TagState) inlineEmojiTag() *gtk.TextTag {
	t, err := s.table.Lookup("emoji")
	if err == nil {
		return t
	}

	t, err = gtk.TextTagNew("emoji")
	if err != nil {
		log.Panicln("Failed to create new emoji tag:", err)
	}

	t.SetProperty("rise", -8192)

	s.table.Add(t)
	return t
}

func (r *Renderer) insertEmoji(url string) {
	// TODO
	var sz = InlineEmojiSize
	// if !s.hasText {
	// 	sz = LargeEmojiSize
	// }

	iter := r.Buffer.GetEndIter()

	i := icons.GetIconUnsafe("image-missing", sz)
	if i == nil {
		e, err := gdk.PixbufNew(gdk.COLORSPACE_RGB, true, 8, sz, sz)
		if err != nil {
			r.Buffer.Insert(iter, "[?]")
			return
		}

		// set the empty pixbuf
		i = e
	}

	// Preserve position:
	lastIndex := iter.GetLineIndex()
	lastLine := iter.GetLine()

	// Insert Pixbuf after s.prev:
	r.Buffer.InsertPixbuf(iter, i)

	// Add to the waitgroup, so we know when to put the state back.
	r.setGroup.Add(1)

	go func() {
		defer r.setGroup.Done()

		pixbuf, err := cache.GetPixbufScaled(url+"?size=64", sz, sz)
		if err != nil {
			log.Errorln("Markdown: Failed to GET " + url)
			return
		}

		r.setEmojis <- setEmoji{
			pb:    pixbuf,
			line:  lastLine,
			index: lastIndex,
		}
	}()
}
