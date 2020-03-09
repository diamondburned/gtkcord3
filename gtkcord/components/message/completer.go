package message

import (
	"strings"
	"sync"
	"time"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

const MaxCompletionEntries = 15

var completionQueue chan func()
var cQueueOnce sync.Once

func initCQueue() {
	cQueueOnce.Do(func() {
		completionQueue = make(chan func(), 1)
		go func() {
			for fn := range completionQueue {
				fn()
			}
		}()
	})
}

type Completer struct {
	*gtk.ListBox
	Entries []*CompleterEntry

	RequestGuildMember func(prefix string)

	Input *Input
	Start *gtk.TextIter
	End   *gtk.TextIter

	lastRequested time.Time
	lastword      string

	channels []discord.Channel
	members  []discord.Member
	users    []discord.User
}

type CompleterEntry struct {
	*gtk.ListBoxRow

	Child gtkutils.ExtendedWidget
	Text  string
}

func (i *Input) initCompleter() {
	if i.Completer == nil {
		initCQueue()

		l, _ := gtk.ListBoxNew()
		gtkutils.InjectCSSUnsafe(l, "completer", "")

		i.Completer = &Completer{
			ListBox: l,
			Input:   i,
		}

		l.Connect("row-activated", i.Completer.ApplyCompletion)
	}
}

func (c *Completer) keyDown(state, key uint) bool {
	// Check if the key pressed is a visible letter:
	if key == gdk.KEY_space {
		if !c.IsEmpty() {
			c.clearCompletion()
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

	// Run the autocompleter:
	c.Run()

	if c.IsEmpty() {
		return false
	}

	if key == gdk.KEY_Escape {
		c.clearCompletion()
		return true
	}

	if key == gdk.KEY_Return || key == gdk.KEY_Tab {
		c.ApplyCompletion()
		return true
	}

	return false
}

func (c *Completer) IsEmpty() bool {
	return len(c.Entries) == 0
}

func (c *Completer) Select(index int) {
	c.ListBox.SelectRow(c.Entries[index].ListBoxRow)
}

func (c *Completer) GetIndex() int {
	r := c.ListBox.GetSelectedRow()
	i := 0

	if r != nil {
		i = r.GetIndex()
	} else {
		c.Select(i)
	}

	return i
}

func (c *Completer) Down() {
	i := c.GetIndex()
	i++
	if i >= len(c.Entries) {
		i = 0
	}
	c.Select(i)
}

func (c *Completer) Up() {
	i := c.GetIndex()
	i--
	if i <= 0 {
		i = len(c.Entries) - 1
	}
	c.Select(i)
}

func (c *Completer) getWord() string {
	mark := c.Input.InputBuf.GetInsert()
	iter := c.Input.InputBuf.GetIterAtMark(mark)

	// Seek backwards for space or start-of-line:
	_, start, ok := iter.BackwardSearch(" ", gtk.TEXT_SEARCH_TEXT_ONLY, nil)
	if !ok {
		start = c.Input.InputBuf.GetStartIter()
	}

	// Seek forwards for space or end-of-line:
	_, end, ok := iter.ForwardSearch(" ", gtk.TEXT_SEARCH_TEXT_ONLY, nil)
	if !ok {
		end = c.Input.InputBuf.GetEndIter()
	}

	c.Start = start
	c.End = end

	// Get word:
	return start.GetText(end)
}

func (c *Completer) Run() {
	select {
	case completionQueue <- c.run:
	default:
		<-completionQueue
		completionQueue <- c.run
	}
}

func (c *Completer) run() {
	word := semaphore.IdleMust(c.getWord).(string)
	if !c.IsEmpty() {
		c.ClearCompletion()
	}

	if word == c.lastword {
		return
	}
	c.lastword = word

	if word == "" {
		return
	}

	c.loadCompletion(word)
	semaphore.IdleMust(c.ListBox.Show)
}

func (c *Completer) ClearCompletion() {
	if len(c.Entries) == 0 {
		return
	}

	semaphore.IdleMust(c.clearCompletion)
}

func (c *Completer) clearCompletion() {
	for i, entry := range c.Entries {
		c.ListBox.Remove(entry)
		c.Entries[i] = nil
	}
	c.Entries = c.Entries[:0]

	c.ListBox.Hide()
}

// Finalizing function
func (c *Completer) ApplyCompletion() {
	r := c.ListBox.GetSelectedRow()
	if r == nil {
		r = c.Entries[0].ListBoxRow
		c.ListBox.SelectRow(r)
	}

	i := r.GetIndex()
	if i < 0 || i >= len(c.Entries) {
		log.Errorln("Index out of bounds:", i)
		return
	}

	if c.Start == nil || c.End == nil {
		log.Errorln("c.Start/c.End nil")
		return
	}

	c.Input.InputBuf.Delete(c.Start, c.End)
	c.Input.InputBuf.Insert(c.Start, c.Entries[i].Text+" ")

	c.clearCompletion()
}

func (c *Completer) loadCompletion(word string) {
	// We don't want to check with an empty string:
	if len(word) < 2 {
		return
	}

	switch word[0] {
	case '@':
		c.completeMentions(strings.ToLower(word[1:]))
	case '#':
		c.completeChannels(word[1:])
	case ':':
	}
}

func (c *Completer) completeChannels(word string) {
	guildID := c.Input.Messages.GuildID
	if !guildID.Valid() {
		return
	}

	chs, err := c.Input.Messages.c.State.Channels(guildID)
	if err != nil {
		log.Errorln("Failed to get channels:", err)
		return
	}

	c.channels = c.channels[:0]

	for _, ch := range chs {
		if strings.HasPrefix(ch.Name, word) {
			c.channels = append(c.channels, ch)

			if len(c.channels) > MaxCompletionEntries {
				break
			}
		}
	}

	if len(c.channels) == 0 {
		return
	}

	semaphore.IdleMust(func() {
		for _, ch := range c.channels {
			l := completerLeftLabel("#" + ch.Name)
			c.addCompletionEntry(l, "<#"+ch.ID.String()+">")
		}
	})
}

func (c *Completer) completeMentions(word string) {
	guildID := c.Input.Messages.GuildID
	if !guildID.Valid() {
		c.completeMentionsDM(word)
		return
	}

	members, err := c.Input.Messages.c.Store.Members(guildID)
	if err != nil {
		log.Errorln("Failed to get members:", err)
		return
	}

	c.members = c.members[:0]

	for i, m := range members {
		var (
			name = strings.ToLower(m.User.Username)
			nick = strings.ToLower(m.Nick)
		)

		if strings.Contains(name, word) || strings.Contains(nick, word) {
			c.members = append(c.members, members[i])

			if len(c.members) > MaxCompletionEntries {
				break
			}
		}
	}

	if len(c.members) == 0 {
		// Request the member in a background goroutine
		c.Input.Messages.c.SearchMember(c.Input.Messages.GuildID, word)
		return
	}

	semaphore.IdleMust(func() {
		for _, m := range c.members {
			b, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)

			var name = m.Nick
			if m.Nick == "" {
				name = m.User.Username
			}

			var url = m.User.AvatarURL()
			if url != "" {
				url += "?size=64"
			}

			b.Add(completerImage(url))
			b.Add(completerLeftLabel(name))
			b.Add(completerRightLabel(m.User.Username + "#" + m.User.Discriminator))
			c.addCompletionEntry(b, m.User.Mention())
		}
	})

}

func (c *Completer) completeMentionsDM(word string) {
	ch, err := c.Input.Messages.c.Channel(c.Input.Messages.ChannelID)
	if err != nil {
		log.Errorln("Failed to get DM channel:", err)
		return
	}

	c.users = c.users[:0]

	for i, u := range ch.DMRecipients {
		var name = strings.ToLower(u.Username)
		if strings.Contains(name, word) {
			c.users = append(c.users, ch.DMRecipients[i])

			if len(c.users) > MaxCompletionEntries {
				break
			}
		}
	}

	if len(c.users) == 0 {
		return
	}

	semaphore.IdleMust(func() {
		for _, u := range c.users {
			b, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)

			var url = u.AvatarURL()
			if url != "" {
				url += "?size=64"
			}

			b.Add(completerImage(url))
			b.Add(completerLeftLabel(u.Username))
			b.Add(completerRightLabel(u.Username + "#" + u.Discriminator))
			c.addCompletionEntry(b, u.Mention())
		}
	})
}

func completerImage(url string) *gtk.Image {
	i, _ := gtk.ImageNewFromIconName(
		"dialog-question-symbolic", gtk.ICON_SIZE_LARGE_TOOLBAR)
	i.SetMarginEnd(10)

	if url != "" {
		go cache.AsyncFetch(url, i, 24, 24, cache.Round)
	}

	return i
}

func completerLeftLabel(markup string) *gtk.Label {
	l, _ := gtk.LabelNew("")
	l.SetMarkup(markup)
	l.SetSingleLineMode(true)
	l.SetLineWrap(true)
	l.SetLineWrapMode(pango.WRAP_WORD_CHAR)
	l.SetHAlign(gtk.ALIGN_START)

	return l
}

func completerRightLabel(markup string) *gtk.Label {
	l := completerLeftLabel(markup)
	l.SetOpacity(0.65)
	l.SetHExpand(true)
	l.SetHAlign(gtk.ALIGN_END)

	return l
}

func (c *Completer) addCompletionEntry(w gtkutils.ExtendedWidget, text string) bool {
	if len(c.Entries) > MaxCompletionEntries {
		return false
	}

	entry := &CompleterEntry{
		Child: w,
		Text:  text,
	}

	if w, ok := w.(gtkutils.Marginator); ok {
		w.SetMarginStart(10 + 5)
		w.SetMarginEnd(10 + 5)
	}

	entry.ListBoxRow, _ = gtk.ListBoxRowNew()
	entry.ListBoxRow.Add(w)

	c.ListBox.Insert(entry, -1)
	entry.ShowAll()

	c.Entries = append(c.Entries, entry)
	return true
}
