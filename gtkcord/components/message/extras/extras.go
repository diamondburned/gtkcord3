package extras

import (
	"path"
	"strconv"

	"github.com/davecgh/go-spew/spew"
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/gtkcord/variables"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
)

const EmbedMainCSS = ".embed { border-left: 4px solid #%06X; }"

func NewAttachmentUnsafe(msg *discord.Message) []gtk.IWidget {
	// Discord's supported formats
	var formats = []string{".jpg", ".jpeg", ".png", ".webp", ".gif"}
	var widgets = make([]gtk.IWidget, 0, len(msg.Attachments))

	for _, att := range msg.Attachments {
		if att.Width == 0 || att.Height == 0 || !validExt(att.Proxy, formats) {
			widgets = append(widgets, NewAnyAttachmentUnsafe(att.Filename, att.URL, att.Size))
			continue
		}

		w, h := maxSize(
			int(att.Width), int(att.Height),
			variables.EmbedMaxWidth, variables.EmbedImgHeight,
		)
		proxyURL := sizeToURL(att.Proxy, w, h)

		img := newExtraImageUnsafe(proxyURL, att.URL, w, h)
		img.SetMarginStart(0)

		widgets = append(widgets, img)
	}

	return widgets
}

func NewEmbedUnsafe(s *ningen.State, msg *discord.Message) []gtk.IWidget {
	if len(msg.Embeds) == 0 {
		return nil
	}

	var embeds = make([]gtk.IWidget, 0, len(msg.Embeds))

	for _, embed := range msg.Embeds {
		w := newEmbedUnsafe(s, msg, embed)
		if w == nil {
			continue
		}

		embeds = append(embeds, w)
	}

	return embeds
}

func newEmbedUnsafe(s *ningen.State, msg *discord.Message, embed discord.Embed) gtk.IWidget {
	switch embed.Type {
	case discord.NormalEmbed, discord.LinkEmbed, discord.ArticleEmbed:
		return newNormalEmbedUnsafe(s, msg, embed)
	case discord.ImageEmbed:
		return newImageEmbedUnsafe(embed)
	case discord.VideoEmbed:
		// I'm tired and lazy.
		if embed.Thumbnail != nil && embed.Image == nil {
			img := embed.Thumbnail
			embed.Image = &discord.EmbedImage{
				URL:    img.URL,
				Proxy:  img.Proxy,
				Width:  img.Width,
				Height: img.Height,
			}
			embed.Thumbnail = nil
		}

		return newNormalEmbedUnsafe(s, msg, embed)
	}

	spew.Dump(embed)
	return nil
}

func newImageEmbedUnsafe(embed discord.Embed) gtkutils.ExtendedWidget {
	if embed.Thumbnail == nil {
		return nil
	}

	w, h := int(embed.Thumbnail.Width), int(embed.Thumbnail.Height)
	w, h = maxSize(w, h, variables.EmbedMaxWidth, variables.EmbedImgHeight)
	link := sizeToURL(embed.Thumbnail.Proxy, w, h)

	img := newExtraImageUnsafe(link, embed.Thumbnail.URL, w, h)
	img.SetMarginStart(0)
	return img
}

func newExtraImageUnsafe(proxy, url string, w, h int, pp ...cache.Processor) *gtk.EventBox {
	img, _ := gtk.ImageNew()
	img.Show()
	img.SetVAlign(gtk.ALIGN_START)
	img.SetHAlign(gtk.ALIGN_START)

	evb, _ := gtk.EventBoxNew()
	evb.Show()
	evb.Add(img)
	evb.SetHAlign(gtk.ALIGN_START)
	evb.Connect("button-release-event", func(_ *gtk.EventBox, ev *gdk.Event) {
		if !gtkutils.EventIsLeftClick(ev) {
			return
		}
		SpawnPreviewDialog(proxy, url)
	})
	embedSetMargin(evb)

	cache.AsyncFetchUnsafe(proxy, img, w, h, pp...)
	return evb
}

func embedSetMargin(w gtkutils.Marginator) {
	w.SetMarginStart(variables.EmbedMargin)
	w.SetMarginEnd(variables.EmbedMargin)
	w.SetMarginBottom(variables.EmbedMargin / 2)
	// w.SetMarginTop(variables.EmbedMargin / 2)
}

func validExt(url string, exts []string) bool {
	var ext = path.Ext(url)

	for _, e := range exts {
		if e == ext {
			return true
		}
	}

	return false
}

func maxSize(w, h, maxW, maxH int) (int, int) {
	// cap width
	maxW = clampWidth(maxW)

	if w == 0 || h == 0 {
		// shit
		return maxW, maxH
	}

	return cache.MaxSize(w, h, maxW, maxH)
}

func sizeToURL(url string, w, h int) string {
	return url + "?width=" + strconv.Itoa(w) + "&height=" + strconv.Itoa(h)
}

func clampWidth(width int) int {
	max := variables.MaxMessageWidth
	max -= variables.AvatarSize + variables.AvatarPadding*2
	max -= variables.EmbedMargin * 3

	if width > max {
		width = max
	}
	return width
}
