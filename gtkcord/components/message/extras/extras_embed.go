package extras

import (
	"fmt"
	"html"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/md"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/gtkcord/variables"
	"github.com/diamondburned/gtkcord3/internal/humanize"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

func newNormalEmbedUnsafe(
	s *ningen.State, msg *discord.Message, embed discord.Embed) gtkutils.ExtendedWidget {

	content, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	content.SetHAlign(gtk.ALIGN_FILL)

	widthHint := 0 // used for calculating requested embed width

	if embed.Author != nil {
		box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
		embedSetMargin(box)

		if embed.Author.ProxyIcon != "" {
			img, _ := gtk.ImageNew()
			img.SetMarginEnd(variables.EmbedMargin)
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
				img.SetMarginEnd(variables.EmbedMargin)
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
		w, h = maxSize(w, h, variables.EmbedMaxWidth, variables.EmbedImgHeight)

		// set width hint to resize embeds accordingly
		widthHint = w

		wrapper.Add(newExtraImageUnsafe(
			sizeToURL(embed.Image.Proxy, w, h),
			embed.Image.URL, w, h,
		))
	}

	// Calculate the embed width without padding:
	var w = clampWidth(variables.EmbedMaxWidth)
	if widthHint > 0 && w > widthHint {
		w = widthHint
	}

	// Wrap the content inside another box:
	main, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	main.SetHAlign(gtk.ALIGN_START)
	main.SetSizeRequest(w+(variables.EmbedMargin*2), 0)
	main.Add(content)

	// Apply margin to content:
	content.SetMarginTop(variables.EmbedMargin) // account for children not having margin top
	content.SetMarginBottom(variables.EmbedMargin / 2)

	// Add a frame around the main embed:
	gtkutils.InjectCSSUnsafe(main, "embed", fmt.Sprintf(EmbedMainCSS, embed.Color))

	return main
}
