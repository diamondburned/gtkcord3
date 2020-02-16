package md

import (
	"github.com/diamondburned/gtkcord3/gtkcord/pbpool"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/gotk3/gotk3/gtk"
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

	iter := semaphore.IdleMust(s.buf.GetEndIter).(*gtk.TextIter)

	i := s.p.GetIcon("image-missing", sz)
	if i == nil {
		semaphore.IdleMust(s.buf.Insert, iter, "[?]")
		log.Errorln("Markdown: Failed to get image-missing icon")
		return
	}

	// Preserve position:
	lastIndex := semaphore.IdleMust(iter.GetLineIndex).(int)
	lastLine := semaphore.IdleMust(iter.GetLine).(int)

	// Insert Pixbuf after s.prev:
	semaphore.IdleMust(s.buf.InsertPixbuf, iter, i)

	// Add to the waitgroup, so we know when to put the state back.
	s.iterWg.Add(1)

	go func() {
		defer s.iterWg.Done()

		pixbuf, err := pbpool.GetScaled(url+"?size=64", sz, sz)
		if err != nil {
			log.Errorln("Markdown: Failed to GET " + url)
			return
		}

		s.iterMu.Lock()
		defer s.iterMu.Unlock()

		// Try and replace the last inserted pixbuf with ours:
		last := semaphore.IdleMust(s.buf.GetIterAtLineIndex, lastLine, lastIndex).(*gtk.TextIter)
		fwdi := semaphore.IdleMust(s.buf.GetIterAtLineIndex, lastLine, lastIndex).(*gtk.TextIter)
		semaphore.IdleMust(fwdi.ForwardChar)

		semaphore.IdleMust(s.buf.Delete, last, fwdi)
		semaphore.IdleMust(s.buf.InsertPixbuf, last, pixbuf)
	}()
}
