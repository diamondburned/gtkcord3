package gtkcord

import (
	"fmt"
	"path"
	"strconv"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/humanize"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

const (
	EmbedAvatarSize = 24
	EmbedMaxWidth   = 480
	EmbedImgHeight  = 400 // max
	EmbedMargin     = 8

	EmbedMainCSS = `
		.embed {
			border-left: 4px solid #%06X;
			background-color: rgba(0, 0, 0, 0.1);
		}
	`
)

func NewAttachment(msg discord.Message) []ExtendedWidget {
	// Discord's supported formats
	var formats = []string{".jpg", ".jpeg", ".png", ".webp", ".gif"}
	var widgets = make([]ExtendedWidget, 0, len(msg.Attachments))

	for _, att := range msg.Attachments {
		if att.Width == 0 || att.Height == 0 {
			continue
		}

		if !validExt(att.Proxy, formats) {
			continue
		}

		w := newExtraImage(
			sizeToURL(att.Proxy,
				int(att.Width), int(att.Height),
				EmbedMaxWidth, EmbedImgHeight),
			cache.Resize(EmbedMaxWidth, EmbedImgHeight),
		)

		if w, ok := w.(embedMarginator); ok {
			must(w.SetMarginStart, 0)
		}

		widgets = append(widgets, w)
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

func NewEmbed(msg discord.Message) []ExtendedWidget {
	if len(msg.Embeds) == 0 {
		return nil
	}

	var embeds = make([]ExtendedWidget, 0, len(msg.Embeds))

	for _, embed := range msg.Embeds {
		w := newEmbed(msg, embed)
		if w == nil {
			continue
		}

		embeds = append(embeds, w)
	}

	return embeds
}

func newEmbed(msg discord.Message, embed discord.Embed) ExtendedWidget {
	switch embed.Type {
	case discord.NormalEmbed:
		return newNormalEmbed(msg, embed)
	case discord.ImageEmbed:
		return newImageEmbed(embed)
	case discord.VideoEmbed:
		// Unsupported
		return nil
	}

	return nil
}

func newExtraImage(url string, pp ...cache.Processor) ExtendedWidget {
	img := must(gtk.ImageNew).(*gtk.Image)
	must(img.SetVAlign, gtk.ALIGN_START)
	must(img.SetHAlign, gtk.ALIGN_START)
	must(embedSetMargin, img)

	asyncFetch(url, img, pp...)

	return img
}

// https://stackoverflow.com/questions/3008772/how-to-smart-resize-a-displayed-image-to-original-aspect-ratio
func sizeToURL(url string, w, h, maxW, maxH int) string {
	if w > maxW && h > maxH {
		return url
	}

	W := float64(w)
	H := float64(h)

	ratio := min(float64(maxW)/W, float64(maxH)/H)
	w = int(W / ratio)
	h = int(H / ratio)

	return url + "?width=" + strconv.Itoa(w) + "&height=" + strconv.Itoa(h)
}

func min(i, j float64) float64 {
	if i < j {
		return i
	}
	return j
}

func newImageEmbed(embed discord.Embed) ExtendedWidget {
	return newExtraImage(
		embed.Thumbnail.Proxy,
		cache.Resize(EmbedMaxWidth, EmbedImgHeight),
	)
}

func newNormalEmbed(msg discord.Message, embed discord.Embed) ExtendedWidget {
	main := must(gtk.BoxNew, gtk.ORIENTATION_VERTICAL, 0).(*gtk.Box)
	must(main.SetHAlign, gtk.ALIGN_START)

	if embed.Author != nil {
		box := must(gtk.BoxNew, gtk.ORIENTATION_HORIZONTAL, 0).(*gtk.Box)
		must(embedSetMargin, box)

		if embed.Author.ProxyIcon != "" {
			img := must(gtk.ImageNew).(*gtk.Image)
			must(img.SetMarginEnd, EmbedMargin)
			asyncFetch(embed.Author.ProxyIcon, img, cache.Resize(24, 24), cache.Round)

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
		var title = `<span size="larger">` + escape(embed.Title) + `</span>`
		if embed.URL != "" {
			title = fmt.Sprintf(`<a href="%s">%s</a>`, embed.URL, title)
		}

		label := must(gtk.LabelNew, "").(*gtk.Label)
		must(label.SetMarkup, title)
		must(label.SetLineWrap, true)
		must(label.SetLineWrapMode, pango.WRAP_WORD_CHAR)
		must(label.SetXAlign, float64(0.0))
		must(embedSetMargin, label)

		must(main.Add, label)
	}

	if embed.Description != "" {
		desc := must(App.parser.NewTextBuffer).(*gtk.TextBuffer)
		App.parser.ParseMessage(&msg, []byte(embed.Description), desc)

		txtv := must(gtk.TextViewNewWithBuffer, desc).(*gtk.TextView)
		must(txtv.SetCursorVisible, false)
		must(txtv.SetEditable, false)
		must(txtv.SetWrapMode, gtk.WRAP_WORD_CHAR)
		must(txtv.SetSizeRequest, EmbedMaxWidth, -1)
		must(embedSetMargin, txtv)

		must(main.Add, txtv)
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
				asyncFetch(embed.Footer.ProxyIcon, img, cache.Resize(24, 24), cache.Round)

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

		must(wrapper.Add, newExtraImage(
			embed.Thumbnail.Proxy,
			cache.Resize(80, 80),
		))
	}

	if embed.Image != nil {
		wrapper := must(gtk.BoxNew, gtk.ORIENTATION_VERTICAL, 0).(*gtk.Box)
		must(wrapper.Add, main)

		// Do a shitty hack again:
		main = wrapper
		must(main.SetHAlign, gtk.ALIGN_START)

		must(wrapper.Add, newExtraImage(
			embed.Image.Proxy,
			cache.Resize(EmbedMaxWidth, EmbedImgHeight),
		))
	}

	InjectCSS(main, "embed", fmt.Sprintf(EmbedMainCSS, embed.Color))

	return main
}

type embedMarginator interface {
	SetMarginStart(int)
	SetMarginEnd(int)
	SetMarginTop(int)
	SetMarginBottom(int)
}

func embedSetMargin(w embedMarginator) {
	w.SetMarginStart(EmbedMargin * 2)
	w.SetMarginEnd(EmbedMargin * 2)
	w.SetMarginTop(EmbedMargin)
	w.SetMarginBottom(EmbedMargin / 2)
}

func asyncFetch(url string, img *gtk.Image, pp ...cache.Processor) {
	icon := App.parser.GetIcon("image-missing", EmbedAvatarSize)
	must(img.SetFromPixbuf, icon)

	go func() {
		if err := cache.SetImage(url, img, pp...); err != nil {
			log.Errorln("Failed to get image", url+":", err)
			return
		}
	}()
}
