package message

import (
	"math/rand"
	"path/filepath"
	"strings"
	"time"

	"github.com/diamondburned/arikawa/v2/api"
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/gotk4-handy/pkg/handy"
	"github.com/diamondburned/gotk4/pkg/gdk/v3"
	"github.com/diamondburned/gotk4/pkg/gdkpixbuf/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/components/emojis"
	"github.com/diamondburned/gtkcord3/gtkcord/components/message/completer"
	"github.com/diamondburned/gtkcord3/gtkcord/components/message/extras"
	"github.com/diamondburned/gtkcord3/gtkcord/components/message/typing"
	"github.com/diamondburned/gtkcord3/gtkcord/components/window"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/variables"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/diamondburned/gtkcord3/internal/zwsp"
	"github.com/diamondburned/ningen/v2"
	"github.com/pkg/errors"
)

type Input struct {
	*handy.Clamp
	Messages *Messages

	Main  *gtk.Box
	Style *gtk.StyleContext

	Completer *completer.State

	InputBox *gtk.Box
	Input    *gtk.TextView
	InputBuf *gtk.TextBuffer
	Upload   *gtk.Button
	Emoji    *gtk.Button
	Send     *gtk.Button

	Bottom *gtk.Box
	Typing *typing.State

	// | Typing...      | Editing. _Cancel_ |
	EditRevealer *gtk.Revealer
	EditLabel    *gtk.Label
	EditCancel   *gtk.Button

	Editing *discord.Message
}

func NewInput(m *Messages) (i *Input) {
	i = &Input{
		Messages: m,
	}

	// Make the inputs first:

	i.Input = gtk.NewTextView()
	i.Input.SetLeftMargin(10)
	i.Input.SetRightMargin(10)
	i.Input.SetBottomMargin(5)
	i.Input.SetTopMargin(5)
	i.Input.SetHExpand(true)
	i.Input.AddEvents(int(gdk.KeyPressMask))
	i.Input.Connect("key-press-event", i.keyDown)
	i.Input.SetWrapMode(gtk.WrapWordChar)
	i.Input.SetVAlign(gtk.AlignCenter)

	i.InputBuf = i.Input.Buffer()

	// wrap Input inside a ScrolledWindow
	isw := gtk.NewScrolledWindow(nil, nil)
	isw.SetPropagateNaturalHeight(true)
	isw.SetMaxContentHeight(144) // from Discord
	isw.SetMinContentHeight(24)  // arbitrary
	isw.SetPlacement(gtk.CornerBottomLeft)
	isw.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	isw.Add(i.Input)

	i.Input.SetFocusVAdjustment(isw.VAdjustment())
	i.Input.SetFocusHAdjustment(isw.HAdjustment())

	// Make the rest of the main widgets:

	i.Clamp = handy.NewClamp()
	i.Clamp.SetHExpand(true)
	i.Clamp.SetSizeRequest(300, -1) // min width
	i.Clamp.SetMaximumSize(i.Messages.MessageWidth)
	i.Clamp.Show()

	i.Style = i.Clamp.StyleContext()
	i.Style.AddClass("message-input")

	i.Main = gtk.NewBox(gtk.OrientationVertical, 0)
	i.Main.SetHExpand(true) // fill

	// Add the completer into the box:
	i.Completer = completer.New(m.c, i.InputBuf, m)

	i.InputBox = gtk.NewBox(gtk.OrientationHorizontal, 0)
	i.InputBox.SetHExpand(true)
	i.InputBox.SetMarginBottom(0) // doing it legit by using label as padding
	gtkutils.Margin2(i.InputBox, 4, 10)

	i.Upload = gtk.NewButtonFromIconName("document-open-symbolic", int(variables.InputIconSize))
	i.Upload.SetVAlign(gtk.AlignBaseline)
	i.Upload.SetRelief(gtk.ReliefNone)
	i.Upload.Connect("clicked", func() {
		extras.SpawnUploader(func(paths []string) {
			i.upload(i.popContent(), paths)
		})
	})

	// Emoji popup constructor:
	espawner := emojis.New(i.Messages.c, func(emoji string) {
		i.InputBuf.InsertAtCursor(emoji)
	})
	espawner.Refocus = i.Input // refocus on close

	i.Emoji = gtk.NewButtonFromIconName("face-smile-symbolic", int(variables.InputIconSize))
	i.Emoji.SetVAlign(gtk.AlignBaseline)
	i.Emoji.SetRelief(gtk.ReliefNone)
	i.Emoji.SetMarginStart(2)
	i.Emoji.SetMarginEnd(2)
	i.Emoji.Connect("clicked", func(b *gtk.Button) {
		opened := espawner.Spawn(b, i.Messages.GuildID())
		opened.Popup()
	})

	send := gtk.NewButtonFromIconName("mail-send", int(variables.InputIconSize))
	i.Send = send
	send.SetVAlign(gtk.AlignBaseline)
	send.SetRelief(gtk.ReliefNone)
	send.Connect("clicked", func() {
		i.send(i.popContent())
	})

	// Initialize the typing state:
	i.Typing = typing.NewState(m.c)

	// Make the edit indicator widgets:
	i.EditRevealer = gtk.NewRevealer()
	i.EditRevealer.SetRevealChild(false)
	i.EditRevealer.SetTransitionType(gtk.RevealerTransitionTypeCrossfade)
	i.EditRevealer.SetTransitionDuration(100)

	// Add the main box into the revealer:
	editBox := gtk.NewBox(gtk.OrientationHorizontal, 0)
	editBox.SetHAlign(gtk.AlignEnd)
	gtkutils.Margin2(editBox, 0, 10)

	i.EditLabel = gtk.NewLabel(`<span color="#3f7ce0" weight="bold">Editing</span>`)
	i.EditLabel.SetUseMarkup(true)
	gtkutils.Margin2(i.EditLabel, 0, 10)

	i.EditCancel = gtk.NewButtonWithLabel("Cancel")
	i.EditCancel.SetRelief(gtk.ReliefNone)
	i.EditCancel.Connect("clicked", i.stopEditing)

	i.Bottom = gtk.NewBox(gtk.OrientationHorizontal, 0)

	// Adding things:

	i.Clamp.Add(i.Main)

	// Add into the main box:
	i.Main.Add(i.Completer)
	i.Main.Add(i.InputBox)
	i.Main.Add(i.Bottom)

	// Separators between the message input box
	s1 := gtk.NewSeparator(gtk.OrientationVertical)
	s2 := gtk.NewSeparator(gtk.OrientationVertical)

	// Add into the input box:
	i.InputBox.PackStart(i.Upload, false, false, 0)
	i.InputBox.PackStart(s1, false, false, 2)
	i.InputBox.PackStart(isw, true, true, 0)
	i.InputBox.PackStart(i.Emoji, false, false, 0)
	i.InputBox.PackStart(s2, false, false, 2)
	i.InputBox.PackStart(send, false, false, 0)

	// Add the typing indicator and edit cancel boxes:
	i.Bottom.Add(i.Typing)
	i.Bottom.Add(i.EditRevealer)

	// Add the edit widgets:
	i.EditRevealer.Add(editBox)
	editBox.Add(i.EditLabel)
	editBox.Add(i.EditCancel)

	i.Main.ShowAll()
	return
}

