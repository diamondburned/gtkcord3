package completer

import (
	"strings"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/gotk4/pkg/gdk/v3"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gotk4/pkg/pango"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/components/roundimage"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/diamondburned/ningen/v2"
)

const MaxCompletionEntries = 10

type State struct {
	*gtk.Revealer

	Scroll  *gtk.ScrolledWindow
	ListBox *gtk.ListBox
	Entries []*Entry

	// state is offline
	state *ningen.State

	// RequestGuildMember func(prefix string)
	GetRecentAuthors func(limit int) []discord.Snowflake
	InputBuf         *gtk.TextBuffer

	// guildID   *discord.Snowflake
	// channelID *discord.Snowflake

	container MessageContainer

	start *gtk.TextIter
	end   *gtk.TextIter

	lastWord string

	channels []discord.Channel
	members  []discord.Member
	users    []discord.User
	// emojis   []discord.Emoji
}

type Entry struct {
	*gtk.ListBoxRow

	Child gtk.Widgetter
	Text  string
}

type MessageContainer interface {
	ChannelID() discord.ChannelID
	GuildID() discord.GuildID
	RecentAuthors(limit int) []discord.UserID
}

func New(state *ningen.State, textbuf *gtk.TextBuffer, msgC MessageContainer) *State {
	revealer := gtk.NewRevealer()
	revealer.SetRevealChild(false)
	revealer.SetTransitionType(gtk.RevealerTransitionTypeSlideUp)

	scroll := gtk.NewScrolledWindow(nil, nil)
	scroll.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	scroll.SetPropagateNaturalHeight(true)
	scroll.SetMinContentHeight(0)
	scroll.SetMaxContentHeight(250) // arbitrary height
	scroll.SetSizeRequest(-1, 250)

	listbox := gtk.NewListBox()
	listbox.SetAdjustment(scroll.VAdjustment())
	listbox.SetFocusVAdjustment(scroll.VAdjustment())
	gtkutils.InjectCSS(listbox, "completer", "")

	s := &State{
		Revealer:  revealer,
		Scroll:    scroll,
		ListBox:   listbox,
		state:     state.Offline(),
		InputBuf:  textbuf,
		container: msgC,
	}
	revealer.Add(scroll)
	scroll.Add(listbox)
	scroll.ShowAll()
	listbox.Connect("row-activated", s.applyCompletion)

	textbuf.Connect("changed", func() {
		// Run the autocompleter.
		s.run()
	})

	return s
}

func (c *State) KeyDown(state gdk.ModifierType, key uint) bool {
	// Check if the key pressed is a visible letter:
	if key == gdk.KEY_space {
		if !c.IsEmpty() {
			c.clearCompletion()
			c.Revealer.SetRevealChild(false)
		}

		return false
	}

	if !c.IsEmpty() {
		switch key {
		case gdk.KEY_Up:
			c.Up()
			return true
		case gdk.KEY_Down:
			c.Down()
			return true
		}
	}

	if c.IsEmpty() {
		return false
	}

	if key == gdk.KEY_Escape {
		c.clearCompletion()
		c.Revealer.SetRevealChild(false)
		return true
	}

	if key == gdk.KEY_Return || key == gdk.KEY_Tab {
		c.applyCompletion()
		return true
	}

	return false
}

func (c *State) IsEmpty() bool {
	return len(c.Entries) == 0
}

func (c *State) Select(index int) {
	c.ListBox.SelectRow(c.Entries[index].ListBoxRow)
}

func (c *State) Index() int {
	r := c.ListBox.SelectedRow()
	i := 0

	if r != nil {
		i = r.Index()
	} else {
		c.Select(i)
	}

	return i
}

func (c *State) Down() {
	i := c.Index()
	i++
	if i >= len(c.Entries) {
		i = 0
	}
	c.Select(i)
}

func (c *State) Up() {
	i := c.Index()
	i--
	if i < 0 {
		i = len(c.Entries) - 1
	}
	c.Select(i)
}

