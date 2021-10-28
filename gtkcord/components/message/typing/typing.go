package typing

import (
	"html"
	"sort"
	"strings"
	"time"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gotk4/pkg/pango"
	"github.com/diamondburned/gtkcord3/gtkcord/components/animations"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/internal/humanize"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/diamondburned/ningen/v2"
)

const TypingTimeout = 10 * time.Second

type State struct {
	*gtk.Box
	Label *gtk.Label
	Users []TypingUser

	// updated by Render() only
	users []string
	state *ningen.State

	lastTyped  time.Time
	loopHandle glib.SourceHandle
}

type TypingUser struct {
	ID   discord.UserID
	Name string
	Time time.Time
}

func NewState(s *ningen.State) *State {
	t := &State{
		state: s,
	}

	// the breathing 3 dot thing
	breathing := animations.NewBreathing()

	t.Label = gtk.NewLabel("")
	t.Label.SetMarginStart(4)
	t.Label.SetSingleLineMode(true)
	t.Label.SetEllipsize(pango.EllipsizeEnd)
	t.Label.SetAttributes(gtkutils.PangoAttrs(
		pango.NewAttrScale(0.8),
	))

	t.Box = gtk.NewBox(gtk.OrientationHorizontal, 0)
	t.Box.SetHAlign(gtk.AlignStart)
	t.Box.SetVAlign(gtk.AlignEnd)
	t.Box.SetHExpand(true)
	t.Box.Add(breathing)
	t.Box.Add(t.Label)
	t.Box.SetOpacity(0)

	gtkutils.Margin2(t.Box, 0, 10)

	return t
}

func (t *State) bindHandler() {
	if t.loopHandle > 0 {
		return
	}

	t.loopHandle = glib.TimeoutSecondsAdd(5, func() bool {
		t.render()

		if t.IsEmpty() {
			t.loopHandle = 0
			return false
		}

		return true
	})
}

func (t *State) unbindHandler() {
	if t.loopHandle > 0 {
		glib.SourceRemove(t.loopHandle)
		t.loopHandle = 0
	}
}

func (t *State) Type(chID discord.ChannelID) {
	now := time.Now()
	// if we sent a typing request the past $TypingTimeout:
	if now.Add(-TypingTimeout).Before(t.lastTyped) {
		return
	}
	t.lastTyped = now

	/* TODO: INSPECT ME */
	go func() {
		if err := t.state.Typing(chID); err != nil {
			log.Errorln("Failed to send typing to ch", chID, ":", err)
		}
	}()
}

func (t *State) IsEmpty() bool {
	return len(t.Users) == 0
}

func (t *State) Reset() {
	t.lastTyped = time.Time{} // zero out
	t.Users = t.Users[:0]

	t.Label.SetText("")
	t.Box.SetOpacity(0)

	t.unbindHandler()
}

func (t *State) Stop() {
	t.Reset()
}

func (t *State) render() {
	t.cleanUp()

	var text strings.Builder

	switch len(t.Users) {
	case 0:
		// empty text
	case 1, 2, 3:
		// clear user string
		t.users = t.users[:0]

		// join
		for i := range t.Users {
			t.users = append(t.users, t.Users[i].Name)
		}
		text.WriteString(humanize.Strings(t.users))

		if len(t.Users) == 1 {
			text.WriteString(" is typing...")
		} else {
			text.WriteString(" are typing...")
		}

	default:
		text.WriteString("Several people are typing...")
	}

	t.Label.SetMarkup(text.String())

	// Show or hide the breathing animation as well:
	if text.Len() == 0 {
		t.Box.SetOpacity(0)
	} else {
		t.Box.SetOpacity(1)
	}
}

func (t *State) Shortest() time.Time {
	if len(t.Users) == 0 {
		return time.Time{}
	}

	t.sort()
	return t.Users[0].Time.Add(-1 * time.Millisecond) // extra overhead just in case
}

func (t *State) cleanUp() {
	// now - timeout
	now := time.Now().Add(-TypingTimeout)

	for i := 0; i < len(t.Users); i++ {
		if t.Users[i].Time.Before(now) {
			t.Users = append(t.Users[:i], t.Users[i+1:]...)
		}
	}
}

func (t *State) sort() {
	sort.Slice(t.Users, func(i, j int) bool {
		// earliest first:
		return t.Users[i].Time.Before(t.Users[j].Time)
	})
}

func (t *State) Add(typing *gateway.TypingStartEvent) {
	me, _ := t.state.Me()
	if typing.UserID == me.ID {
		return
	}

	defer t.render()

	// Check duplicates:
	for _, u := range t.Users {
		if u.ID == typing.UserID {
			u.Time = typing.Timestamp.Time()
			t.sort()
			return
		}
	}

	user := TypingUser{
		ID:   typing.UserID,
		Name: "",
		Time: typing.Timestamp.Time(),
	}

	// Attempt 1: if the event gives us the Member struct:
	if typing.Member != nil {
		user.Name = typing.Member.User.Username
		if typing.Member.Nick != "" {
			user.Name = typing.Member.Nick
		}
	}

	// Attempt 2: if the event has a GuildID
	if user.Name == "" && typing.GuildID.IsValid() {
		n, err := t.state.MemberDisplayName(typing.GuildID, typing.UserID)
		if err == nil {
			user.Name = n
		}
	}

	// Attempt 3: Check the DM channel:
	if c, err := t.state.Channel(typing.ChannelID); err == nil {
		for _, r := range c.DMRecipients {
			if r.ID == typing.UserID {
				user.Name = r.Username
				break
			}
		}
	}

	// Attempt 4: just use the ID
	if user.Name == "" {
		user.Name = typing.UserID.String()
	}

	// Escape and format the name:
	user.Name = `<span weight="bold">` + html.EscapeString(user.Name) + `</span>`

	t.Users = append(t.Users, user)
	t.bindHandler()
}

func (t *State) Remove(id discord.UserID) {
	defer t.render()

	for i := range t.Users {
		if t.Users[i].ID == id {
			t.Users = append(t.Users[:i], t.Users[i+1:]...)
			break
		}
	}

	if t.IsEmpty() {
		t.unbindHandler()
	}
}
