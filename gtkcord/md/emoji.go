package md

import (
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
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

	emojiTag := s.p.InlineEmojiTag()

	go func() {
		defer s.iterWg.Done()

		pixbuf, err := cache.GetPixbufScaled(url+"?size=64", 0, 0, cache.Resize(sz, sz))
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
