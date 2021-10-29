package reactions

import (
	"strconv"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/md"
	"github.com/diamondburned/ningen/v2"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gtkcord3/internal/log"
)

const EmojiSize = 22

type Container struct {
	*gtk.FlowBox
	Reactions map[discord.APIEmoji]*Reaction

	state *ningen.State

	// constants
	MessageID discord.MessageID
	ChannelID discord.ChannelID
}

func NewContainer(m *discord.Message) *Container {
	f := gtk.NewFlowBox()

	gtkutils.InjectCSS(f, "reactions", "")

	container := &Container{
		FlowBox:   f,
		Reactions: map[discord.APIEmoji]*Reaction{},
		MessageID: m.ID,
		ChannelID: m.ChannelID,
	}

	for _, reaction := range m.Reactions {
		container.addReaction(reaction)
	}

	// Setting properties after adding may help?
	f.SetColumnSpacing(0) // buttons already have margins
	f.SetRowSpacing(0)
	f.SetHAlign(gtk.AlignStart)
	f.SetMaxChildrenPerLine(22)
	f.SetHomogeneous(true)
	f.Show()

	return container
}

func (c *Container) SetState(s *ningen.State) {
	c.state = s
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
	if r.MessageID != r.MessageID || r.ChannelID != c.ChannelID {
		return
	}
	glib.IdleAdd(func() {
		c.reactSomething(r.Emoji, reactAdd)
	})
}

func (c *Container) ReactRemove(r *gateway.MessageReactionRemoveEvent) {
	if r.MessageID != r.MessageID || r.ChannelID != c.ChannelID {
		return
	}
	glib.IdleAdd(func() {
		c.reactSomething(r.Emoji, reactRemove)
	})
}

// RemoveAll removes everything.
func (c *Container) RemoveAll() {
	c.removeAll(nil)
}

func (c *Container) RemoveEmoji(emoji discord.Emoji) {
	c.removeAll(&emoji)
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

type reactOp uint8

const (
	reactAdd reactOp = iota
	reactRemove
)

func (c *Container) reactSomething(emoji discord.Emoji, op reactOp) {
	if c.state == nil {
		return
	}

	// Reaction found. Do the unoptimized thing.
	m, err := c.state.Offline().Message(c.ChannelID, c.MessageID)
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

	r, ok := c.Reactions[emoji.APIString()]
	if ok {
		r.update(target)
		return
	}

	switch op {
	case reactAdd:
		if target == nil {
			log.Errorln("Can't find reaction:", emoji)
			return
		}
		// Reaction not found, add it into the message.
		c.addReaction(*target)

	case reactRemove:
		// can't do anything.
	}
}

func (c *Container) clicked(r *Reaction) {
	if r.Button.Active() {
		// Only increment the counter by the event. If react() fails, it
		// will deactivate the button.
		c.react(r)
	} else {
		// Same as above, but decrement.
		c.unreact(r)
	}
}

func (c *Container) react(r *Reaction) {
	if c.state == nil {
		return
	}

	go func() {
		if err := c.state.React(c.ChannelID, c.MessageID, r.String); err == nil {
			return
		}

		// Unactivate the button, because there won't be an event.
		glib.IdleAdd(func() { r.Button.SetActive(false) })
	}()
}

func (c *Container) unreact(r *Reaction) {
	if c.state == nil {
		return
	}

	go func() {
		if err := c.state.Unreact(c.ChannelID, c.MessageID, r.String); err == nil {
			return
		}

		// Unactivate the button, because there won't be an event.
		glib.IdleAdd(func() { r.Button.SetActive(true) })
	}()
}

type Reaction struct {
	*gtk.FlowBoxChild
	Button *gtk.ToggleButton
	Emoji  gtk.Widgetter // *gtk.Image or *gtk.Label

	String discord.APIEmoji
}

func newReaction(emoji discord.Emoji, count int, me bool) *Reaction {
	f := gtk.NewFlowBoxChild()
	f.Show()

	b := gtk.NewToggleButton()
	b.SetRelief(gtk.ReliefNone)
	b.SetActive(me)
	b.SetAlwaysShowImage(true)
	b.SetImagePosition(gtk.PosLeft)
	b.SetLabel(strconv.Itoa(count))
	b.Show()

	f.Add(b)
	gtkutils.InjectCSS(b, "reaction", "")

	reaction := &Reaction{FlowBoxChild: f, Button: b, String: emoji.APIString()}

	// If the emoji is a custom one:
	if emoji.ID.IsValid() {
		url := md.EmojiURL(emoji.ID.String(), emoji.Animated)

		i := gtk.NewImage()
		i.SetSizeRequest(EmojiSize, EmojiSize)
		i.SetVAlign(gtk.AlignCenter)
		i.SetHAlign(gtk.AlignCenter)
		i.SetMarginEnd(2)
		i.Show()
		cache.SetImageURLScaled(i, url, EmojiSize, EmojiSize)

		reaction.Emoji = i
		b.SetImage(i)

	} else {
		l := gtk.NewLabel(emoji.Name)
		l.SetSizeRequest(EmojiSize, EmojiSize)
		l.SetVAlign(gtk.AlignCenter)
		l.SetHAlign(gtk.AlignCenter)
		l.SetMarginEnd(2)
		l.Show()

		reaction.Emoji = l
		b.SetImage(l)
	}

	// Set "padding"
	if c := b.Child(); c != nil {
		gtk.BaseWidget(c).SetSizeRequest(EmojiSize+9, -1)
		gtkutils.Margin2(c, 2, 5)
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
	if r.Button.Active() != reaction.Me {
		r.Button.SetActive(reaction.Me)
	}
}
