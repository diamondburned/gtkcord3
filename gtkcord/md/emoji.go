package md

import (
	"log"

	"github.com/diamondburned/gtkcord3/httpcache"
	"github.com/gotk3/gotk3/gdk"
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

func (s *mdState) InsertAsyncPixbuf(buf *gtk.TextBuffer, url string) error {
	var sz = InlineSize
	if !s.hasText {
		sz = LargeSize
	}

	i, err := s.p.theme.LoadIcon(
		"user-available-symbolic",
		sz, gtk.IconLookupFlags(gtk.ICON_LOOKUP_FORCE_SIZE))
	if err != nil {
		return errors.Wrap(err, "Failed to get user-available-symbolic icon")
	}

	// Lock the iter mutex:
	s.iterMu.Lock()
	defer s.iterMu.Unlock()

	iter := buf.GetEndIter()

	// Pre-insert s.prev:
	buf.InsertMarkup(iter, string(escape(s.prev)))

	// Preserve position:
	lastIndex := iter.GetLineIndex()
	lastLine := iter.GetLine()

	// Insert Pixbuf after s.prev:
	buf.InsertPixbuf(iter, i)

	log.Println("Inserted icon at", lastIndex, lastLine)

	// Clear so the buffers don't get added again:
	s.chunk = s.chunk[:0]
	s.prev = s.prev[:0]

	// Add to the waitgroup, so we know when to put the state back.
	s.iterWg.Add(1)

	go func() {
		defer s.iterWg.Done()

		b, err := httpcache.HTTPGet(url + "?size=64")
		if err != nil {
			s.p.Error(errors.Wrap(err, "Failed to GET "+url))
			return
		}

		l, err := gdk.PixbufLoaderNew()
		if err != nil {
			s.p.Error(errors.Wrap(err, "Failed to create a new pixbuf loader"))
			return
		}

		l.SetSize(sz, sz)

		if _, err := l.Write(b); err != nil {
			s.p.Error(errors.Wrap(err, "Failed to set image to pixbuf"))
			return
		}

		pixbuf, err := l.GetPixbuf()
		if err != nil {
			s.p.Error(errors.Wrap(err, "Failed to create pixbuf"))
			return
		}

		// Try and replace the last inserted pixbuf with ours:
		glib.IdleAdd(func() {
			s.iterMu.Lock()
			defer s.iterMu.Unlock()

			lastIter := buf.GetIterAtLineIndex(lastLine, lastIndex)
			lastIterFwd := buf.GetIterAtLineIndex(lastLine, lastIndex)
			lastIterFwd.ForwardChar()

			buf.Delete(lastIter, lastIterFwd)
			buf.InsertPixbuf(lastIter, pixbuf)
		})
	}()

	return nil
}
