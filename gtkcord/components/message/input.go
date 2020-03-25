package message

import (
	"math/rand"
	"path/filepath"
	"strings"
	"time"

	"github.com/diamondburned/arikawa/api"
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/components/message/completer"
	"github.com/diamondburned/gtkcord3/gtkcord/components/window"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/diamondburned/handy"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const InputIconSize = gtk.ICON_SIZE_LARGE_TOOLBAR

type Input struct {
	*handy.Column
	Messages *Messages

	Main  *gtk.Box
	Style *gtk.StyleContext

	InputBox  *gtk.Box
	Completer *completer.State

	Input    *gtk.TextView
	InputBuf *gtk.TextBuffer
	Upload   *gtk.Button
	Send     *gtk.Button

	Editing *discord.Message

	OnTyping func()

	// uploadButton
	// emojiButton
}

func NewInput(m *Messages) (i *Input) {
	c := handy.ColumnNew()
	c.Show()
	c.SetMaximumWidth(MaxMessageWidth)
	style, _ := c.GetStyleContext()

	main, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	main.SetSizeRequest(MaxMessageWidth, -1) // fill

	ibox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)

	upload, _ := gtk.ButtonNewFromIconName("document-open-symbolic", InputIconSize)
	send, _ := gtk.ButtonNewFromIconName("mail-send", InputIconSize)

	input, _ := gtk.TextViewNew()
	ibuf, _ := input.GetBuffer()

	i = &Input{
		Column:   c,
		Messages: m,
		Main:     main,
		Style:    style,
		InputBox: ibox,
		Upload:   upload,
		Send:     send,
		Input:    input,
		InputBuf: ibuf,
	}

	style.AddClass("message-input")

	// Initialize the completer:
	i.initCompleter()
	// Prepend the completer box:
	main.Add(i.Completer)

	// Prepare the message input box:
	gtkutils.Margin2(ibox, 4, 10)
	ibox.SetMarginBottom(0) // doing it legit by using label as padding

	upload.SetVAlign(gtk.ALIGN_START)
	upload.SetRelief(gtk.RELIEF_NONE)
	upload.Connect("clicked", func() {
		go SpawnUploader(i.upload)
	})

	send.SetVAlign(gtk.ALIGN_START)
	send.SetRelief(gtk.RELIEF_NONE)
	send.Connect("clicked", func() {
		text := i.popContent()

		go func() {
			if err := i.send(text); err != nil {
				log.Errorln("Failed to send message:", err)
			}
		}()
	})

	gtkutils.Margin2(input, 4, 10)
	input.AddEvents(int(gdk.KEY_PRESS_MASK))
	input.Connect("key-press-event", i.keyDown)
	input.SetWrapMode(gtk.WRAP_WORD_CHAR)
	input.SetVAlign(gtk.ALIGN_CENTER)

	// Add the message input box:
	main.Add(ibox)

	// Add the main box:
	c.Add(main)

	// Separators between the message input box
	s1, _ := gtk.SeparatorNew(gtk.ORIENTATION_VERTICAL)
	s2, _ := gtk.SeparatorNew(gtk.ORIENTATION_VERTICAL)

	ibox.PackStart(upload, false, false, 0)
	ibox.PackStart(s1, false, false, 2)
	ibox.PackStart(input, true, true, 0)
	ibox.PackStart(s2, false, false, 2)
	ibox.PackStart(send, false, false, 0)

	i.Main.ShowAll()

	return
}

func (i *Input) initCompleter() {
	if i.Completer == nil {
		i.Completer = completer.New(i.Messages.c, i.InputBuf, i.Messages)
	}
}

