package message

import (
	"time"

	"github.com/diamondburned/arikawa/discord"
)

func injectMessage(m *Messages, w *Message) {
	w.OnUserClick = m.onAvatarClick
}

func shouldCondense(msgs []*Message, msg *Message) bool {
	if len(msgs) == 0 {
		return false
	}

	var last = msgs[len(msgs)-1]

	return last.AuthorID == msg.AuthorID &&
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
	if shouldCondense(msgs, msg) {
		msg.setOffset(lastMessageFrom(msgs, msg.AuthorID))
		msg.SetCondensedUnsafe(true)
	}
}
