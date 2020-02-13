package gtkcord

import (
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

type MessageInput struct {
	ExtendedWidget
	Messages *Messages

	Main *gtk.Box

	InputBox   *gtk.Box
	Completion *Completer

	Input  *gtk.TextView
	Upload *gtk.Button
	Emoji  *gtk.Button

	// counter for nonce
	counter uint32

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

func (messages *Messages) loadMessageInput() (*MessageInput, error) {
	if messages.Input == nil {
		messages.Input = &MessageInput{
			Messages: messages,
		}
	}

	m := messages.Input

	if m.Input == nil {
		main, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to create main box")
		}
		m.Main = main

		ibox, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to create input box")
		}
		m.InputBox = ibox

		// TODO completer
		// comp, err := gtk.

		input, err := gtk.TextViewNew()
		if err != nil {
			return nil, errors.Wrap(err, "Failed to create input textview")
		}
		m.Input = input

		upload, err := gtk.ButtonNewFromIconName(
			"document-open-symbolic", gtk.ICON_SIZE_LARGE_TOOLBAR)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to create upload button")
		}
		m.Upload = upload

		emoji, err := gtk.ButtonNewFromIconName(
			"face-smile-symbolic", gtk.ICON_SIZE_LARGE_TOOLBAR)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to create emoji button")
		}
		m.Emoji = emoji
	}

	return m, nil
}

// func (messages *Messages) send(content string) error {
// 	var nonce = fmt.Sprintf("%d:%d", messages.Channel.ID, messages.counter)
// 	messages.counter++

// 	App.State.SendMessageComplex(messages.Channel.ID, api.SendMessageData{
// 		Nonce: nonce,
// 	})
// }