func (c *State) getWord() string {
	mark := c.InputBuf.GetInsert()
	iter := c.InputBuf.IterAtMark(mark)

	// Seek backwards for space or start-of-line:
	_, start, ok := iter.BackwardSearch(" ", gtk.TextSearchTextOnly, nil)
	if !ok {
		start = c.InputBuf.StartIter()
	}

	// Seek forwards for space or end-of-line:
	_, end, ok := iter.ForwardSearch(" ", gtk.TextSearchTextOnly, nil)
	if !ok {
		end = c.InputBuf.EndIter()
	}

	c.start = start
	c.end = end

	// Get word:
	return start.Text(end)
}

func (c *State) run() {
	word := c.getWord()
	if word == c.lastWord {
		return
	}

	c.lastWord = word
	c.execute()
}

func (c *State) execute() {
	word := c.lastWord
	if word == "" {
		// Clear the completion if there's no word.
		c.ClearCompletion()
		return
	}

	if !c.IsEmpty() && len(c.Entries) > 0 {
		// Clear completion without hiding:
		c.clearCompletion()
	}

	c.loadCompletion(word)

	// Reveal (true) if c.Entries is not empty.
	c.SetRevealChild(len(c.Entries) > 0)
}

func (c *State) ClearCompletion() {
	if len(c.Entries) == 0 {
		return
	}

	c.clearCompletion()
	c.SetRevealChild(false)
}

func (c *State) clearCompletion() {
	for _, entry := range c.Entries {
		c.ListBox.Remove(entry)
	}
	c.Entries = nil

	c.channels = nil
	c.members = nil
	c.users = nil
	// c.emojis = c.emojis[:0]

	// c.ScrolledWindow.Hide()
}

// Finalizing function
func (c *State) applyCompletion() {
	r := c.ListBox.SelectedRow()
	i := r.Index()
	if i < 0 || i >= len(c.Entries) {
		log.Errorln("Index out of bounds:", i)
		return
	}

	if c.start == nil || c.end == nil {
		log.Errorln("c.Start/c.End nil")
		return
	}

	c.InputBuf.Delete(c.start, c.end)
	c.InputBuf.Insert(c.start, c.Entries[i].Text+" ")

	c.clearCompletion()
	c.SetRevealChild(false)
}

func (c *State) loadCompletion(word string) {
	switch word[0] {
	case '@':
		c.completeMentions(strings.ToLower(word[1:]))
	case '#':
		c.completeChannels(word[1:])
	case ':':
		c.completeEmojis(strings.ToLower(word[1:]))
	}
}

func completerImage(url string) *roundimage.Image {
	i := roundimage.NewImage(0)
	i.SetMarginEnd(10)
	i.SetSizeRequest(24, 24)
	i.SetFromIconName("dialog-question-symbolic", 0)
	i.SetPixelSize(24)

	if url != "" {
		cache.SetImageURLScaled(i, url, 24, 24)
	}

	return i
}

func completerLeftLabel(text string) *gtk.Label {
	l := gtk.NewLabel(text)
	l.SetSingleLineMode(true)
	l.SetLineWrap(false)
	l.SetEllipsize(pango.EllipsizeEnd)
	l.SetHAlign(gtk.AlignStart)

	return l
}

func completerRightLabel(text string) *gtk.Label {
	l := completerLeftLabel(text)
	l.SetOpacity(0.65)
	l.SetHExpand(true)
	l.SetHAlign(gtk.AlignEnd)

	return l
}

func (c *State) addCompletionEntry(w gtk.Widgetter, text string) bool {
	if len(c.Entries) > MaxCompletionEntries {
		return false
	}

	entry := &Entry{
		Child: w,
		Text:  text,
	}

	w.BaseWidget().SetMarginStart(20)
	w.BaseWidget().SetMarginEnd(20)

	entry.ListBoxRow = gtk.NewListBoxRow()
	entry.ListBoxRow.Add(w)
	entry.ListBoxRow.SetFocusVAdjustment(c.Scroll.VAdjustment())

	c.ListBox.Insert(entry, -1)
	entry.ShowAll()

	if len(c.Entries) == 0 {
		c.ListBox.SelectRow(entry.ListBoxRow)
	}

	c.Entries = append(c.Entries, entry)
	return true
}

// match is assumed to already be lower-cased
func contains(full, match string) bool {
	return strings.Contains(strings.ToLower(full), match)
}
