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

func shouldCondense(msgs []*Message, msg, lastSameAuthor *Message) bool {
	if len(msgs) == 0 {
		return false
	}

	if lastSameAuthor == nil {
		return false
	}

	var latest = msgs[len(msgs)-1]

	if msg.AuthorID != latest.AuthorID {
		return false
	}

	return msg.Timestamp.Sub(lastSameAuthor.Timestamp) < 5*time.Minute
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

	if shouldCondense(msgs, msg, last) {
		msg.setOffset(last)
		msg.SetCondensedUnsafe(true)
	}
}
