package md

import (
	"log"

	"github.com/diamondburned/gtkcord3/gtkcord/pbpool"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
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

	i, err := s.p.theme.LoadIcon("user-available-symbolic", sz, gtk.ICON_LOOKUP_FORCE_SIZE)
	if err != nil {
		s.buf.Insert(iter, "[broken emoji]")
		log.Println("Markdown: Failed to get user-available-symbolic icon:", err)
		return
	}

	// Preserve position:
	lastIndex := iter.GetLineIndex()
	lastLine := iter.GetLine()

	// Insert Pixbuf after s.prev:
	s.buf.InsertPixbuf(iter, i)

	// Add to the waitgroup, so we know when to put the state back.
	s.iterWg.Add(2)

	semaphore.Go(func() {
		defer s.iterWg.Done()

		pixbuf, err := pbpool.GetScaled(url+"?size=64", sz, sz)
		if err != nil {
			s.p.Error(errors.Wrap(err, "Failed to GET "+url))
			return
		}

		// Try and replace the last inserted pixbuf with ours:
		glib.IdleAdd(func(s *mdState) bool {
			s.iterMu.Lock()
			defer s.iterMu.Unlock()
			defer s.iterWg.Done()

			lastIter := s.buf.GetIterAtLineIndex(lastLine, lastIndex)
			lastIterFwd := s.buf.GetIterAtLineIndex(lastLine, lastIndex)
			lastIterFwd.ForwardChar()

			s.buf.Delete(lastIter, lastIterFwd)
			s.buf.InsertPixbuf(lastIter, pixbuf)

			return false
		}, s)
	})
}
