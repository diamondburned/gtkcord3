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

	main := must(gtk.BoxNew, gtk.ORIENTATION_VERTICAL, 0).(*gtk.Box)
	i.Main = main
	i.ExtendedWidget = main

	ibox := must(gtk.BoxNew, gtk.ORIENTATION_HORIZONTAL, 0).(*gtk.Box)
	i.InputBox = ibox

	// TODO completer
	// comp, err := gtk.

	upload := must(gtk.ButtonNewFromIconName,
		"document-open-symbolic", gtk.ICON_SIZE_LARGE_TOOLBAR).(*gtk.Button)
	i.Upload = upload

	emoji := must(gtk.ButtonNewFromIconName,
		"face-smile-symbolic", gtk.ICON_SIZE_LARGE_TOOLBAR).(*gtk.Button)
	i.Emoji = emoji

	input := must(gtk.TextViewNew).(*gtk.TextView)
	i.Input = input

	ibuf := must(input.GetBuffer).(*gtk.TextBuffer)
	i.InputBuf = ibuf

	must(func() {
		style, _ := main.GetStyleContext()
		style.AddClass("message-input")

		margin2(&ibox.Widget, 8, AvatarPadding)
		upload.SetVAlign(gtk.ALIGN_START)
		upload.Connect("clicked", func() {
			go SpawnUploader(i.upload)
		})
		emoji.SetVAlign(gtk.ALIGN_START)

		margin2(&input.Widget, 6, 12)

		input.AddEvents(int(gdk.KEY_PRESS_MASK))
		input.Connect("key-press-event", i.keyDown)

		main.Add(ibox)

		ibox.PackEnd(upload, false, false, 0)
		ibox.PackEnd(input, true, true, 0)
		ibox.PackEnd(emoji, false, false, 0)

		messages.Main.PackEnd(i.Main, false, false, 0)
		messages.Main.ShowAll()
	})

	return nil
}

func (i *MessageInput) keyDown(_ *gtk.TextView, ev *gdk.Event) bool {
	evKey := gdk.EventKeyNewFromEvent(ev)
	if evKey.Type() != gdk.EVENT_KEY_PRESS {
		return false // propagate
	}

	const shiftMask = uint(gdk.GDK_SHIFT_MASK)
	shift := evKey.State()&shiftMask == shiftMask
	enter := evKey.KeyVal() == gdk.KEY_Return

	// If Enter isn't being pressed:
	if !enter {
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

	if text == "" {
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

func (i *MessageInput) makeMessage(content string) discord.Message {
	return discord.Message{
		Type:      discord.DefaultMessage,
		ChannelID: i.Messages.Channel.ID,
		GuildID:   i.Messages.Channel.Guild,
		Author:    *App.Me,
		Content:   content,
		Timestamp: discord.Timestamp(time.Now()),
		Nonce:     randString(),
	}
}

func (i *MessageInput) send(content string) error {
	// An invalid ID keeps the message invalid until it is sent.
	m := i.makeMessage(content)

	if err := i.Messages.Insert(m); err != nil {
		log.Errorln("Failed to add message to be sent:", err)
	}

	s, err := App.State.SendMessageComplex(m.ChannelID, api.SendMessageData{
		Content: m.Content,
		Nonce:   m.Nonce,
	})
	if err != nil {
		i.Messages.deleteNonce(m.Nonce)
		return errors.Wrap(err, "Failed to send message")
	}

	s.Nonce = m.Nonce
	i.Messages.Update(*s)

	return nil
}

func (i *MessageInput) upload(paths []string) {
	if err := i._upload(paths); err != nil {
		logWrap(err, "Failed to upload")
	}
}

func (i *MessageInput) _upload(paths []string) error {
	u, err := NewMessageUploader(paths)
	if err != nil {
		return err
	}

	m := i.makeMessage("")
	s := u.MakeSendData(m)

	w, err := newMessageCustom(m)
	if err != nil {
		return errors.Wrap(err, "Failed to create a message container")
	}
	must(w.rightBottom.Add, u)

	if err := i.Messages.Insert(m); err != nil {
		log.Errorln("Failed to add messages to be uploaded:", err)
	}

	n, err := App.State.SendMessageComplex(m.ChannelID, s)
	if err != nil {
		i.Messages.deleteNonce(m.Nonce)
		return errors.Wrap(err, "Failed to upload message")
	}
	s.Nonce = m.Nonce

	must(w.rightBottom.Remove, u)
	i.Messages.Update(*n)

	return nil
}
