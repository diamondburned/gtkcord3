package md

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/diamondburned/arikawa/discord"
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
		return AttrUnderline
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

func (s *mdState) tagTable() *gtk.TextTagTable {
	if s.ttt != nil {
		return s.ttt
	}

	ttt, err := s.buf.GetTagTable()
	if err != nil {
		log.Panicln("Failed to get tag table:", err)
	}
	s.ttt = ttt

	return ttt
}

func (s *mdState) setHandler(fn func(PressedEvent)) func(*gtk.TextTag, *gtk.TextView, *gdk.Event) {
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

// func (s *mdState) Hyperlink(url string) *gtk.TextTag {
// 	return semaphore.IdleMust(s.hyperlink, url).(*gtk.TextTag)
// }

func (s *mdState) hyperlink(url string) *gtk.TextTag {
	v, err := s.ttt.Lookup("link_" + url)
	if err == nil {
		return v
	}

	t, err := gtk.TextTagNew("link_" + url)
	if err != nil {
		log.Panicln("Failed to create new hyperlink tag:", err)
	}

	t.SetProperty("underline", pango.UNDERLINE_SINGLE)
	t.SetProperty("foreground", "#3F7CE0")
	t.Connect("event", s.setHandler(func(PressedEvent) {
		if err := open.Start(url); err != nil {
			log.Errorln("Failed to open image URL:", err)
		}
	}))

	s.ttt.Add(t)
	return t
}

func (s *mdState) insertUserMention(id []byte) {
	d, err := discord.ParseSnowflake(string(id))
	if err != nil {
		s.insertWithTag(id, nil)
		return
	}

	var target discord.GuildUser
	for _, user := range s.m.Mentions {
		if user.ID == d {
			target = user
			break
		}
	}

	if !target.ID.Valid() {
		s.insertWithTag(id, nil)
		return
	}

	t := s.mentionTag("@"+target.ID.String(), func(ev PressedEvent) {
		if UserPressed != nil {
			UserPressed(ev, target)
		}
	})

	if !s.m.GuildID.Valid() {
		s.insertWithTag([]byte("@"+target.User.Username), t)
		return
	}

	var name = target.User.Username
	if m, err := s.d.Store.Member(s.m.GuildID, target.ID); err == nil && m.Nick != "" {
		name = m.Nick
	}

	s.insertWithTag([]byte("@"+name), t)
}

func (s *mdState) insertChannelMention(id []byte) {
	d, err := discord.ParseSnowflake(string(id))
	if err != nil {
		s.insertWithTag(id, nil)
		return
	}

	c, err := s.d.Store.Channel(d)
	if err != nil {
		s.insertWithTag(id, nil)
		return
	}

	var channel = *c

	t := s.mentionTag("#"+c.ID.String(), func(ev PressedEvent) {
		if ChannelPressed != nil {
			ChannelPressed(ev, channel)
		}
	})

	s.insertWithTag([]byte("#"+c.Name), t)
}

// func (s *mdState) MentionTag(key string, asyncH func(PressedEvent)) *gtk.TextTag {
// 	return semaphore.IdleMust(s.mentionTag, key, asyncH).(*gtk.TextTag)
// }

func (s *mdState) mentionTag(key string, asyncH func(PressedEvent)) *gtk.TextTag {
	v, err := s.ttt.Lookup(key)
	if err == nil {
		return v
	}

	t, err := gtk.TextTagNew(key)
	if err != nil {
		log.Panicln("Failed to create new hyperlink tag:", err)
	}
	t.SetProperty("foreground", "#7289DA")
	t.Connect("event", s.setHandler(asyncH))

	s.ttt.Add(t)
	return t
}

// func (s *mdState) InlineEmojiTag() *gtk.TextTag {
// 	return semaphore.IdleMust(s.inlineEmojiTag).(*gtk.TextTag)
// }

func (s *mdState) inlineEmojiTag() *gtk.TextTag {
	t, err := s.ttt.Lookup("emoji")
	if err == nil {
		return t
	}

	t, err = gtk.TextTagNew("emoji")
	if err != nil {
		log.Panicln("Failed to create new emoji tag:", err)
	}

	t.SetProperty("rise", -4096)
	t.SetProperty("rise-set", true)

	s.ttt.Add(t)
	return t
}

// func (s *mdState) tag(attr Attribute) *gtk.TextTag {
// 	return s.colorTag(attr, "")
// }

// func (s *mdState) ColorTag(attr Attribute, color string) *gtk.TextTag {
// 	return semaphore.IdleMust(s.colorTag, attr, color).(*gtk.TextTag)
// }

func (s *mdState) colorTag(attr Attribute, color string) *gtk.TextTag {
	var key = attr.StringInt() + color

	v, err := s.ttt.Lookup(key)
	if err == nil {
		return v
	}

	t, err := gtk.TextTagNew(key)
	if err != nil {
		log.Panicln("Failed to create new tag with", attr, color)
	}

	t.SetProperty("weight", pango.WEIGHT_MEDIUM)

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
		t.SetProperty("scale", 0.84)
		t.SetProperty("scale-set", true)
	}

	s.ttt.Add(t)
	return t
}

func (s *mdState) tagAdd(attr Attribute) *gtk.TextTag {
	if s.attr != s.attr|attr {
		s.attr |= attr
		s.tag = s.colorTag(s.attr, s.color)
	}
	return s.tag
}

func (s *mdState) tagRemove(attr Attribute) *gtk.TextTag {
	s.attr &= ^attr
	s.tag = s.colorTag(s.attr, s.color)
	return s.tag
}

func (s *mdState) tagReset() *gtk.TextTag {
	s.attr = 0
	s.color = ""
	s.tag = s.colorTag(s.attr, s.color)
	return s.tag
}

func (s *mdState) tagSetColor(color string) *gtk.TextTag {
	if s.color != color {
		s.color = color
		s.tag = s.colorTag(s.attr, s.color)
	}
	return s.tag
}

func (s *mdState) tagSetAttrAndColor(attr Attribute, color string) *gtk.TextTag {
	s.color = color
	s.attr = attr
	s.tag = s.colorTag(s.attr, s.color)
	return s.tag
}

func (s *mdState) tagWith(attr Attribute) *gtk.TextTag {
	return s.colorTag(s.attr|attr, s.color)
}

func (s *mdState) tagWithColor(color string) *gtk.TextTag {
	return s.colorTag(s.attr, color)
}
