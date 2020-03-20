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
		t.SetProperty("foreground", "rgba(0, 0, 0, 0)") // transparent
		t.SetProperty("background", "#202225")

		t.Connect("event", func(t *gtk.TextTag, _ *gtk.TextView, ev *gdk.Event) {
			if gtkutils.EventIsLeftClick(ev) {
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

	tag   *gtk.TextTag
	attr  Attribute
	color string
}

func (s *TagState) colorTag(attr Attribute, color string) *gtk.TextTag {
	return ColorTag(s.table, attr, color)
}

func (s *TagState) tagSet(attr Attribute, enter bool) *gtk.TextTag {
	if enter {
		return s.tagAdd(attr)
	} else {
		return s.tagRemove(attr)
	}
}

func (s *TagState) tagAdd(attr Attribute) *gtk.TextTag {
	if s.attr != s.attr|attr {
		s.attr |= attr
		s.tag = s.colorTag(s.attr, s.color)
	}
	return s.tag
}

func (s *TagState) tagRemove(attr Attribute) *gtk.TextTag {
	s.attr &= ^attr
	s.tag = s.colorTag(s.attr, s.color)
	return s.tag
}

func (s *TagState) tagReset() *gtk.TextTag {
	s.attr = 0
	s.color = ""
	s.tag = s.colorTag(s.attr, s.color)
	return s.tag
}

func (s *TagState) tagSetColor(color string) *gtk.TextTag {
	if s.color != color {
		s.color = color
		s.tag = s.colorTag(s.attr, s.color)
	}
	return s.tag
}

func (s *TagState) tagSetAttrAndColor(attr Attribute, color string) *gtk.TextTag {
	s.color = color
	s.attr = attr
	s.tag = s.colorTag(s.attr, s.color)
	return s.tag
}

func (s *TagState) injectTag(tag *gtk.TextTag) {
	s.attr.tag(tag, s.color)
}

func (s *TagState) tagWith(attr Attribute) *gtk.TextTag {
	return s.colorTag(s.attr|attr, s.color)
}

func (s *TagState) tagWithColor(color string) *gtk.TextTag {
	return s.colorTag(s.attr, color)
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

// func (s *TagState)

type PressedEvent struct {
	*gdk.EventButton
	TextView *gtk.TextView
}

func setHandler(fn func(PressedEvent)) func(*gtk.TextTag, *gtk.TextView, *gdk.Event) {
	return func(_ *gtk.TextTag, tv *gtk.TextView, ev *gdk.Event) {
		evButton := gdk.EventButtonNewFromEvent(ev)
		if evButton.Type() != gdk.EVENT_BUTTON_RELEASE || evButton.Button() != 1 {
			return
		}

		fn(PressedEvent{
			EventButton: evButton,
			// copy textview so we can still reuse mdState
			TextView: tv,
		})
	}
}
