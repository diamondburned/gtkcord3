package completer

import (
	"strings"
	"time"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/diamondburned/gtkcord3/internal/moreatomic"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

const MaxCompletionEntries = 10

var completionQueue chan func()

func init() {
	completionQueue = make(chan func(), 1)
	go func() {
		for fn := range completionQueue {
			fn()
		}
	}()
}

type State struct {
	*gtk.Revealer

	Scroll  *gtk.ScrolledWindow
	ListBox *gtk.ListBox
	Entries []*Entry

	state *ningen.State

	InputBuf  *gtk.TextBuffer
	container MessageContainer

	start *gtk.TextIter
	end   *gtk.TextIter

	lastRequested time.Time
	lastWord      moreatomic.String

	channels []discord.Channel
	members  []discord.Member
	users    []discord.User
	// emojis   []discord.Emoji
}

type Entry struct {
	*gtk.ListBoxRow

	Child gtkutils.ExtendedWidget
	Text  string
}

type MessageContainer interface {
	GetChannelID() discord.ChannelID
	GetGuildID() discord.GuildID
	GetRecentAuthorsUnsafe(limit int) []discord.UserID
}

func New(state *ningen.State, textbuf *gtk.TextBuffer, msgC MessageContainer) *State {
	revealer, _ := gtk.RevealerNew()
	revealer.Show()
	revealer.SetRevealChild(false)
	revealer.SetTransitionType(gtk.REVEALER_TRANSITION_TYPE_NONE)

	scroll, _ := gtk.ScrolledWindowNew(nil, nil)
	scroll.Show()
	scroll.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_AUTOMATIC)
	scroll.SetProperty("propagate-natural-height", true)
	scroll.SetProperty("min-content-height", 0)
	scroll.SetProperty("max-content-height", 250) // arbitrary height
	scroll.SetSizeRequest(-1, 250)

	listbox, _ := gtk.ListBoxNew()
	listbox.Show()
	listbox.SetAdjustment(scroll.GetVAdjustment())
	listbox.SetFocusVAdjustment(scroll.GetVAdjustment())
	gtkutils.InjectCSSUnsafe(listbox, "completer", "")

	s := &State{
		Revealer:  revealer,
		Scroll:    scroll,
		ListBox:   listbox,
		state:     state,
		InputBuf:  textbuf,
		container: msgC,
	}
	revealer.Add(scroll)
	scroll.Add(listbox)
	listbox.Connect("row-activated", s.applyCompletion)

	textbuf.Connect("changed", func() {
		// Run the autocompleter.
		s.run()
	})

	return s
}

func (c *State) KeyDown(state, key uint) bool {
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

func (c *State) GetIndex() int {
	r := c.ListBox.GetSelectedRow()
	i := 0

	if r != nil {
		i = r.GetIndex()
	} else {
		c.Select(i)
	}

	return i
}

func (c *State) Down() {
	i := c.GetIndex()
	i++
	if i >= len(c.Entries) {
		i = 0
	}
	c.Select(i)
}

func (c *State) Up() {
	i := c.GetIndex()
	i--
	if i < 0 {
		i = len(c.Entries) - 1
	}
	c.Select(i)
}

func (c *State) getWord() string {
	mark := c.InputBuf.GetInsert()
	iter := c.InputBuf.GetIterAtMark(mark)

	// Seek backwards for space or start-of-line:
	_, start, ok := iter.BackwardSearch(" ", gtk.TEXT_SEARCH_TEXT_ONLY, nil)
	if !ok {
		start = c.InputBuf.GetStartIter()
	}

	// Seek forwards for space or end-of-line:
	_, end, ok := iter.ForwardSearch(" ", gtk.TEXT_SEARCH_TEXT_ONLY, nil)
	if !ok {
		end = c.InputBuf.GetEndIter()
	}

	c.start = start
	c.end = end

	// Get word:
	return start.GetText(end)
}

func (c *State) run() {
	word := c.getWord()
	if word == c.lastWord.Get() {
		return
	}
	c.lastWord.Set(word)

	select {
	case completionQueue <- c.execute:
	default:
		<-completionQueue
		completionQueue <- c.execute
	}
}

func (c *State) execute() {
	var word = c.lastWord.Get()

	if word == "" {
		// Clear the completion if there's no word.
		c.ClearCompletion()
		return
	}

	if !c.IsEmpty() && len(c.Entries) > 0 {
		// Clear completion without hiding:
		semaphore.IdleMust(c.clearCompletion)
	}

	c.loadCompletion(word)

	// Reveal (true) if c.Entries is not empty.
	if len(c.Entries) > 0 {
		semaphore.IdleMust(func() {
			c.SetRevealChild(true)
		})
	}
}

func (c *State) ClearCompletion() {
	if len(c.Entries) == 0 {
		return
	}

	semaphore.IdleMust(func() {
		c.clearCompletion()
		c.SetRevealChild(false)
	})
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
	r := c.ListBox.GetSelectedRow()
	i := r.GetIndex()
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

func completerImage(url string, pp ...cache.Processor) *gtk.Image {
	i, _ := gtk.ImageNew()
	i.SetMarginEnd(10)
	i.SetSizeRequest(24, 24)
	gtkutils.ImageSetIcon(i, "dialog-question-symbolic", 24)

	if url != "" {
		go cache.AsyncFetch(url, i, 24, 24, pp...)
	}

	return i
}

func completerLeftLabel(text string) *gtk.Label {
	l, _ := gtk.LabelNew(text)
	l.SetSingleLineMode(true)
	l.SetLineWrap(false)
	l.SetEllipsize(pango.ELLIPSIZE_END)
	l.SetHAlign(gtk.ALIGN_START)

	return l
}

func completerRightLabel(text string) *gtk.Label {
	l := completerLeftLabel(text)
	l.SetOpacity(0.65)
	l.SetHExpand(true)
	l.SetHAlign(gtk.ALIGN_END)

	return l
}

func (c *State) addCompletionEntry(w gtkutils.ExtendedWidget, text string) bool {
	if len(c.Entries) > MaxCompletionEntries {
		return false
	}

	entry := &Entry{
		Child: w,
		Text:  text,
	}

	if w, ok := w.(gtkutils.Marginator); ok {
		w.SetMarginStart(20)
		w.SetMarginEnd(20)
	}

	entry.ListBoxRow, _ = gtk.ListBoxRowNew()
	entry.ListBoxRow.Add(w)
	entry.ListBoxRow.SetFocusVAdjustment(c.Scroll.GetVAdjustment())

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
