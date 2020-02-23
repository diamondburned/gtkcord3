package gtkcord

import (
	"strings"

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

	queue chan struct{}
}

type CompleterEntry struct {
	*gtk.ListBoxRow

	Child gtkutils.ExtendedWidget
	Text  string
}

func (i *MessageInput) initCompleter() {
	if i.Completer == nil {
		l, _ := gtk.ListBoxNew()
		gtkutils.InjectCSSUnsafe(l, "completer", `
		.completer {
			background-color: transparent;
		}
	`)

		i.Completer = &Completer{
			ListBox: l,
			Input:   i,
		}

		l.Connect("row-activated", i.Completer.ApplyCompletion)
	}

	if i.Completer.queue != nil {
		return
	}

	i.Completer.queue = make(chan struct{})

	go func() {
		c := i.Completer

		for range c.queue {
			word := must(c.getWord).(string)
			must(c.ClearCompletion)

			if word == "" {
				continue
			}

			i.Completer.loadCompletion(word)
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

	if key == gdk.KEY_Return {
		c.ApplyCompletion()
		return true
	}

	return false
}

func (c *Completer) Close() {
	close(c.queue)
	c.queue = nil

	c.ClearCompletion()
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
	start, _, ok := iter.BackwardSearch(" ", gtk.TEXT_SEARCH_TEXT_ONLY, nil)
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
	case c.queue <- struct{}{}:
	default:
	}
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
}

// Finalizing function
func (c *Completer) ApplyCompletion() {
	r := c.ListBox.GetSelectedRow()
	if r == nil {
		c.ClearCompletion()
		return
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
			c.addCompletionEntry(l, "<#"+ch.ID.String()+">")
		}
	}
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

func (c *Completer) addCompletionEntry(w gtkutils.ExtendedWidget, text string) {
	entry := &CompleterEntry{
		Child: w,
		Text:  text,
	}

	must(func() {
		if w, ok := w.(gtkutils.Marginator); ok {
			w.SetMarginStart(AvatarPadding + 12 + 24) // 24px is LARGE_TOOLBAR
			w.SetMarginEnd(AvatarPadding + 12 + 24)
		}

		entry.ListBoxRow, _ = gtk.ListBoxRowNew()
		entry.ListBoxRow.Add(w)

		c.ListBox.Insert(entry, -1)
		entry.ShowAll()
	})

	c.Entries = append(c.Entries, entry)
}
