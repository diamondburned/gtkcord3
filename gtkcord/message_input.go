package gtkcord

import "github.com/gotk3/gotk3/gtk"

type MessageInput struct {
	ExtendedWidget
	Channel *Channel

	Input      *gtk.TextView
	Completion *gtk.ListBox
}

type MessageCompleter struct {
}

// func (c *Channel) getMessageInput() (*MessageInput, error) {

// }
