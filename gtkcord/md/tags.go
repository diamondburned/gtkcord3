package md

import (
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/state"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/diamondburned/ningen/md"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
	"github.com/skratchdot/open-golang/open"
)

type Attribute uint16

const (
	_ Attribute = 1 << iota
	AttrBold
	AttrItalics
	AttrUnderline
	AttrStrikethrough
	AttrSpoiler
	AttrMonospace
	AttrQuoted
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
	t.SetProperty("weight", pango.WEIGHT_MEDIUM)

	if color != "" {
		t.SetProperty("foreground", color)
	}

	// TODO: hidden unless on hover
	if attr.Has(md.AttrSpoiler) {
		// Same color, so text appears invisible.
		t.SetProperty("foreground", "#202225")
		t.SetProperty("background", "#202225")

		t.Connect("event", func(t *gtk.TextTag, _ *gtk.TextView, ev *gdk.Event) {
			if gtkutils.EventIsLeftClick(ev) {
				// Show text:
				t.SetProperty("foreground-set", false)
				t.SetProperty("background-set", false)
			}
		})
	}

	if attr.Has(md.AttrBold) {
		t.SetProperty("weight", pango.WEIGHT_BOLD)
	}
	if attr.Has(md.AttrItalics) {
		t.SetProperty("style", pango.STYLE_ITALIC)
	}
	if attr.Has(md.AttrUnderline) {
		t.SetProperty("underline", pango.UNDERLINE_SINGLE)
	}
	if attr.Has(md.AttrStrikethrough) {
		t.SetProperty("strikethrough", true)
	}
	if attr.Has(md.AttrQuoted) && color == "" {
		t.SetProperty("foreground", "#789922")
	}
	if attr.Has(md.AttrSpoiler) && color == "" {
		t.SetProperty("foreground", "#808080")
	}
	if attr.Has(md.AttrMonospace) {
		t.SetProperty("family", "monospace")
		t.SetProperty("scale", 0.84)
		t.SetProperty("scale-set", true)
	}
}

func ColorTag(table *gtk.TextTagTable, attr md.Attribute, color string) *gtk.TextTag {
	var key = attr.StringInt() + color

	v, err := table.Lookup(key)
	if err == nil {
		return v
	}

	t, err := gtk.TextTagNew(key)
	if err != nil {
		log.Panicln("Failed to create new tag with", attr, color)
	}

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

func (s *TagState) tagReset() *gtk.TextTag {
	s.Attr = 0
	s.Color = ""
	s.tag = s.colorTag(s.Tag)
	return s.tag
}

func (s *TagState) tagSetColor(color string) *gtk.TextTag {
	if s.Color != color {
		s.Color = color
		s.tag = s.colorTag(s.Tag)
	}
	return s.tag
}

func (s *TagState) tagSetAttrAndColor(attr md.Attribute, color string) *gtk.TextTag {
	s.Color = color
	s.Attr = attr
	s.tag = s.colorTag(s.Tag)
	return s.tag
}

// injectTag copies attributes and colors from the state to the given tag.
func (s *TagState) injectTag(textTag *gtk.TextTag) {
	tag(s.Attr, textTag, s.Color)
}

func (s *TagState) tagWith(attr md.Attribute) *gtk.TextTag {
	return ColorTag(s.table, s.Attr|attr, s.Color)
}

func (s *TagState) tagWithColor(color string) *gtk.TextTag {
	return ColorTag(s.table, s.Attr, color)
}

// does not change state
func (s *TagState) hyperlink(url string) *gtk.TextTag {
	key := "link_" + url

	v, err := s.table.Lookup(key)
	if err == nil {
		return v
	}

	t, err := gtk.TextTagNew(key)
	if err != nil {
		log.Panicln("Failed to create new hyperlink tag:", err)
	}

	t.SetProperty("underline", pango.UNDERLINE_SINGLE)
	t.SetProperty("foreground", "#3F7CE0")
	t.Connect("event", setHandler(func(PressedEvent) {
		if err := open.Start(url); err != nil {
			log.Errorln("Failed to open image URL:", err)
		}
	}))

	s.table.Add(t)
	return t
}

func searchMember(state state.Store, guild, user discord.Snowflake) *discord.GuildUser {
	m, err := state.Member(guild, user)
	if err == nil {
		return &discord.GuildUser{
			User:   m.User,
			Member: m,
		}
	}

	// Maybe?
	p, err := state.Presence(guild, user)
	if err == nil {
		return &discord.GuildUser{
			User: p.User,
		}
	}

	return nil
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

func (s *TagState) inlineEmojiTag() *gtk.TextTag {
	t, err := s.table.Lookup("emoji")
	if err == nil {
		return t
	}

	t, err = gtk.TextTagNew("emoji")
	if err != nil {
		log.Panicln("Failed to create new emoji tag:", err)
	}

	t.SetProperty("rise", -8192)

	s.table.Add(t)
	return t
}

func (s *TagState) timestamp() *gtk.TextTag {
	v, err := s.table.Lookup("timestamp")
	if err == nil {
		return v
	}

	v, err = gtk.TextTagNew("timestamp")
	if err != nil {
		log.Panicln("Failed to create a new timestamp tag:", err)
	}

	v.SetProperty("scale", 0.84)
	v.SetProperty("scale-set", true)
	v.SetProperty("foreground", "#808080")

	s.table.Add(v)
	return v
}

func (s *TagState) addHandler(key string, handler func(PressedEvent)) *gtk.TextTag {
	v, err := s.table.Lookup(key)
	if err == nil {
		return v
	}

	t, err := gtk.TextTagNew(key)
	if err != nil {
		log.Panicln("Failed to create new hyperlink tag:", err)
	}
	t.SetProperty("foreground", "#7289DA")
	t.Connect("event", setHandler(handler))

	s.table.Add(t)
	return t
}

type PressedEvent struct {
	*gdk.EventButton
	TextView *gtk.TextView
}

func setHandler(fn func(PressedEvent)) func(*gtk.TextTag, *gtk.TextView, *gdk.Event) {
	return func(_ *gtk.TextTag, tv *gtk.TextView, ev *gdk.Event) {
		evButton := gdk.EventButtonNewFromEvent(ev)
		if evButton.Type() != gdk.EVENT_BUTTON_RELEASE || evButton.Button() != gdk.BUTTON_PRIMARY {
			return
		}

		fn(PressedEvent{
			EventButton: evButton,
			TextView:    tv,
		})
	}
}
