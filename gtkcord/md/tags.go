package md

import (
	"bytes"
	"strconv"
	"strings"
	"sync/atomic"

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

func Tag(buf *gtk.TextBuffer, name string, attr Attribute) *gtk.TextTag {
	return ColorTag(buf, name, attr, "")
}

func ColorTag(buf *gtk.TextBuffer, name string, attr Attribute, color string) *gtk.TextTag {
	var attrs = map[string]interface{}{}

	if color != "" {
		attrs["foreground"] = color
	}

	// TODO: hidden unless on hover

	if attr.Has(AttrBold) {
		attrs["weight"] = pango.WEIGHT_BOLD
	}
	if attr.Has(AttrItalics) {
		attrs["style"] = pango.STYLE_ITALIC
	}
	if attr.Has(AttrUnderline) {
		attrs["weight"] = pango.UNDERLINE_SINGLE
	}
	if attr.Has(AttrStrikethrough) {
		attrs["strikethrough"] = true
	}
	if attr.Has(AttrQuoted) && color == "" {
		attrs["foreground"] = "#789922"
	}
	if attr.Has(AttrSpoiler) && color == "" {
		attrs["foreground"] = "#808080"
	}
	if attr.Has(AttrMonospace) {
		attrs["family"] = "monospace"
	}

	return buf.CreateTag(name, attrs)
}

type TagState struct {
	buf     *gtk.TextBuffer
	tag     *gtk.TextTag
	color   string
	attr    Attribute
	counter uint64
}

func (s *TagState) incrCounter() string {
	return strconv.FormatUint(atomic.AddUint64(&s.counter, 1), 10)
}

func (s *TagState) Get() *gtk.TextTag {
	return s.tag
}

func (s *TagState) Use(buf *gtk.TextBuffer) {
	s.buf = buf
	s.attr = 0
	s.counter = 0
	s.color = ""
	s.tag = ColorTag(s.buf, s.incrCounter(), s.attr, s.color)
}

func (s *TagState) Attr() Attribute {
	return s.attr
}

func (s *TagState) Add(attr Attribute) *gtk.TextTag {
	if s.attr != s.attr|attr {
		s.attr |= attr
		s.tag = ColorTag(s.buf, s.incrCounter(), s.attr, s.color)
	}
	return s.tag
}

func (s *TagState) Remove(attr Attribute) *gtk.TextTag {
	s.attr &= ^attr
	s.tag = ColorTag(s.buf, s.incrCounter(), s.attr, s.color)
	return s.tag
}

func (s *TagState) Reset() *gtk.TextTag {
	s.attr = 0
	s.color = ""
	s.tag = ColorTag(s.buf, s.incrCounter(), s.attr, s.color)
	return s.tag
}

func (s *TagState) SetColor(color string) *gtk.TextTag {
	if s.color != color {
		s.color = color
		s.tag = ColorTag(s.buf, s.incrCounter(), s.attr, s.color)
	}
	return s.tag
}

func (s *TagState) SetAttrAndColor(attr Attribute, color string) *gtk.TextTag {
	s.color = color
	s.attr = attr
	s.tag = ColorTag(s.buf, s.incrCounter(), s.attr, s.color)
	return s.tag
}

func (s *TagState) With(attr Attribute) *gtk.TextTag {
	return ColorTag(s.buf, s.incrCounter(), s.attr|attr, s.color)
}

func (s *TagState) WithColor(color string) *gtk.TextTag {
	return ColorTag(s.buf, s.incrCounter(), s.attr, color)
}
