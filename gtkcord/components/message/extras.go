package message

import (
	"fmt"
	"html"
	"path"
	"strconv"

	"github.com/davecgh/go-spew/spew"
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/md"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/internal/humanize"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

const (
	EmbedAvatarSize = 24
	EmbedMaxWidth   = 300
	EmbedImgHeight  = 300 // max
	EmbedMargin     = 8

	EmbedMainCSS = ".embed { border-left: 4px solid #%06X; }"
)

func clampWidth(width int) int {
	if max := MaxMessageWidth - (AvatarSize + AvatarPadding*2) - (EmbedMargin * 3); width > max {
		width = max
	}
	return width
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

func NewAttachmentUnsafe(msg *discord.Message) []gtk.IWidget {
	// Discord's supported formats
	var formats = []string{".jpg", ".jpeg", ".png", ".webp", ".gif"}
	var widgets = make([]gtk.IWidget, 0, len(msg.Attachments))

	for _, att := range msg.Attachments {
		if att.Width == 0 || att.Height == 0 {
			continue
		}

		if !validExt(att.Proxy, formats) {
			continue
		}

		w, h := maxSize(int(att.Width), int(att.Height), EmbedMaxWidth, EmbedImgHeight)
		proxyURL := sizeToURL(att.Proxy, w, h)

		img := newExtraImageUnsafe(proxyURL, att.URL, w, h)
		img.SetMarginStart(0)

		widgets = append(widgets, img)
	}

	return widgets
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
	w, h = maxSize(w, h, EmbedMaxWidth, EmbedImgHeight)
	link := sizeToURL(embed.Thumbnail.Proxy, w, h)

	img := newExtraImageUnsafe(link, embed.Thumbnail.URL, w, h)
	img.SetMarginStart(0)
	return img
}

func newNormalEmbedUnsafe(
	s *ningen.State, msg *discord.Message, embed discord.Embed) gtkutils.ExtendedWidget {

	content, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	content.SetHAlign(gtk.ALIGN_START)

	widthHint := 0 // used for calculating requested embed width

	if embed.Author != nil {
		box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
		embedSetMargin(box)

		if embed.Author.ProxyIcon != "" {
			img, _ := gtk.ImageNew()
			img.SetMarginEnd(EmbedMargin)
			cache.AsyncFetchUnsafe(embed.Author.ProxyIcon, img, 24, 24, cache.Round)

			box.Add(img)
		}

		if embed.Author.Name != "" {
			author, _ := gtk.LabelNew(embed.Author.Name)
			author.SetUseMarkup(true)
			author.SetLineWrap(true)
			author.SetLineWrapMode(pango.WRAP_WORD_CHAR)
			author.SetXAlign(0.0)

			if embed.Author.URL != "" {
				author.SetMarkup(fmt.Sprintf(
					`<a href="%s">%s</a>`,
					html.EscapeString(embed.Author.URL), html.EscapeString(embed.Author.Name),
				))
			}

			box.Add(author)
		}

		content.Add(box)
	}

	if embed.Title != "" {
		var title = `<span weight="heavy">` + html.EscapeString(embed.Title) + `</span>`
		if embed.URL != "" {
			title = fmt.Sprintf(`<a href="%s">%s</a>`, html.EscapeString(embed.URL), title)
		}

		label, _ := gtk.LabelNew("")
		label.SetMarkup(title)
		label.SetLineWrap(true)
		label.SetLineWrapMode(pango.WRAP_WORD_CHAR)
		label.SetXAlign(0.0)
		embedSetMargin(label)

		content.Add(label)
	}

	if embed.Description != "" {
		txtv, _ := gtk.TextViewNew()
		txtv.SetCursorVisible(false)
		txtv.SetEditable(false)
		txtv.SetWrapMode(gtk.WRAP_WORD_CHAR)

		embedSetMargin(txtv)
		md.ParseWithMessage([]byte(embed.Description), txtv, s.Store, msg)

		// Make text smaller:
		md.WrapTag(txtv, map[string]interface{}{
			"scale":     0.84,
			"scale-set": true,
		})

		content.Add(txtv)
	}

	if len(embed.Fields) > 0 {
		var fields *gtk.Grid

		fields, _ = gtk.GridNew()
		embedSetMargin(fields)
		fields.SetRowSpacing(uint(7))
		fields.SetColumnSpacing(uint(14))

		content.Add(fields)

		col, row := 0, 0

		for _, field := range embed.Fields {
			text, _ := gtk.LabelNew("")
			text.SetEllipsize(pango.ELLIPSIZE_END)
			text.SetXAlign(float64(0.0))
			text.SetMarkup(fmt.Sprintf(
				`<span weight="heavy">%s</span>`+"\n"+`<span weight="light">%s</span>`,
				field.Name, field.Value,
			))
			text.SetTooltipText(field.Name + "\n" + field.Value)

			// I have no idea what this does. It')s just improvised.
			if field.Inline && col < 3 {
				fields.Attach(text, col, row, 1, 1)
				col++

			} else {
				if col > 0 {
					row++
				}

				col = 0
				fields.Attach(text, col, row, 1, 1)

				if !field.Inline {
					row++
				} else {
					col++
				}
			}
		}
	}

	if embed.Footer != nil || embed.Timestamp.Valid() {
		footer, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
		embedSetMargin(footer)

		if embed.Footer != nil {
			if embed.Footer.ProxyIcon != "" {
				img, _ := gtk.ImageNew()
				img.SetMarginEnd(EmbedMargin)
				cache.AsyncFetchUnsafe(embed.Footer.ProxyIcon, img, 24, 24, cache.Round)

				footer.Add(img)
			}

			if embed.Footer.Text != "" {
				text, _ := gtk.LabelNew(embed.Footer.Text)
				text.SetVAlign(gtk.ALIGN_START)
				text.SetOpacity(0.65)
				text.SetLineWrap(true)
				text.SetLineWrapMode(pango.WRAP_WORD_CHAR)
				text.SetXAlign(0.0)

				footer.Add(text)
			}
		}

		if embed.Timestamp.Valid() {
			time := humanize.TimeAgo(embed.Timestamp.Time())

			var text, _ = gtk.LabelNew(time)
			if embed.Footer != nil {
				text.SetText(" - " + time)
			}

			footer.Add(text)
		}

		content.Add(footer)
	}

	if embed.Thumbnail != nil {
		wrapper, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
		wrapper.Add(content)

		// Do a shitty hack:
		content = wrapper
		content.SetHAlign(gtk.ALIGN_START)

		w, h := int(embed.Thumbnail.Width), int(embed.Thumbnail.Height)
		w, h = maxSize(w, h, 80, 80)

		wrapper.Add(newExtraImageUnsafe(
			sizeToURL(embed.Thumbnail.Proxy, w, h),
			embed.Thumbnail.URL, 0, 0,
		))
	}

	if embed.Image != nil {
		wrapper, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
		wrapper.Add(content)

		// Do a shitty hack again:
		content = wrapper
		content.SetHAlign(gtk.ALIGN_START)

		w, h := int(embed.Image.Width), int(embed.Image.Height)
		w, h = maxSize(w, h, EmbedMaxWidth, EmbedImgHeight)

		// set width hint to resize embeds accordingly
		widthHint = w

		wrapper.Add(newExtraImageUnsafe(
			sizeToURL(embed.Image.Proxy, w, h),
			embed.Image.URL, w, h,
		))
	}

	// Calculate the embed width without padding:
	var w = clampWidth(EmbedMaxWidth)
	if widthHint > 0 && w > widthHint {
		w = widthHint
	}

	// Wrap the content inside another box:
	main, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	main.SetHAlign(gtk.ALIGN_START)
	main.SetSizeRequest(w+(EmbedMargin*2), 0)
	main.Add(content)

	// Apply margin to content:
	content.SetMarginTop(EmbedMargin) // account for children not having margin top
	content.SetMarginBottom(EmbedMargin / 2)

	// Add a frame around the main embed:
	gtkutils.InjectCSSUnsafe(main, "embed", fmt.Sprintf(EmbedMainCSS, embed.Color))

	return main
}

func embedSetMargin(w gtkutils.Marginator) {
	w.SetMarginStart(EmbedMargin)
	w.SetMarginEnd(EmbedMargin)
	w.SetMarginBottom(EmbedMargin / 2)
	// w.SetMarginTop(EmbedMargin / 2)
}