func (i *Input) keyDown(_ *gtk.TextView, ev *gdk.Event) bool {
	evKey := gdk.EventKeyNewFromEvent(ev)

	if evKey.Type() != gdk.EVENT_KEY_PRESS {
		return false
	}

	// Send an OnTyping request:
	i.Messages.Typing.Type(i.Messages.ChannelID)

	const shiftMask = uint(gdk.GDK_SHIFT_MASK)
	const cntrlMask = uint(gdk.GDK_CONTROL_MASK)

	var (
		state = evKey.State()
		key   = evKey.KeyVal()
	)

	if i.Completer.KeyDown(state, key) {
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
		var latest = i.Messages.LastFromMe()

		// If we can find the message:
		if latest != nil {
			// Trigger the edit message:
			go func() {
				if err := i.editMessage(latest.ID); err != nil {
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

	// Get text
	text := i.getContent()

	// Check if the numbers of ``` are odd.
	if !shift && strings.Count(i.getContent(), "```")%2 > 0 {
		// If yes, assume shift is held. We want the Enter key to insert new
		// lines.
		shift = true
	}

	// If Shift is being held:
	if shift {
		// Check if the start of line is a blockquote.
		var blockquote = false
		var lines = strings.Split(text, "\n")
		if len(lines) > 0 && strings.HasPrefix(lines[len(lines)-1], ">") {
			blockquote = true
		}

		// Insert a new line
		i.InputBuf.InsertAtCursor("\n")

		// If we're writing a blockquote:
		if blockquote {
			i.InputBuf.InsertAtCursor("> ")
		}

		return true
	}

	i.deleteContent()

	// Shift is not being held, send the message:
	go func() {
		if err := i.send(text); err != nil {
			log.Errorln("Failed to send message:", err)
		}
	}()

	return true
}

func (i *Input) editMessage(id discord.Snowflake) error {
	m, err := i.Messages.c.State.Store.Message(i.Messages.ChannelID, id)
	if err != nil {
		return errors.Wrap(err, "Failed to get message")
	}

	i.Editing = m
	semaphore.IdleMust(i.Style.AddClass, "editing")

	semaphore.IdleMust(i.InputBuf.SetText, i.Editing.Content)
	return nil
}

func (i *Input) getContent() string {
	var iStart, iEnd = i.InputBuf.GetBounds()

	text, err := i.InputBuf.GetText(iStart, iEnd, true)
	if err != nil {
		log.Errorln("Failed to get chatbox text buffer:", err)
		return ""
	}

	return text
}

func (i *Input) deleteContent() {
	i.InputBuf.Delete(i.InputBuf.GetBounds())
}

// popContent gets the current messages and deletes the buffer.
func (i *Input) popContent() string {
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

func (i *Input) makeMessage(content string) *discord.Message {
	return &discord.Message{
		Type:      discord.DefaultMessage,
		ChannelID: i.Messages.ChannelID,
		GuildID:   i.Messages.GuildID,
		Author:    i.Messages.c.Ready.User,
		Content:   content,
		Timestamp: discord.Timestamp(time.Now()),
		Nonce:     randString(),
	}
}

func (i *Input) paste(content string, pic *gdk.Pixbuf) error {
	path := filepath.Join(cache.Path, "clipboard.png")

	if err := pic.SavePNG(path, 9); err != nil {
		return errors.Wrap(err, "Failed to save PNG to "+path+":")
	}

	return i._upload(content, []string{path})
}

func (i *Input) send(content string) error {
	if i.Editing != nil {
		edit := i.Editing
		i.Editing = nil
		semaphore.IdleMust(i.Style.RemoveClass, "editing")

		if edit.Content == content {
			return nil
		}

		// If the content is empty, we delete the message instead:
		if content == "" {
			err := i.Messages.c.State.DeleteMessage(edit.ChannelID, edit.ID)
			return errors.Wrap(err, "Failed to delete message")
		}

		_, err := i.Messages.c.State.EditMessage(edit.ChannelID, edit.ID, content, nil, false)
		return errors.Wrap(err, "Failed to edit message")
	}

	// If the content is empty but we're not editing, don't send.
	if content == "" {
		return nil
	}

	// An invalid ID keeps the message invalid until it is sent.
	m := i.makeMessage(content)
	i.Messages.Insert(m)

	_, err := i.Messages.c.State.SendMessageComplex(m.ChannelID, api.SendMessageData{
		Content: m.Content,
		Nonce:   m.Nonce,
	})
	if err != nil {
		i.Messages.deleteNonce(m.Nonce)
		return errors.Wrap(err, "Failed to send message")
	}

	return nil
}

func (i *Input) upload(paths []string) {
	text := semaphore.IdleMust(i.popContent).(string)
	if err := i._upload(text, paths); err != nil {
		log.Fatalln("Failed to upload:", err)
	}
}

func (i *Input) _upload(content string, paths []string) error {
	u, err := NewMessageUploader(paths)
	if err != nil {
		return err
	}
	defer u.Close()

	m := i.makeMessage(content)
	s := u.MakeSendData(m)

	w := newMessageCustom(m)
	semaphore.IdleMust(w.rightBottom.Add, u)

	i.Messages._insert(w)
	go w.updateAuthor(i.Messages.c, m.GuildID, m.Author)

	_, err = i.Messages.c.State.SendMessageComplex(m.ChannelID, s)
	if err != nil {
		i.Messages.deleteNonce(m.Nonce)
		return errors.Wrap(err, "Failed to upload message")
	}
	semaphore.IdleMust(w.rightBottom.Remove, u)

	return nil
}

func randString() string {
	const randLen = 20
	const alphabet = "abcdefghijklmnopqrstuvwxyz" +
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	b := make([]byte, randLen)
	for i := range b {
		b[i] = alphabet[rand.Intn(len(alphabet))]
	}

	return string(b)
}
