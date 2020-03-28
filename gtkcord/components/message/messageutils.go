package message

import (
	"time"

	"github.com/diamondburned/arikawa/discord"
)

func injectMessage(m *Messages, w *Message) {
	w.OnUserClick = m.onAvatarClick
	w.OnRightClick = m.onRightClick
	w.ListBoxRow.SetFocusVAdjustment(m.Messages.GetFocusVAdjustment())
}

func shouldCondense(msg, last *Message) bool {
	return true &&
		msg.AuthorID == last.AuthorID &&
		msg.Timestamp.Sub(last.Timestamp) < 5*time.Minute
}

func lastMessageFrom(msgs []*Message, author discord.Snowflake) *Message {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msg := msgs[i]; msg.AuthorID == author && !msg.Condensed {
			return msg
		}
	}
	return nil
}

func tryCondense(msgs []*Message, msg *Message) {
	if len(msgs) == 0 {
		return
	}

	var last = lastMessageFrom(msgs, msg.AuthorID)
	if last == nil {
		return
	}

	if shouldCondense(msg, last) {
		msg.setOffset(last)
		msg.SetCondensedUnsafe(true)
	}
}
