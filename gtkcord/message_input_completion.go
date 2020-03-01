package gtkcord

import (
	"strings"
	"time"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

type Completer struct {
	*gtk.ListBox
	Entries []*CompleterEntry

	Input *MessageInput
	Start *gtk.TextIter
	End   *gtk.TextIter

	lastRequested time.Time
}

type CompleterEntry struct {
	*gtk.ListBoxRow

	Child gtkutils.ExtendedWidget
	Text  string
}

func (i *MessageInput) initCompleter() {
	if i.Completer == nil {
		l, _ := gtk.ListBoxNew()
		gtkutils.InjectCSSUnsafe(l, "completer", "")

		i.Completer = &Completer{
			ListBox: l,
			Input:   i,
		}

		l.Connect("row-activated", i.Completer.ApplyCompletion)
	}
}

func (c *Completer) requestMember(prefix string) {
	guildID := c.Input.Messages.GuildID
	if !guildID.Valid() {
		return
	}

	if time.Now().Before(c.lastRequested) {
		return
	}

	c.lastRequested = time.Now().Add(time.Second)

	go func() {
		err := App.State.Gateway.RequestGuildMembers(gateway.RequestGuildMembersData{
			GuildID:   []discord.Snowflake{guildID},
			Query:     prefix,
			Presences: true,
		})

		if err != nil {
			log.Errorln("Failed to request guild members for completion:", err)
		}
	}()
}

func (c *Completer) keyDown(state, key uint) bool {
	// Check if the key pressed is a visible letter:
	if key == gdk.KEY_space {
		c.ClearCompletion()
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
		c.ClearCompletion()
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
	case App.completionQueue <- c.run:
	default:
	}
}

func (c *Completer) run() {
	word := must(c.getWord).(string)
	if !c.IsEmpty() {
		must(c.ClearCompletion)
	}

	if word == "" {
		return
	}

	c.loadCompletion(word)
	c.ListBox.Show()

}

func (c *Completer) ClearCompletion() {
	if len(c.Entries) == 0 {
		return
	}

	for i, entry := range c.Entries {
		c.ListBox.Remove(entry)
		entry.Destroy()
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

	c.ClearCompletion()
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
	sb, ok := App.Sidebar.(*Channels)
	if !ok {
		log.Errorln("App.Sidebar is not of type *Channels")
		return
	}

	for _, ch := range sb.Channels {
		if strings.HasPrefix(ch.Name, word) {
			l := completerLeftLabel("#" + ch.Name)

			if !c.addCompletionEntry(l, "<#"+ch.ID.String()+">") {
				break
			}
		}
	}
}

func (c *Completer) completeMentions(word string) {
	if !c.Input.Messages.GuildID.Valid() {
		c.completeMentionsDM(word)
		return
	}

	members, err := App.State.Store.Members(c.Input.Messages.GuildID)
	if err != nil {
		log.Errorln("Failed to get members:", err)
		return
	}

	for _, m := range members {
		var (
			name = strings.ToLower(m.User.Username)
			nick = strings.ToLower(m.Nick)
		)

		if strings.Contains(name, word) || strings.Contains(nick, word) {
			b := must(gtk.BoxNew, gtk.ORIENTATION_HORIZONTAL, 0).(*gtk.Box)

			var name = m.Nick
			if m.Nick == "" {
				name = m.User.Username
			}

			var url = m.User.AvatarURL()
			if url != "" {
				url += "?size=64"
			}

			must(b.Add, completerImage(url))
			must(b.Add, completerLeftLabel(name))
			must(b.Add, completerRightLabel(m.User.Username+"#"+m.User.Discriminator))

			if !c.addCompletionEntry(b, m.User.Mention()) {
				break
			}
		}
	}

	if len(c.Entries) == 0 {
		// Request the member in a background goroutine
		c.requestMember(word)
	}
}

func (c *Completer) completeMentionsDM(word string) {
	ch, err := App.State.Channel(c.Input.Messages.ChannelID)
	if err != nil {
		log.Errorln("Failed to get DM channel:", err)
		return
	}

	for _, u := range ch.DMRecipients {
		var name = strings.ToLower(u.Username)
		if strings.Contains(name, word) {
			b := must(gtk.BoxNew, gtk.ORIENTATION_HORIZONTAL, 0).(*gtk.Box)

			var url = u.AvatarURL()
			if url != "" {
				url += "?size=64"
			}

			must(b.Add, completerImage(url))
			must(b.Add, completerLeftLabel(u.Username))
			must(b.Add, completerRightLabel(u.Username+"#"+u.Discriminator))

			if !c.addCompletionEntry(b, u.Mention()) {
				break
			}
		}
	}
}

func completerImage(url string) *gtk.Image {
	i := must(gtk.ImageNewFromIconName,
		"dialog-question-symbolic", gtk.ICON_SIZE_LARGE_TOOLBAR).(*gtk.Image)
	if url != "" {
		asyncFetch(url, i, 24, 24, cache.Round)
	}

	must(i.SetMarginEnd, AvatarPadding)

	return i
}

func completerLeftLabel(markup string) *gtk.Label {
	l, _ := must(gtk.LabelNew, "").(*gtk.Label)
	must(func() {
		l.SetMarkup(markup)
		l.SetSingleLineMode(true)
		l.SetLineWrap(true)
		l.SetLineWrapMode(pango.WRAP_WORD_CHAR)
		l.SetHAlign(gtk.ALIGN_START)
	})

	return l
}

func completerRightLabel(markup string) *gtk.Label {
	l := completerLeftLabel(markup)
	must(func() {
		l.SetOpacity(0.65)
		l.SetHExpand(true)
		l.SetHAlign(gtk.ALIGN_END)
	})

	return l
}

func (c *Completer) addCompletionEntry(w gtkutils.ExtendedWidget, text string) bool {
	if len(c.Entries) > 15 {
		return false
	}

	entry := &CompleterEntry{
		Child: w,
		Text:  text,
	}

	must(func() {
		if w, ok := w.(gtkutils.Marginator); ok {
			w.SetMarginStart(AvatarPadding + 5)
			w.SetMarginEnd(AvatarPadding + 5)
		}

		entry.ListBoxRow, _ = gtk.ListBoxRowNew()
		entry.ListBoxRow.Add(w)

		c.ListBox.Insert(entry, -1)
		entry.ShowAll()
	})

	c.Entries = append(c.Entries, entry)
	return true
}
