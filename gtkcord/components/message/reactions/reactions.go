package reactions

import (
	"strconv"

	"github.com/diamondburned/arikawa/api"
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/md"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/gotk3/gotk3/gtk"
)

const EmojiSize = 22

type Container struct {
	*gtk.FlowBox
	Reactions map[api.EmojiAPI]*Reaction

	MessageID discord.Snowflake
	ChannelID discord.Snowflake

	state *ningen.State
}

func NewContainer(m *discord.Message) *Container {
	f, _ := gtk.FlowBoxNew()

	gtkutils.InjectCSSUnsafe(f, "reactions", "")

	container := &Container{
		FlowBox:   f,
		Reactions: map[api.EmojiAPI]*Reaction{},
		MessageID: m.ID,
		ChannelID: m.ChannelID,
	}

	for _, reaction := range m.Reactions {
		container.addReaction(reaction)
	}

	// Setting properties after adding may help?
	f.SetColumnSpacing(0) // buttons already have margins
	f.SetRowSpacing(0)
	f.SetHAlign(gtk.ALIGN_START)
	f.SetMaxChildrenPerLine(22)
	f.SetHomogeneous(true)
	f.Show()

	return container
}

func (c *Container) SetState(s *ningen.State) {
	c.state = s
}

func (c *Container) Search(chID, msgID discord.Snowflake, emoji api.EmojiAPI) *Reaction {
	if r, ok := c.Reactions[emoji]; ok {
		return r
	}
	return nil
}

func (c *Container) addReaction(reaction discord.Reaction) {
	r := newReaction(reaction.Emoji, reaction.Count, reaction.Me)
	c.FlowBox.Add(r)
	c.Reactions[r.String] = r

	r.Connect("destroy", func() {
		delete(c.Reactions, r.String)
	})
	r.Button.Connect("toggled", func() {
		c.clicked(r)
	})
}

func (c *Container) ReactAdd(r *gateway.MessageReactionAddEvent) {
	c.reactSomething(r.ChannelID, r.MessageID, r.Emoji, 0)
}

func (c *Container) ReactRemove(r *gateway.MessageReactionRemoveEvent) {
	c.reactSomething(r.ChannelID, r.MessageID, r.Emoji, 1)
}

// RemoveAll removes everything.
func (c *Container) RemoveAll() {
	semaphore.Async(c.removeAll, (*discord.Emoji)(nil))
}

func (c *Container) RemoveEmoji(emoji discord.Emoji) {
	semaphore.Async(c.removeAll, &emoji)
}

func (c *Container) removeAll(emoji *discord.Emoji) {
	if emoji != nil {
		if r, ok := c.Reactions[emoji.APIString()]; ok {
			r.update(nil)
		}
		return
	}

	// Remove EVERYTHING.
	for k, r := range c.Reactions {
		c.Remove(r)
		delete(c.Reactions, k)
	}
}

// add or remove? dunno, but this is short code, i like. 0: add, 1: remove.
func (c *Container) reactSomething(ch, msg discord.Snowflake, emoji discord.Emoji, code int) {
	if c.ChannelID != ch || c.MessageID != msg || c.state == nil {
		return
	}

	// Reaction found. Do the unoptimized thing.
	m, err := c.state.Store.Message(ch, msg)
	if err != nil {
		log.Errorln("react: message store get failed:", err)
		return
	}

	var target *discord.Reaction
	// The fact that this code is hella unoptimizes bothers me a lot. It
	// could've been a lot more optimized just by me keeping state. But no.
	for _, r := range m.Reactions {
		if r.Emoji.ID == emoji.ID && r.Emoji.Name == emoji.Name {
			target = &r
			break
		}
	}

	semaphore.Async(func() {
		if r := c.Search(ch, msg, emoji.APIString()); r != nil {
			// Reaction found, remove.
			r.update(target)
			return
		}

		switch code {
		case 0:
			if target == nil {
				log.Errorln("Can't find reaction:", emoji)
				return
			}
			// Reaction not found, add it into the message.
			c.addReaction(*target)

		case 1:
			// can't do anything.
		}
	})
}

func (c *Container) clicked(r *Reaction) {
	if r.Button.GetActive() {
		// Only increment the counter by the event. If react() fails, it
		// will deactivate the button.
		go c.react(r)
	} else {
		// Same as above, but decrement.
		go c.unreact(r)
	}
}

func (c *Container) react(r *Reaction) {
	if c.state == nil {
		return
	}

	if err := c.state.React(c.ChannelID, c.MessageID, r.String); err == nil {
		// Worked.
		return
	}

	// Unactivate the button, because there won't be an event.
	semaphore.Async(r.Button.SetActive, false)
}

func (c *Container) unreact(r *Reaction) {
	if c.state == nil {
		return
	}

	if err := c.state.Unreact(c.ChannelID, c.MessageID, r.String); err == nil {
		// Worked.
		return
	}

	// Unactivate the button, because there won't be an event.
	semaphore.Async(r.Button.SetActive, true)
}

type Reaction struct {
	*gtk.FlowBoxChild
	Button *gtk.ToggleButton
	Emoji  gtk.IWidget // *gtk.Image or *gtk.Label

	String api.EmojiAPI
}

func newReaction(emoji discord.Emoji, count int, me bool) *Reaction {
	f, _ := gtk.FlowBoxChildNew()
	f.Show()

	b, _ := gtk.ToggleButtonNew()
	b.SetRelief(gtk.RELIEF_NONE)
	b.SetActive(me)
	b.SetAlwaysShowImage(true)
	b.SetImagePosition(gtk.POS_LEFT)
	b.SetLabel(strconv.Itoa(count))
	b.Show()

	f.Add(b)
	gtkutils.InjectCSSUnsafe(b, "reaction", "")

	reaction := &Reaction{FlowBoxChild: f, Button: b, String: emoji.APIString()}

	// If the emoji is a custom one:
	if emoji.ID.Valid() {
		url := md.EmojiURL(emoji.ID.String(), emoji.Animated)

		i, _ := gtk.ImageNew()
		i.SetSizeRequest(EmojiSize, EmojiSize)
		i.SetVAlign(gtk.ALIGN_CENTER)
		i.SetHAlign(gtk.ALIGN_CENTER)
		i.SetMarginEnd(2)
		i.Show()
		cache.AsyncFetchUnsafe(url, i, EmojiSize, EmojiSize)

		reaction.Emoji = i
		b.SetImage(i)

	} else {
		l, _ := gtk.LabelNew(emoji.Name)
		l.SetSizeRequest(EmojiSize, EmojiSize)
		l.SetVAlign(gtk.ALIGN_CENTER)
		l.SetHAlign(gtk.ALIGN_CENTER)
		l.SetMarginEnd(2)
		l.Show()

		reaction.Emoji = l
		b.SetImage(l)
	}

	// Set "padding"
	if c, _ := b.GetChild(); c != nil {
		c.(gtkutils.SizeRequester).SetSizeRequest(EmojiSize+9, -1)
		gtkutils.Margin2(c.(gtkutils.Marginator), 2, 5)
	}

	return reaction
}

func (r *Reaction) update(reaction *discord.Reaction) {
	// If the reaction is gone:
	if reaction == nil || reaction.Count == 0 {
		r.Destroy()
		return
	}

	r.Button.SetLabel(strconv.Itoa(reaction.Count))

	// Prevent flickering.
	if r.Button.GetActive() != reaction.Me {
		r.Button.SetActive(reaction.Me)
	}
}
