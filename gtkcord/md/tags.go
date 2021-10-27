package md

import (
	"strings"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/gotk4/pkg/gdk/v3"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gotk4/pkg/pango"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/diamondburned/ningen/v2/md"
	"github.com/skratchdot/open-golang/open"
)

func AttrMarkup(a md.Attribute) string {
	var attrs = make([]string, 0, 7)

	if a.Has(md.AttrBold) {
		attrs = append(attrs, `weight="bold"`)
	}
	if a.Has(md.AttrItalics) {
		attrs = append(attrs, `style="italic"`)
	}
	if a.Has(md.AttrUnderline) {
		attrs = append(attrs, `underline="single"`)
	}
	if a.Has(md.AttrStrikethrough) {
		attrs = append(attrs, `strikethrough="true"`)
	}
	if a.Has(md.AttrSpoiler) {
		attrs = append(attrs, `foreground="#808080"`) // no fancy click here
	}
	if a.Has(md.AttrMonospace) {
		attrs = append(attrs, `font_family="monospace"`)
	}

	// only append this if not spoiler to avoid duplicate tags
	if a.Has(md.AttrQuoted) && !a.Has(md.AttrStrikethrough) {
		attrs = append(attrs, `foreground="#789922"`)
	}

	return strings.Join(attrs, " ")
}

func tag(attr md.Attribute, t *gtk.TextTag, color string) {
	t.SetObjectProperty("weight", pango.WeightMedium)

	if color != "" {
		t.SetObjectProperty("foreground", color)
	}

	// TODO: hidden unless on hover
	if attr.Has(md.AttrSpoiler) {
		// Same color, so text appears invisible.
		t.SetObjectProperty("foreground", "#202225")
		t.SetObjectProperty("background", "#202225")

		t.Connect("event", func(t *gtk.TextTag, _ *gtk.TextView, ev *gdk.Event) {
			if gtkutils.EventIsLeftClick(ev) {
				// Show text:
				t.SetObjectProperty("foreground-set", false)
				t.SetObjectProperty("background-set", false)
			}
		})
	}

	if attr.Has(md.AttrBold) {
		t.SetObjectProperty("weight", pango.WeightBold)
	}
	if attr.Has(md.AttrItalics) {
		t.SetObjectProperty("style", pango.StyleItalic)
	}
	if attr.Has(md.AttrUnderline) {
		t.SetObjectProperty("underline", pango.UnderlineSingle)
	}
	if attr.Has(md.AttrStrikethrough) {
		t.SetObjectProperty("strikethrough", true)
	}
	if attr.Has(md.AttrQuoted) && color == "" {
		t.SetObjectProperty("foreground", "#789922")
	}
	if attr.Has(md.AttrSpoiler) && color == "" {
		t.SetObjectProperty("foreground", "#808080")
	}
	if attr.Has(md.AttrMonospace) {
		t.SetObjectProperty("family", "monospace")
		t.SetObjectProperty("scale", 0.84)
		t.SetObjectProperty("scale-set", true)
	}
}

func ColorTag(table *gtk.TextTagTable, attr md.Attribute, color string) *gtk.TextTag {
	var key = attr.StringInt() + color

	t := table.Lookup(key)
	if t != nil {
		return t
	}

	t = gtk.NewTextTag(key)
	// Load
	tag(attr, t, color)

	table.Add(t)
	return t
}

type TagState struct {
	table *gtk.TextTagTable

	tag *gtk.TextTag
	Tag
}

var EmptyTag = Tag{}

type Tag struct {
	Attr  md.Attribute
	Color string
}

func (t Tag) Combine(tag Tag) Tag {
	attr := t.Attr | tag.Attr
	if tag.Color != "" {
		return Tag{attr, tag.Color}
	}

	return Tag{attr, t.Color}
}

func (s *TagState) colorTag(tag Tag) *gtk.TextTag {
	return ColorTag(s.table, tag.Attr, tag.Color)
}

func (s *TagState) tagSet(attr md.Attribute, enter bool) *gtk.TextTag {
	if enter {
		return s.tagAdd(attr)
	} else {
		return s.tagRemove(attr)
	}
}

func (s *TagState) tagAdd(attr md.Attribute) *gtk.TextTag {
	s.Attr |= attr
	s.tag = s.colorTag(s.Tag)
	return s.tag
}

func (s *TagState) tagRemove(attr md.Attribute) *gtk.TextTag {
	s.Attr &= ^attr
	s.tag = s.colorTag(s.Tag)
	return s.tag
}

// injectTag copies attributes and colors from the state to the given tag.
func (s *TagState) injectTag(textTag *gtk.TextTag) {
	tag(s.Attr, textTag, s.Color)
}

// does not change state
func (s *TagState) hyperlink(url string) *gtk.TextTag {
	key := "link_" + url

	tag := s.table.Lookup(key)
	if tag != nil {
		return tag
	}

	tag = gtk.NewTextTag(key)
	tag.SetObjectProperty("underline", pango.UnderlineSingle)
	tag.SetObjectProperty("foreground", "#3F7CE0")
	tag.Connect("event", setHandler(func(PressedEvent) {
		if err := open.Start(url); err != nil {
			log.Errorln("Failed to open image URL:", err)
		}
	}))

	s.table.Add(tag)
	return tag
}

func (s *TagState) guildUser(gu *discord.GuildUser) *gtk.TextTag {
	if UserPressed == nil {
		return nil
	}

	return s.addHandler("@"+gu.ID.String(), func(ev PressedEvent) {
		UserPressed(ev, gu)
	})
}

func (s *TagState) channel(ch *discord.Channel) *gtk.TextTag {
	if ChannelPressed == nil {
		return nil
	}

	return s.addHandler("#"+ch.ID.String(), func(ev PressedEvent) {
		ChannelPressed(ev, ch)
	})
}

/*
func (s *TagState) inlineEmojiTag() *gtk.TextTag {
	tag := s.table.Lookup("emoji")
	if tag != nil {
		return tag
	}

	tag = gtk.NewTextTag("emoji")
	tag.SetObjectProperty("rise", -8192)

	s.table.Add(tag)
	return tag
}
*/

func (s *TagState) timestamp() *gtk.TextTag {
	v := s.table.Lookup("timestamp")
	if v != nil {
		return v
	}

	v = gtk.NewTextTag("timestamp")
	v.SetObjectProperty("scale", 0.84)
	v.SetObjectProperty("scale-set", true)
	v.SetObjectProperty("foreground", "#808080")

	s.table.Add(v)
	return v
}

func (s *TagState) addHandler(key string, handler func(PressedEvent)) *gtk.TextTag {
	v := s.table.Lookup(key)
	if v != nil {
		return v
	}

	tag := gtk.NewTextTag(key)
	tag.SetObjectProperty("foreground", "#7289DA")
	tag.Connect("event", setHandler(handler))

	s.table.Add(tag)
	return tag
}

type PressedEvent struct {
	*gdk.EventButton
	TextView *gtk.TextView
}

func setHandler(fn func(PressedEvent)) func(*gtk.TextTag, *gtk.TextView, *gdk.Event) {
	return func(_ *gtk.TextTag, tv *gtk.TextView, ev *gdk.Event) {
		if ev.AsType() != gdk.ButtonReleaseType {
			return
		}

		evButton := ev.AsButton()
		if evButton.Button() != gdk.BUTTON_PRIMARY {
			return
		}

		fn(PressedEvent{
			EventButton: evButton,
			TextView:    tv,
		})
	}
}
