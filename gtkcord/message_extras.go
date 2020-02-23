package gtkcord

import (
	"fmt"
	"path"
	"strconv"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/humanize"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

const (
	EmbedAvatarSize = 24
	EmbedMaxWidth   = 450
	EmbedImgHeight  = 350 // max
	EmbedMargin     = 8

	EmbedMainCSS = `
		.embed {
			border-left: 4px solid #%06X;
			background-color: rgba(0, 0, 0, 0.1);
		}
	`
)

func newExtraImage(proxy, direct string, w, h int, pp ...cache.Processor) gtkutils.ExtendedWidget {
	img := must(gtk.ImageNew).(*gtk.Image)
	must(img.SetVAlign, gtk.ALIGN_START)
	must(img.SetHAlign, gtk.ALIGN_START)

	evb := must(gtk.EventBoxNew).(*gtk.EventBox)
	must(evb.Add, img)
	must(evb.Connect, "button-release-event", func() {
		SpawnPreviewDialog(proxy, direct)
	})
	must(embedSetMargin, evb)

	asyncFetch(proxy, img, w, h, pp...)

	return evb
}

func maxSize(w, h, maxW, maxH int) (int, int) {
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

func NewAttachment(msg discord.Message) []gtkutils.ExtendedWidget {
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
			must(img.SetMarginStart, 0)
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

func NewEmbed(msg discord.Message) []gtkutils.ExtendedWidget {
	if len(msg.Embeds) == 0 {
		return nil
	}

	var embeds = make([]gtkutils.ExtendedWidget, 0, len(msg.Embeds))

	for _, embed := range msg.Embeds {
		w := newEmbed(msg, embed)
		if w == nil {
			continue
		}

		embeds = append(embeds, w)
	}

	return embeds
}

func newEmbed(msg discord.Message, embed discord.Embed) gtkutils.ExtendedWidget {
	switch embed.Type {
	case discord.NormalEmbed, discord.LinkEmbed:
		return newNormalEmbed(msg, embed)
	case discord.ImageEmbed:
		return newImageEmbed(embed)
	case discord.VideoEmbed:
		// Unsupported
		return nil
	}

	return nil
}

func newImageEmbed(embed discord.Embed) gtkutils.ExtendedWidget {
	w, h := int(embed.Thumbnail.Width), int(embed.Thumbnail.Height)
	w, h = maxSize(w, h, EmbedMaxWidth, EmbedImgHeight)

	img := newExtraImage(embed.Thumbnail.Proxy, embed.Thumbnail.URL, w, h)
	if img, ok := img.(gtkutils.Marginator); ok {
		must(img.SetMarginStart, 0)
	}
	return img
}

func newNormalEmbed(msg discord.Message, embed discord.Embed) gtkutils.ExtendedWidget {
	main := must(gtk.BoxNew, gtk.ORIENTATION_VERTICAL, 0).(*gtk.Box)
	must(main.SetHAlign, gtk.ALIGN_START)

	if embed.Author != nil {
		box := must(gtk.BoxNew, gtk.ORIENTATION_HORIZONTAL, 0).(*gtk.Box)
		must(embedSetMargin, box)

		if embed.Author.ProxyIcon != "" {
			img := must(gtk.ImageNew).(*gtk.Image)
			must(img.SetMarginEnd, EmbedMargin)
			asyncFetch(embed.Author.ProxyIcon, img, 24, 24, cache.Round)

			must(box.Add, img)
		}

		if embed.Author.Name != "" {
			author := must(gtk.LabelNew, embed.Author.Name).(*gtk.Label)
			must(author.SetLineWrap, true)
			must(author.SetLineWrapMode, pango.WRAP_WORD_CHAR)
			must(author.SetXAlign, float64(0.0))

			if embed.Author.URL != "" {
				must(author.SetMarkup, fmt.Sprintf(
					`<a href="%s">%s</a>`,
					embed.Author.URL, escape(embed.Author.Name),
				))
			}

			must(box.Add, author)
		}

		must(main.Add, box)
	}

	if embed.Title != "" {
		var title = `<span weight="heavy">` + escape(embed.Title) + `</span>`
		if embed.URL != "" {
			title = fmt.Sprintf(`<a href="%s">%s</a>`, embed.URL, title)
		}

		must(func() {
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
		desc := must(App.parser.NewTextBuffer).(*gtk.TextBuffer)
		App.parser.ParseMessage(&msg, []byte(embed.Description), desc)

		must(func() {
			txtv, _ := gtk.TextViewNewWithBuffer(desc)
			txtv.SetCursorVisible(false)
			txtv.SetEditable(false)
			txtv.SetWrapMode(gtk.WRAP_WORD_CHAR)
			txtv.SetSizeRequest(EmbedMaxWidth, -1)
			embedSetMargin(txtv)

			main.Add(txtv)
		})
	}

	if len(embed.Fields) > 0 {
		fields := must(gtk.GridNew).(*gtk.Grid)
		must(embedSetMargin, fields)
		must(fields.SetRowSpacing, uint(7))
		must(fields.SetColumnSpacing, uint(14))
		must(main.Add, fields)

		col, row := 0, 0

		for _, field := range embed.Fields {
			text := must(gtk.LabelNew, "").(*gtk.Label)
			must(text.SetLineWrap, true)
			must(text.SetLineWrapMode, pango.WRAP_WORD_CHAR)
			must(text.SetXAlign, float64(0.0))
			must(text.SetMarkup, fmt.Sprintf(
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
	}

	if embed.Footer != nil || embed.Timestamp.Valid() {
		footer := must(gtk.BoxNew, gtk.ORIENTATION_HORIZONTAL, 0).(*gtk.Box)
		must(embedSetMargin, footer)

		if embed.Footer != nil {
			if embed.Footer.ProxyIcon != "" {
				img := must(gtk.ImageNew).(*gtk.Image)
				must(img.SetMarginEnd, EmbedMargin)
				asyncFetch(embed.Footer.ProxyIcon, img, 24, 24, cache.Round)

				must(footer.Add, img)
			}

			if embed.Footer.Text != "" {
				text := must(gtk.LabelNew, embed.Footer.Text).(*gtk.Label)
				must(text.SetOpacity, 0.65)
				must(text.SetLineWrap, true)
				must(text.SetLineWrapMode, pango.WRAP_WORD_CHAR)
				must(text.SetXAlign, float64(0.0))

				must(footer.Add, text)
			}
		}

		if embed.Timestamp.Valid() {
			time := humanize.TimeAgo(embed.Timestamp.Time())
			text := must(gtk.LabelNew, time).(*gtk.Label)
			if embed.Footer != nil {
				must(text.SetText, " - "+time)
			}

			must(footer.Add, text)
		}

		must(main.Add, footer)
	}

	if embed.Thumbnail != nil {
		wrapper := must(gtk.BoxNew, gtk.ORIENTATION_HORIZONTAL, 0).(*gtk.Box)
		must(wrapper.Add, main)

		// Do a shitty hack:
		main = wrapper
		must(main.SetHAlign, gtk.ALIGN_START)

		w, h := int(embed.Thumbnail.Width), int(embed.Thumbnail.Height)
		w, h = maxSize(w, h, 80, 80)

		must(wrapper.Add, newExtraImage(
			sizeToURL(embed.Thumbnail.Proxy, w, h),
			embed.Thumbnail.URL, 0, 0,
		))
	}

	if embed.Image != nil {
		wrapper := must(gtk.BoxNew, gtk.ORIENTATION_VERTICAL, 0).(*gtk.Box)
		must(wrapper.Add, main)

		// Do a shitty hack again:
		main = wrapper
		must(main.SetHAlign, gtk.ALIGN_START)

		w, h := int(embed.Image.Width), int(embed.Image.Height)
		w, h = maxSize(w, h, EmbedMaxWidth, EmbedImgHeight)

		must(wrapper.Add, newExtraImage(
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
	icon := App.parser.GetIcon("image-missing", EmbedAvatarSize)
	must(img.SetFromPixbuf, icon)

	if len(pp) == 0 && w != 0 && h != 0 {
		go func() {
			if err := cache.SetImageAsync(url, img, w, h); err != nil {
				log.Errorln("Failed to get image", url+":", err)
				return
			}
		}()

	} else {
		go func() {
			if err := cache.SetImageScaled(url, img, w, h, pp...); err != nil {
				log.Errorln("Failed to get image", url+":", err)
				return
			}
		}()
	}
}
