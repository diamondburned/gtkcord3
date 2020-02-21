package md

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/log"
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
		return AttrItalics
	case bytes.Equal(tag, []byte("*")), bytes.Equal(tag, []byte("_")):
		return AttrItalics
	case bytes.Equal(tag, []byte("***")):
		return AttrBold | AttrItalics
	case bytes.Equal(tag, []byte("~~")):
		return AttrStrikethrough
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

func (p *Parser) Hyperlink(url string) *gtk.TextTag {
	return semaphore.IdleMust(p.hyperlink, url).(*gtk.TextTag)
}

func (p *Parser) hyperlink(url string) *gtk.TextTag {
	v, err := p.table.Lookup("link_" + url)
	if err == nil {
		return v
	}

	t, err := gtk.TextTagNew("link_" + url)
	if err != nil {
		log.Panicln("Failed to create new hyperlink tag:", err)
	}

	t.SetProperty("underline", pango.UNDERLINE_SINGLE)
	t.SetProperty("foreground", "#3F7CE0")
	t.Connect("event", func(_ *gtk.TextTag, _ *gtk.TextView, ev *gdk.Event) {
		evKey := gdk.EventButtonNewFromEvent(ev)
		if evKey.Type() != gdk.EVENT_BUTTON_RELEASE || evKey.Button() != 1 {
			return
		}

		go func() {
			if err := open.Run(url); err != nil {
				log.Errorln("Failed to open image URL:", err)
			}
		}()
	})

	p.table.Add(t)
	return t
}

func (p *Parser) Tag(attr Attribute) *gtk.TextTag {
	return p.ColorTag(attr, "")
}

func (p *Parser) ColorTag(attr Attribute, color string) *gtk.TextTag {
	return semaphore.IdleMust(p.colorTag, attr, color).(*gtk.TextTag)
}

func (p *Parser) colorTag(attr Attribute, color string) *gtk.TextTag {
	var key = attr.StringInt() + color

	v, err := p.table.Lookup(key)
	if err == nil {
		return v
	}

	t, err := gtk.TextTagNew(key)
	if err != nil {
		log.Panicln("Failed to create new tag with", attr, color)
	}

	if color != "" {
		t.SetProperty("foreground", color)
	}

	// TODO: hidden unless on hover

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
		t.SetProperty("size", "smaller")
	}

	p.table.Add(t)
	return t
}

func (s *mdState) tagAdd(attr Attribute) *gtk.TextTag {
	if s.attr != s.attr|attr {
		s.attr |= attr
		s.tag = s.p.ColorTag(s.attr, s.color)
	}
	return s.tag
}

func (s *mdState) tagRemove(attr Attribute) *gtk.TextTag {
	s.attr &= ^attr
	s.tag = s.p.ColorTag(s.attr, s.color)
	return s.tag
}

func (s *mdState) tagReset() *gtk.TextTag {
	s.attr = 0
	s.color = ""
	s.tag = s.p.ColorTag(s.attr, s.color)
	return s.tag
}

func (s *mdState) tagSetColor(color string) *gtk.TextTag {
	if s.color != color {
		s.color = color
		s.tag = s.p.ColorTag(s.attr, s.color)
	}
	return s.tag
}

func (s *mdState) tagSetAttrAndColor(attr Attribute, color string) *gtk.TextTag {
	s.color = color
	s.attr = attr
	s.tag = s.p.ColorTag(s.attr, s.color)
	return s.tag
}

func (s *mdState) tagWith(attr Attribute) *gtk.TextTag {
	return s.p.ColorTag(s.attr|attr, s.color)
}

func (s *mdState) tagWithColor(color string) *gtk.TextTag {
	return s.p.ColorTag(s.attr, color)
}
