package gtkcord

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/diamondburned/arikawa/api"
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/window"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

type MessageInput struct {
	gtkutils.ExtendedWidget
	Messages *Messages

	Main  *gtk.Box
	Style *gtk.StyleContext

	InputBox  *gtk.Box
	Completer *Completer

	Input    *gtk.TextView
	InputBuf *gtk.TextBuffer
	Upload   *gtk.Button
	Emoji    *gtk.Button

	Editing *discord.Message

	// uploadButton
	// emojiButton
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

	style, _ := must(main.GetStyleContext).(*gtk.StyleContext)
	i.Style = style

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
		style.AddClass("message-input")

		// Initialize the completer:
		i.initCompleter()
		// Prepend the completer box:
		main.Add(i.Completer)

		// Prepare the message input box:
		margin2(ibox, 8, AvatarPadding)
		upload.SetVAlign(gtk.ALIGN_START)
		upload.Connect("clicked", func() {
			go SpawnUploader(i.upload)
		})
		emoji.SetVAlign(gtk.ALIGN_START)

		margin2(input, 6, 12)

		input.AddEvents(int(gdk.KEY_PRESS_MASK))
		input.Connect("key-press-event", i.keyDown)
		input.SetWrapMode(gtk.WRAP_WORD_CHAR)

		// Add the message input box:
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
		return false
	}

	const shiftMask = uint(gdk.GDK_SHIFT_MASK)
	const cntrlMask = uint(gdk.GDK_CONTROL_MASK)

	var (
		state = evKey.State()
		key   = evKey.KeyVal()
	)

	if i.Completer.keyDown(state, key) {
		return true
	}

	var (
		cntrl = state&cntrlMask == cntrlMask
		vkey  = key == gdk.KEY_v
	)

	// If Ctrl-V is pressed:
	if cntrl && vkey && window.Window.Clipboard != nil {
		clipboard := window.Window.Clipboard

		// Is there an image in the clipboard?
		if !clipboard.WaitIsImageAvailable() {
			// No.
			return false
		}
		// Yes.

		p, err := clipboard.WaitForImage()
		if err != nil {
			log.Errorln("Failed to get image from clipboard:", err)
			return false
		}
		text := i.popContent()

		go func() {
			if err := i.paste(text, p); err != nil {
				log.Errorln("Failed to paste message:", err)
			}
		}()

		return true
	}

	var esc = key == gdk.KEY_Escape

	// If escape key is pressed and we're editing something:
	if esc && i.Editing != nil {
		// Clear the text box:
		i.InputBuf.Delete(i.InputBuf.GetStartIter(), i.InputBuf.GetEndIter())

		// Reset state:
		i.Editing = nil
		i.Style.RemoveClass("editing")

		return true
	}

	var upArr = key == gdk.KEY_Up

	// If arrow up is pressed and the input box is empty:
	if upArr && i.getContent() == "" {
		// Try and look backwards for the latest message:
		var latest discord.Snowflake

		for n := len(i.Messages.messages) - 1; n >= 0; n-- {
			if msg := i.Messages.messages[n]; msg.AuthorID == App.Me.ID {
				latest = msg.ID
				break
			}
		}

		// If we can find the message:
		if latest.Valid() {
			// Trigger the edit message:
			go func() {
				if err := i.editMessage(latest); err != nil {
					log.Errorln("Failed to edit messages:", err)
				}
			}()
		}

		return true
	}

	var (
		shift = state&shiftMask == shiftMask
		enter = key == gdk.KEY_Return
	)

	// If Enter isn't being pressed:
	if !enter {
		return false // propagate
	}

	// Check if the numbers of ``` are odd.
	if strings.Count(i.getContent(), "```")%2 > 0 {
		// If yes, assume shift is held. We want the Enter key to insert new
		// lines.
		shift = true
	}

	// If Shift is being held:
	if shift {
		// Insert a new line
		i.InputBuf.InsertAtCursor("\n")
		return true
	}

	text := i.popContent()

	// Shift is not being held, send the message:
	go func() {
		if err := i.send(text); err != nil {
			log.Errorln("Failed to paste message:", err)
		}
	}()

	return true
}

