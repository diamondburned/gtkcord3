package gtkcord

import (
	"time"

	"github.com/diamondburned/arikawa/api"
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

type MessageInput struct {
	ExtendedWidget
	Messages *Messages

	Main *gtk.Box

	InputBox   *gtk.Box
	Completion *Completer

	Input    *gtk.TextView
	InputBuf *gtk.TextBuffer
	Upload   *gtk.Button
	Emoji    *gtk.Button

	// uploadButton
	// emojiButton
}

type Completer struct {
	*gtk.ListBox
	Entries []*CompleterEntry
}

type CompleterEntry struct {
	*gtk.ListBoxRow
	Icon   *gtk.Image
	Pixbuf *Pixbuf

	Left  *gtk.Label
	Right *gtk.Label
}

func (messages *Messages) loadMessageInput() error {
	if messages.Input == nil {
		messages.Input = &MessageInput{
			Messages: messages,
		}
	}

	i := messages.Input

	if i.Input != nil {
		return nil
	}

	main, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	if err != nil {
		return errors.Wrap(err, "Failed to create main box")
	}
	i.Main = main
	i.ExtendedWidget = main

	ibox, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	if err != nil {
		return errors.Wrap(err, "Failed to create input box")
	}
	i.InputBox = ibox

	// TODO completer
	// comp, err := gtk.

	upload, err := gtk.ButtonNewFromIconName(
		"document-open-symbolic", gtk.ICON_SIZE_LARGE_TOOLBAR)
	if err != nil {
		return errors.Wrap(err, "Failed to create upload button")
	}
	i.Upload = upload

	emoji, err := gtk.ButtonNewFromIconName(
		"face-smile-symbolic", gtk.ICON_SIZE_LARGE_TOOLBAR)
	if err != nil {
		return errors.Wrap(err, "Failed to create emoji button")
	}
	i.Emoji = emoji

	ttt, err := gtk.TextTagTableNew()
	if err != nil {
		return errors.Wrap(err, "Faield to create a text tag table")
	}

	ibuf, err := gtk.TextBufferNew(ttt)
	if err != nil {
		return errors.Wrap(err, "Failed to create input buffer")
	}
	i.InputBuf = ibuf

	must(func() {
		input, err := gtk.TextViewNewWithBuffer(ibuf)
		if err != nil {
			log.Panicln("Die: " + err.Error())
		}
		i.Input = input

		input.AddEvents(int(gdk.KEY_PRESS_MASK))
		input.Connect("key-press-event", i.keyDown)

		main.Add(ibox)

		ibox.PackEnd(upload, false, false, 0)
		ibox.PackEnd(input, true, true, 0)
		ibox.PackEnd(emoji, false, false, 0)

		messages.Main.PackEnd(i.Main, false, false, 0)
	})

	return nil
}

func (i *MessageInput) keyDown(_ *gtk.TextView, ev *gdk.Event) bool {
	evKey := gdk.EventKeyNewFromEvent(ev)
	if evKey.Type() != gdk.EVENT_KEY_PRESS {
		return false // propagate
	}

	shift := evKey.State() == uint(gdk.GDK_SHIFT_MASK)
	enter := evKey.KeyVal() == gdk.KEY_KP_Enter

	log.Println("Got shift/enter:", shift, enter)

	if !shift || !enter {
		return false // propagate
	}

	// If Shift is being held:
	if shift {
		// Insert a new line
		i.InputBuf.InsertAtCursor("\n")
		return true
	}

	var (
		iStart = i.InputBuf.GetStartIter()
		iEnd   = i.InputBuf.GetEndIter()
	)

	text, err := i.InputBuf.GetText(iStart, iEnd, true)
	if err != nil {
		log.Errorln("Failed to get chatbox text buffer:", err)
		return true
	}

	i.InputBuf.Delete(iStart, iEnd)

	// Shift is not being held, send the message:
	semaphore.Go(func() {
		if err := i.send(text); err != nil {
			log.Println("Failed to send message:", err)
		}
	})

	return true
}

func (i *MessageInput) send(content string) error {
	// An invalid ID keeps the message invalid until it is sent.
	m := discord.Message{
		Type:      discord.DefaultMessage,
		ChannelID: i.Messages.Channel.ID,
		GuildID:   i.Messages.Channel.Channels.Guild.ID,
		Author:    *App.Me,
		Content:   content,
		Timestamp: discord.Timestamp(time.Now()),
		Nonce:     randString(),
	}

	if err := i.Messages.Insert(m); err != nil {
		log.Errorln("Failed to add message to be sent:", err)
	}

	_, err := App.State.SendMessageComplex(m.ChannelID, api.SendMessageData{
		Content: m.Content,
		Nonce:   m.Nonce,
	})
	if err != nil {
		i.Messages.deleteNonce(m.Nonce)
		return errors.Wrap(err, "Failed to send message")
	}

	return nil
}
