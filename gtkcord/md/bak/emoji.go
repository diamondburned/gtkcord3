package md

import (
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/icons"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/gotk3/gotk3/gdk"
)

func EmojiURL(emojiID string, animated bool) string {
	const EmojiBaseURL = "https://cdn.discordapp.com/emojis/"

	if animated {
		return EmojiBaseURL + emojiID + ".gif"
	}

	return EmojiBaseURL + emojiID + ".png"
}

var (
	InlineSize = 22
	LargeSize  = 48
)

func (s *mdState) InsertAsyncPixbuf(url string) {
	var sz = InlineSize
	if !s.hasText {
		sz = LargeSize
	}

	iter := s.buf.GetEndIter()

	i := icons.GetIconUnsafe("image-missing", sz)
	if i == nil {
		e, err := gdk.PixbufNew(gdk.COLORSPACE_RGB, true, 8, sz, sz)
		if err != nil {
			log.Errorln("Markdown: Failed to make placeholder pixbuf:", err)
			s.buf.Insert(iter, "[?]")
			return
		}
		// set the empty pixbuf
		i = e
	}

	// Preserve position:
	lastIndex := iter.GetLineIndex()
	lastLine := iter.GetLine()

	// Insert Pixbuf after s.prev:
	s.buf.InsertPixbuf(iter, i)

	// Add to the waitgroup, so we know when to put the state back.
	s.iterWg.Add(1)

	emojiTag := s.inlineEmojiTag()

	go func() {
		defer s.iterWg.Done()

		pixbuf, err := cache.GetPixbufScaled(url+"?size=64", sz, sz)
		if err != nil {
			log.Errorln("Markdown: Failed to GET " + url)
			return
		}

		s.iterMu.Lock()
		defer s.iterMu.Unlock()

		// Try and replace the last inserted pixbuf with ours:
		semaphore.IdleMust(func() {
			last := s.buf.GetIterAtLineIndex(lastLine, lastIndex)
			fwdi := s.buf.GetIterAtLineIndex(lastLine, lastIndex)
			fwdi.ForwardChar()

			s.buf.Delete(last, fwdi)
			s.buf.InsertPixbuf(last, pixbuf)

			first := s.buf.GetIterAtLineIndex(lastLine, lastIndex)
			s.buf.ApplyTag(emojiTag, first, last)
		})
	}()
}
