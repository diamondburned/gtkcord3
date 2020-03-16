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
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/humanize"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

const (
	EmbedAvatarSize = 24
	EmbedMaxWidth   = 300
	EmbedImgHeight  = 300 // max
	EmbedMargin     = 8

	EmbedMainCSS = `
		.embed {
			border-left: 4px solid #%06X;
			background-color: rgba(0, 0, 0, 0.1);
		}
	`
)

func newExtraImage(proxy, direct string, w, h int, pp ...cache.Processor) gtkutils.ExtendedWidget {
	var img *gtk.Image
	var evb *gtk.EventBox

	semaphore.IdleMust(func() {
		img, _ = gtk.ImageNew()
		img.SetVAlign(gtk.ALIGN_START)
		img.SetHAlign(gtk.ALIGN_START)

		evb, _ = gtk.EventBoxNew()
		evb.Add(img)
		evb.Connect("button-release-event", func(_ *gtk.EventBox, ev *gdk.Event) {
			if !gtkutils.EventIsLeftClick(ev) {
				return
			}
			SpawnPreviewDialog(proxy, direct)
		})
		embedSetMargin(evb)
	})

	cache.AsyncFetch(proxy, img, w, h, pp...)
	return evb
}

func maxSize(w, h, maxW, maxH int) (int, int) {
	if w == 0 || h == 0 {
		// shit
		return maxW, maxH
	}

	if w > h {
		h = h * maxW / w
		w = maxW
	} else {
		w = w * maxH / h
		h = maxH
	}

	return w, h
}

// https://stackoverflow.com/questions/3008772/how-to-smart-resize-a-displayed-image-to-original-aspect-ratio
func sizeToURL(url string, w, h int) string {
	return url + "?width=" + strconv.Itoa(w) + "&height=" + strconv.Itoa(h)
}

