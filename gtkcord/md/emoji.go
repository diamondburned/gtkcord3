package md

import (
	"github.com/diamondburned/gtkcord3/httpcache"
	"github.com/gotk3/gotk3/gdk"
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

func NewPixbuf(large bool, url string) (*gdk.Pixbuf, error) {
	b, err := httpcache.HTTPGet(url + "?size=64")
	if err != nil {
		return nil, errors.Wrap(err, "Failed to GET "+url)
	}

	l, err := gdk.PixbufLoaderNew()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create a new pixbuf loader")
	}

	if large {
		l.SetSize(LargeSize, LargeSize)
	} else {
		l.SetSize(InlineSize, InlineSize)
	}

	if _, err := l.Write(b); err != nil {
		return nil, errors.Wrap(err, "Failed to set image to pixbuf")
	}

	p, err := l.GetPixbuf()
	return p, errors.Wrap(err, "Failed to create pixbuf")
}
