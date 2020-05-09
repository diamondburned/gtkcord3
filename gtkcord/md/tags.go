package md

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
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

func TagAttribute(tag []byte) Attribute {
	switch {
	case bytes.Equal(tag, []byte("**")):
		return AttrBold
	case bytes.Equal(tag, []byte("__")):
		return AttrUnderline
	case bytes.Equal(tag, []byte("*")), bytes.Equal(tag, []byte("_")):
		return AttrItalics
	case bytes.Equal(tag, []byte("***")):
		return AttrBold | AttrItalics
	case bytes.Equal(tag, []byte("~~")):
		return AttrStrikethrough
	case bytes.Equal(tag, []byte("||")):
		return AttrSpoiler
	case bytes.Equal(tag, []byte("`")):
		return AttrMonospace
	}
	return 0
}

func (a Attribute) Has(attr Attribute) bool {
	return a&attr == attr
}

func (a *Attribute) Add(attr Attribute) {
	*a |= attr
}
func (a *Attribute) Remove(attr Attribute) {
	*a &= ^attr
}

func (a Attribute) Markup() string {
	var attrs = make([]string, 0, 7)

	if a.Has(AttrBold) {
		attrs = append(attrs, `weight="bold"`)
	}
	if a.Has(AttrItalics) {
		attrs = append(attrs, `style="italic"`)
	}
	if a.Has(AttrUnderline) {
		attrs = append(attrs, `underline="single"`)
	}
	if a.Has(AttrStrikethrough) {
		attrs = append(attrs, `strikethrough="true"`)
	}
	if a.Has(AttrSpoiler) {
		attrs = append(attrs, `foreground="#808080"`) // no fancy click here
	}
	if a.Has(AttrMonospace) {
		attrs = append(attrs, `font_family="monospace"`)
	}

	// only append this if not spoiler to avoid duplicate tags
	if a.Has(AttrQuoted) && !a.Has(AttrStrikethrough) {
		attrs = append(attrs, `foreground="#789922"`)
	}

	return strings.Join(attrs, " ")
}

func (a Attribute) StringInt() string {
	return strconv.FormatUint(uint64(a), 10)
}

func (a Attribute) String() string {
	var attrs = make([]string, 0, 7)
	if a.Has(AttrBold) {
		attrs = append(attrs, "bold")
	}
	if a.Has(AttrItalics) {
		attrs = append(attrs, "italics")
	}
	if a.Has(AttrUnderline) {
		attrs = append(attrs, "underline")
	}
	if a.Has(AttrStrikethrough) {
		attrs = append(attrs, "strikethrough")
	}
	if a.Has(AttrSpoiler) {
		attrs = append(attrs, "spoiler")
	}
	if a.Has(AttrMonospace) {
		attrs = append(attrs, "monospace")
	}
	if a.Has(AttrQuoted) {
		attrs = append(attrs, "quoted")
	}
	return strings.Join(attrs, ", ")
}

func (attr Attribute) tag(t *gtk.TextTag, color string) {
	t.SetProperty("weight", pango.WEIGHT_MEDIUM)

	if color != "" {
		t.SetProperty("foreground", color)
	}

	// TODO: hidden unless on hover
	if attr.Has(AttrSpoiler) {
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

	if attr.Has(AttrBold) {
		t.SetProperty("weight", pango.WEIGHT_BOLD)
	}
	if attr.Has(AttrItalics) {
		t.SetProperty("style", pango.STYLE_ITALIC)
	}
	if attr.Has(AttrUnderline) {
		t.SetProperty("underline", pango.UNDERLINE_SINGLE)
	}
	if attr.Has(AttrStrikethrough) {
		t.SetProperty("strikethrough", true)
	}
	if attr.Has(AttrQuoted) && color == "" {
		t.SetProperty("foreground", "#789922")
	}
	if attr.Has(AttrSpoiler) && color == "" {
		t.SetProperty("foreground", "#808080")
	}
	if attr.Has(AttrMonospace) {
		t.SetProperty("family", "monospace")
		t.SetProperty("scale", 0.84)
		t.SetProperty("scale-set", true)
	}
}

func ColorTag(table *gtk.TextTagTable, attr Attribute, color string) *gtk.TextTag {
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
	attr.tag(t, color)

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
	Attr  Attribute
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

func (s *TagState) tagSet(attr Attribute, enter bool) *gtk.TextTag {
	if enter {
		return s.tagAdd(attr)
	} else {
		return s.tagRemove(attr)
	}
}

func (s *TagState) tagAdd(attr Attribute) *gtk.TextTag {
	s.Attr |= attr
	s.tag = s.colorTag(s.Tag)
	return s.tag
}

func (s *TagState) tagRemove(attr Attribute) *gtk.TextTag {
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

func (s *TagState) tagSetAttrAndColor(attr Attribute, color string) *gtk.TextTag {
	s.Color = color
	s.Attr = attr
	s.tag = s.colorTag(s.Tag)
	return s.tag
}

// injectTag copies attributes and colors from the state to the given tag.
func (s *TagState) injectTag(tag *gtk.TextTag) {
	s.Attr.tag(tag, s.Color)
}

func (s *TagState) tagWith(attr Attribute) *gtk.TextTag {
	return ColorTag(s.table, s.Attr|attr, s.Color)
}

func (s *TagState) tagWithColor(color string) *gtk.TextTag {
	return ColorTag(s.table, s.Attr, color)
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

// func (s *TagState)

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