func NewAttachment(msg *discord.Message) []gtkutils.ExtendedWidget {
	// Discord's supported formats
	var formats = []string{".jpg", ".jpeg", ".png", ".webp", ".gif"}
	var widgets = make([]gtkutils.ExtendedWidget, 0, len(msg.Attachments))

	for _, att := range msg.Attachments {
		if att.Width == 0 || att.Height == 0 {
			continue
		}

		if !validExt(att.Proxy, formats) {
			continue
		}

		w, h := maxSize(int(att.Width), int(att.Height), EmbedMaxWidth, EmbedImgHeight)
		proxyURL := sizeToURL(att.Proxy, w, h)

		img := newExtraImage(proxyURL, att.URL, 0, 0)
		if img, ok := img.(gtkutils.Marginator); ok {
			semaphore.IdleMust(img.SetMarginStart, 0)
		}

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

func NewEmbed(s *ningen.State, msg *discord.Message) []gtkutils.ExtendedWidget {
	if len(msg.Embeds) == 0 {
		return nil
	}

	var embeds = make([]gtkutils.ExtendedWidget, 0, len(msg.Embeds))

	for _, embed := range msg.Embeds {
		w := newEmbed(s, msg, embed)
		if w == nil {
			continue
		}

		embeds = append(embeds, w)
	}

	return embeds
}

func newEmbed(s *ningen.State, msg *discord.Message, embed discord.Embed) gtkutils.ExtendedWidget {
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

	spew.Dump(embed)
	return nil
}

func newImageEmbed(embed discord.Embed) gtkutils.ExtendedWidget {
	w, h := int(embed.Thumbnail.Width), int(embed.Thumbnail.Height)
	w, h = maxSize(w, h, EmbedMaxWidth, EmbedImgHeight)

	img := newExtraImage(embed.Thumbnail.Proxy, embed.Thumbnail.URL, w, h)
	if img, ok := img.(gtkutils.Marginator); ok {
		semaphore.IdleMust(img.SetMarginStart, 0)
	}
	return img
}

func newNormalEmbed(
	s *ningen.State, msg *discord.Message, embed discord.Embed) gtkutils.ExtendedWidget {

	main := semaphore.IdleMust(gtk.BoxNew, gtk.ORIENTATION_VERTICAL, 0).(*gtk.Box)
	semaphore.IdleMust(main.SetHAlign, gtk.ALIGN_START)

	if embed.Author != nil {
		box := semaphore.IdleMust(gtk.BoxNew, gtk.ORIENTATION_HORIZONTAL, 0).(*gtk.Box)
		semaphore.IdleMust(embedSetMargin, box)

		if embed.Author.ProxyIcon != "" {
			img := semaphore.IdleMust(gtk.ImageNew).(*gtk.Image)
			semaphore.IdleMust(img.SetMarginEnd, EmbedMargin)
			asyncFetch(embed.Author.ProxyIcon, img, 24, 24, cache.Round)

			semaphore.IdleMust(box.Add, img)
		}

		if embed.Author.Name != "" {
			author := semaphore.IdleMust(gtk.LabelNew, embed.Author.Name).(*gtk.Label)
			semaphore.IdleMust(author.SetLineWrap, true)
			semaphore.IdleMust(author.SetLineWrapMode, pango.WRAP_WORD_CHAR)
			semaphore.IdleMust(author.SetXAlign, float64(0.0))

			if embed.Author.URL != "" {
				semaphore.IdleMust(author.SetMarkup, fmt.Sprintf(
					`<a href="%s">%s</a>`,
					html.EscapeString(embed.Author.URL), html.EscapeString(embed.Author.Name),
				))
			}

			semaphore.IdleMust(box.Add, author)
		}

		semaphore.IdleMust(main.Add, box)
	}

	if embed.Title != "" {
		var title = `<span weight="heavy">` + html.EscapeString(embed.Title) + `</span>`
		if embed.URL != "" {
			title = fmt.Sprintf(`<a href="%s">%s</a>`, html.EscapeString(embed.URL), title)
		}

		semaphore.IdleMust(func() {
			label, _ := gtk.LabelNew("")
			label.SetMarkup(title)
			label.SetLineWrap(true)
			label.SetLineWrapMode(pango.WRAP_WORD_CHAR)
			label.SetXAlign(0.0)
			embedSetMargin(label)

			main.Add(label)
		})
	}

	if embed.Description != "" {
		txtb := semaphore.IdleMust(gtk.TextBufferNew, (*gtk.TextTagTable)(nil)).(*gtk.TextBuffer)
		md.ParseMessage(s, msg, []byte(embed.Description), txtb)

		semaphore.IdleMust(func() {
			txtv, _ := gtk.TextViewNew()
			txtv.SetBuffer(txtb)
			txtv.SetCursorVisible(false)
			txtv.SetEditable(false)
			txtv.SetWrapMode(gtk.WRAP_WORD_CHAR)
			txtv.SetSizeRequest(EmbedMaxWidth, -1)
			embedSetMargin(txtv)

			main.Add(txtv)
		})
	}

	if len(embed.Fields) > 0 {
		semaphore.IdleMust(func() {
			var fields *gtk.Grid

			fields, _ = gtk.GridNew()
			embedSetMargin(fields)
			fields.SetRowSpacing(uint(7))
			fields.SetColumnSpacing(uint(14))

			main.Add(fields)

			col, row := 0, 0

			for _, field := range embed.Fields {
				text, _ := gtk.LabelNew("")
				text.SetLineWrap(true)
				text.SetLineWrapMode(pango.WRAP_WORD_CHAR)
				text.SetXAlign(float64(0.0))
				text.SetMarkup(fmt.Sprintf(
					`<span weight="heavy">%s</span>`+"\n"+`<span weight="light">%s</span>`,
					field.Name, field.Value,
				))

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
		})
	}

	if embed.Footer != nil || embed.Timestamp.Valid() {
		footer := semaphore.IdleMust(gtk.BoxNew, gtk.ORIENTATION_HORIZONTAL, 0).(*gtk.Box)
		semaphore.IdleMust(embedSetMargin, footer)

		if embed.Footer != nil {
			if embed.Footer.ProxyIcon != "" {
				img := semaphore.IdleMust(gtk.ImageNew).(*gtk.Image)
				semaphore.IdleMust(img.SetMarginEnd, EmbedMargin)
				asyncFetch(embed.Footer.ProxyIcon, img, 24, 24, cache.Round)

				semaphore.IdleMust(footer.Add, img)
			}

			if embed.Footer.Text != "" {
				text := semaphore.IdleMust(gtk.LabelNew, embed.Footer.Text).(*gtk.Label)
				semaphore.IdleMust(text.SetOpacity, 0.65)
				semaphore.IdleMust(text.SetLineWrap, true)
				semaphore.IdleMust(text.SetLineWrapMode, pango.WRAP_WORD_CHAR)
				semaphore.IdleMust(text.SetXAlign, float64(0.0))

				semaphore.IdleMust(footer.Add, text)
			}
		}

		if embed.Timestamp.Valid() {
			time := humanize.TimeAgo(embed.Timestamp.Time())
			text := semaphore.IdleMust(gtk.LabelNew, time).(*gtk.Label)
			if embed.Footer != nil {
				semaphore.IdleMust(text.SetText, " - "+time)
			}

			semaphore.IdleMust(footer.Add, text)
		}

		semaphore.IdleMust(main.Add, footer)
	}

	if embed.Thumbnail != nil {
		wrapper := semaphore.IdleMust(gtk.BoxNew, gtk.ORIENTATION_HORIZONTAL, 0).(*gtk.Box)
		semaphore.IdleMust(wrapper.Add, main)

		// Do a shitty hack:
		main = wrapper
		semaphore.IdleMust(main.SetHAlign, gtk.ALIGN_START)

		w, h := int(embed.Thumbnail.Width), int(embed.Thumbnail.Height)
		w, h = maxSize(w, h, 80, 80)

		semaphore.IdleMust(wrapper.Add, newExtraImage(
			sizeToURL(embed.Thumbnail.Proxy, w, h),
			embed.Thumbnail.URL, 0, 0,
		))
	}

	if embed.Image != nil {
		wrapper := semaphore.IdleMust(gtk.BoxNew, gtk.ORIENTATION_VERTICAL, 0).(*gtk.Box)
		semaphore.IdleMust(wrapper.Add, main)

		// Do a shitty hack again:
		main = wrapper
		semaphore.IdleMust(main.SetHAlign, gtk.ALIGN_START)

		w, h := int(embed.Image.Width), int(embed.Image.Height)
		w, h = maxSize(w, h, EmbedMaxWidth, EmbedImgHeight)

		semaphore.IdleMust(wrapper.Add, newExtraImage(
			sizeToURL(embed.Image.Proxy, w, h),
			embed.Image.URL, 0, 0,
		))
	}

	gtkutils.InjectCSS(main, "embed", fmt.Sprintf(EmbedMainCSS, embed.Color))

	return main
}

func embedSetMargin(w gtkutils.Marginator) {
	w.SetMarginStart(EmbedMargin * 2)
	w.SetMarginEnd(EmbedMargin * 2)
	w.SetMarginTop(EmbedMargin / 2)
	w.SetMarginBottom(EmbedMargin / 2)
}

func asyncFetch(url string, img *gtk.Image, w, h int, pp ...cache.Processor) {
	cache.AsyncFetch(url, img, w, h, pp...)
}
