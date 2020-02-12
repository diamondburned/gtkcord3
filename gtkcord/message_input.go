package gtkcord

import (
	"github.com/gotk3/gotk3/gtk"
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

// func (m *Messages) loadMessageInput() (*MessageInput, error) {
// 	if m.Input == nil {
// 		m.Input = &MessageInput{
// 			Messages: m,
// 		}
// 	}

// 	if m.Input == nil {
// 		main, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
// 		if err != nil {
// 			return nil, errors.Wrap(err, "Failed to create main box")
// 		}

// 		ibox, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
// 		if err != nil {
// 			return nil, errors.Wrap(err, "Failed to create input box")
// 		}

// 		// TODO completer
// 		// comp, err := gtk.

// 		input, err := gtk.TextViewNew()
// 		if err != nil {
// 			return nil, errors.Wrap(err, "Failed to create input textview")
// 		}
// 	}
// }