func (i *MessageInput) editMessage(id discord.Snowflake) error {
	m, err := App.State.Store.Message(i.Messages.ChannelID, id)
	if err != nil {
		return errors.Wrap(err, "Failed to get message")
	}

	i.Editing = m
	must(i.Style.AddClass, "editing")

	must(i.InputBuf.SetText, i.Editing.Content)
	return nil
}

func (i *MessageInput) getContent() string {
	var iStart, iEnd = i.InputBuf.GetBounds()

	text, err := i.InputBuf.GetText(iStart, iEnd, true)
	if err != nil {
		log.Errorln("Failed to get chatbox text buffer:", err)
		return ""
	}

	return text
}

// popContent gets the current messages and deletes the buffer.
func (i *MessageInput) popContent() string {
	var iStart, iEnd = i.InputBuf.GetBounds()

	text, err := i.InputBuf.GetText(iStart, iEnd, true)
	if err != nil {
		log.Errorln("Failed to get chatbox text buffer:", err)
		return ""
	}

	if text == "" {
		return ""
	}

	i.InputBuf.Delete(iStart, iEnd)
	return text
}

func (i *MessageInput) makeMessage(content string) discord.Message {
	return discord.Message{
		Type:      discord.DefaultMessage,
		ChannelID: i.Messages.ChannelID,
		GuildID:   i.Messages.GuildID,
		Author:    *App.Me,
		Content:   content,
		Timestamp: discord.Timestamp(time.Now()),
		Nonce:     randString(),
	}
}

func (i *MessageInput) paste(content string, pic *gdk.Pixbuf) error {
	path := filepath.Join(cache.Path, "clipboard.png")

	if err := pic.SavePNG(path, 9); err != nil {
		return errors.Wrap(err, "Failed to save PNG to "+path+":")
	}

	return i._upload(content, []string{path})
}

func (i *MessageInput) send(content string) error {
	if i.Editing != nil {
		edit := i.Editing
		i.Editing = nil
		must(i.Style.RemoveClass, "editing")

		if edit.Content == content {
			return nil
		}

		// If the content is empty, we delete the message instead:
		if content == "" {
			err := App.State.DeleteMessage(edit.ChannelID, edit.ID)
			return errors.Wrap(err, "Failed to delete message")
		}

		_, err := App.State.EditMessage(edit.ChannelID, edit.ID, content, nil, false)
		if err != nil {
			return errors.Wrap(err, "Failed to edit message")
		}

		return nil
	}

	// If the content is empty but we're not editing, don't send.
	if content == "" {
		return nil
	}

	// An invalid ID keeps the message invalid until it is sent.
	m := i.makeMessage(content)

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

func (i *MessageInput) upload(paths []string) {
	text := must(i.popContent).(string)
	if err := i._upload(text, paths); err != nil {
		log.Fatalln("Failed to upload:", err)
	}
}

func (i *MessageInput) _upload(content string, paths []string) error {
	u, err := NewMessageUploader(paths)
	if err != nil {
		return err
	}
	defer u.Close()

	m := i.makeMessage(content)
	s := u.MakeSendData(m)

	w, err := newMessageCustom(m)
	if err != nil {
		return errors.Wrap(err, "Failed to create a message container")
	}
	must(w.rightBottom.Add, u)

	if err := i.Messages.insert(w, m); err != nil {
		log.Errorln("Failed to add messages to be uploaded:", err)
	}

	_, err = App.State.SendMessageComplex(m.ChannelID, s)
	if err != nil {
		i.Messages.deleteNonce(m.Nonce)
		return errors.Wrap(err, "Failed to upload message")
	}
	must(w.rightBottom.Remove, u)

	return nil
}