func (i *Input) keyDown(ev *gdk.Event) bool {
	if ev.AsType() != gdk.KeyPressType {
		return false
	}

	const shiftMask = gdk.ShiftMask
	const cntrlMask = gdk.ControlMask

	var (
		evKey = ev.AsKey()
		state = evKey.State()
		key   = evKey.Keyval()
	)

	if i.Messages.InputOnTyping && gtkutils.KeyIsASCII(key) {
		// Send an OnTyping request. This does not acquire the mutex, but instead
		// gets the ID atomically.
		i.Typing.Type(i.Messages.ChannelID())
	}

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
		text := i.popContent()

		clipboard.RequestImage(func(_ *gtk.Clipboard, pixbuf *gdkpixbuf.Pixbuf) {
			if pixbuf == nil {
				log.Errorln("failed to get image from clipboard")
				return
			}

			i.paste(text, pixbuf)
		})

		return true
	}

	isEsc := key == gdk.KEY_Escape

	// If escape key is pressed and we're editing something:
	if isEsc && i.Editing != nil {
		i.stopEditing()
		return true
	}

	isUpArrow := key == gdk.KEY_Up

	// If arrow up is pressed and the input box is empty:
	if isUpArrow && i.getContent() == "" {
		// Try and look backwards for the latest message:
		latest := i.Messages.LastFromMe()

		// If we can find the message:
		if latest != nil {
			// Trigger the edit message:
			i.editMessage(latest.ID)
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
	i.send(text)
	return true
}

func (i *Input) stopEditing() {
	// Clear the text box:
	i.InputBuf.SetText("")

	// Reset state:
	i.Editing = nil
	i.Style.RemoveClass("editing")

	// Collapse the button:
	i.EditRevealer.SetRevealChild(false)
}

func (i *Input) editMessage(id discord.MessageID) {
	m, err := i.Messages.c.State.Cabinet.Message(i.Messages.ChannelID(), id)
	if err != nil {
		// TODO: blink red.
		log.Errorln("failed to get message for editing:", err)
		return
	}

	i.Editing = m

	// Add class
	i.Style.AddClass("editing")

	// Reveal the cancel buttons:
	i.EditRevealer.SetRevealChild(true)

	// Set the content:
	i.InputBuf.SetText(i.Editing.Content)
}

func (i *Input) getContent() string {
	start, end := i.InputBuf.Bounds()
	return i.InputBuf.Text(start, end, true)
}

func (i *Input) getContentUpToCursor() string {
	start := i.InputBuf.StartIter()
	cursor := i.InputBuf.IterAtMark(i.InputBuf.GetInsert())
	return i.InputBuf.Text(start, cursor, true)
}

func (i *Input) deleteContent() {
	i.InputBuf.Delete(i.InputBuf.Bounds())
}

// popContent gets the current messages and deletes the buffer.
func (i *Input) popContent() string {
	start, end := i.InputBuf.Bounds()

	text := i.InputBuf.Text(start, end, true)
	if text == "" {
		return ""
	}

	i.InputBuf.Delete(start, end)
	return text
}

func (i *Input) makeMessage(content string) *discord.Message {
	if i.Messages.InputZeroWidth {
		content = zwsp.Insert(content)
	}

	me, _ := i.Messages.c.Me()

	return &discord.Message{
		Type:      discord.DefaultMessage,
		ChannelID: i.Messages.ChannelID(),
		GuildID:   i.Messages.GuildID(),
		Author:    *me,
		Content:   content,
		Timestamp: discord.Timestamp(time.Now()),
		Nonce:     randString(),
	}
}

func (i *Input) paste(content string, pic *gdkpixbuf.Pixbuf) {
	go func() {
		path := filepath.Join(cache.TmpPath(), "clipboard.png")

		if err := pic.Savev(path, "png", nil, nil); err != nil {
			log.Errorln("failed to save clipboard PNG:", err)
			return
		}

		glib.IdleAdd(func() {
			i.upload(content, []string{path})
		})
	}()
}

func (i *Input) send(content string) {
	if i.Editing != nil {
		edit := i.Editing
		i.stopEditing()

		if edit.Content == content {
			return
		}

		go func() {
			var err error
			if content == "" {
				err = i.Messages.c.State.DeleteMessage(edit.ChannelID, edit.ID)
			} else {
				_, err = i.Messages.c.State.EditText(edit.ChannelID, edit.ID, content)
			}

			if err == nil {
				return
			}

			glib.IdleAdd(func() {
				if msg := i.Messages.Find(edit.ID); msg != nil {
					msg.ShowError(errors.Wrap(err, "failed to edit message"))
				}
			})
		}()
	}

	// If the content is empty but we're not editing, don't send.
	if content == "" {
		return
	}

	// An invalid ID keeps the message invalid until it is sent.
	m := i.makeMessage(content)
	w := i.Messages.Upsert(m)

	go func() {
		_, err := i.Messages.c.State.SendMessageComplex(m.ChannelID, api.SendMessageData{
			Content: m.Content,
			Nonce:   m.Nonce,
		})
		if err == nil {
			return
		}
		log.Errorln("failed to send message:", err)
		glib.IdleAdd(func() {
			w.ShowError(errors.Wrap(err, "failed to send message"))
		})
	}()
}

func (i *Input) upload(content string, paths []string) {
	m := i.makeMessage(content)

	w := NewMessageCustom(m)
	w.UpdateAuthor(i.Messages.c, m.GuildID, m.Author)
	i.Messages.Insert(w)

	go func() {
		if err := upload(i.Messages.c, w, m, paths); err != nil {
			log.Errorln("failed to upload:", err)
			glib.IdleAdd(func() {
				w.ShowError(errors.Wrap(err, "failed to upload"))
			})
		}
	}()
}

func upload(n *ningen.State, w *Message, m *discord.Message, paths []string) error {
	u, err := extras.NewMessageUploader(paths)
	if err != nil {
		return err
	}
	defer u.Close()

	s := u.MakeSendData(m)

	glib.IdleAdd(func() { w.rightBottom.Add(u) })

	_, err = n.SendMessageComplex(m.ChannelID, s)
	if err != nil {
		return err
	}

	glib.IdleAdd(func() { w.rightBottom.Remove(u) })
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
