package extras

import (
	"path"
	"strconv"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/gotk4/pkg/gdk/v3"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/variables"
	"github.com/diamondburned/ningen/v2"
)

const EmbedMainCSS = ".embed { border-left: 4px solid #%06X; }"

var attachmentFormats = []string{".jpg", ".jpeg", ".png", ".webp", ".gif"}

func NewAttachment(msg *discord.Message) []gtk.Widgetter {
	// Discord's supported formats
	widgets := make([]gtk.Widgetter, 0, len(msg.Attachments))

	for _, att := range msg.Attachments {
		if att.Width == 0 || att.Height == 0 || !validExt(att.Proxy, attachmentFormats) {
			widgets = append(widgets, NewAnyAttachment(att.Filename, att.URL, att.Size))
			continue
		}

		w, h := maxSize(
			int(att.Width), int(att.Height),
			variables.EmbedMaxWidth, variables.EmbedImgHeight,
		)
		proxyURL := sizeToURL(att.Proxy, w, h)

		img := newExtraImage(proxyURL, att.URL, w, h)
		img.SetMarginStart(0)

		widgets = append(widgets, img)
	}

	return widgets
}

func NewEmbed(s *ningen.State, msg *discord.Message) []gtk.Widgetter {
	if len(msg.Embeds) == 0 {
		return nil
	}

	embeds := make([]gtk.Widgetter, 0, len(msg.Embeds))

	for _, embed := range msg.Embeds {
		w := newEmbed(s, msg, embed)
		if w == nil {
			continue
		}

		embeds = append(embeds, w)
	}

	return embeds
}

func newEmbed(s *ningen.State, msg *discord.Message, embed discord.Embed) gtk.Widgetter {
	switch embed.Type {
	case discord.NormalEmbed, discord.LinkEmbed, discord.ArticleEmbed:
		return newNormalEmbed(s, msg, embed)
	case discord.ImageEmbed:
		return newImageEmbed(embed)
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

		return newNormalEmbed(s, msg, embed)
	}

	// spew.Dump(embed)
	return nil
}

func newImageEmbed(embed discord.Embed) gtk.Widgetter {
	if embed.Thumbnail == nil {
		return nil
	}

	w, h := int(embed.Thumbnail.Width), int(embed.Thumbnail.Height)
	w, h = maxSize(w, h, variables.EmbedMaxWidth, variables.EmbedImgHeight)
	link := sizeToURL(embed.Thumbnail.Proxy, w, h)

	img := newExtraImage(link, embed.Thumbnail.URL, w, h)
	img.SetMarginStart(0)
	return img
}

func newExtraImage(proxy, url string, w, h int) *gtk.EventBox {
	img := gtk.NewImage()
	img.Show()
	img.SetVAlign(gtk.AlignStart)
	img.SetHAlign(gtk.AlignStart)

	evb := gtk.NewEventBox()
	evb.Show()
	evb.Add(img)
	evb.SetHAlign(gtk.AlignStart)
	evb.Connect("button-release-event", func(_ *gtk.EventBox, ev *gdk.Event) {
		if !gtkutils.EventIsLeftClick(ev) {
			return
		}
		SpawnPreviewDialog(proxy, url)
	})
	embedSetMargin(evb)

	cache.SetImageStreamed(img, proxy, w, h)
	return evb
}

func embedSetMargin(widget gtk.Widgetter) {
	w := gtk.BaseWidget(widget)
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
