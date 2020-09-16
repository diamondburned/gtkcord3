package typing

import (
	"html"
	"sort"
	"time"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/arikawa/state"
	"github.com/diamondburned/gtkcord3/gtkcord/components/animations"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/internal/humanize"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/diamondburned/gtkcord3/internal/moreatomic"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

const TypingTimeout = 10 * time.Second

var typingHandler chan *State

func initHandler() {
	if typingHandler == nil {
		typingHandler = make(chan *State, 1)
		go handler()
	}
}

func handler() {
	var tOld *State
	var tick = time.NewTimer(TypingTimeout)

	for {
		// First, catch a TypingState
		// for tOld == nil {
		// 	tOld = <-typingHandler
		// 	// stop until we get something
		// }

		// Block until a tick or a typing state
		select {
		case <-tick.C:
		case t := <-typingHandler:
			// Drain the ticker if it's not drained:
			if !tick.Stop() {
				select {
				case <-tick.C:
				default:
				}
			}

			if t != nil {
				// Reset the timer to the shortest tick:
				if T := t.shortest.Get(); !T.IsZero() {
					tick.Reset(time.Now().Sub(T))
				}
			}

			// Then set the new tOld.
			tOld = t
		}

		// if tOld is nil, skip this turn and let the above for loop do its
		// work.
		if tOld == nil {
			continue
		}

		// Render the typing state:
		if empty := tOld.Render(); empty {
			// If there's nothing left to update, mark it.
			tOld = nil
		}
	}
}

type State struct {
	*gtk.Box
	Label *gtk.Label

	Users    []typingUser
	users    []string
	shortest moreatomic.Time

	state *state.State

	lastTyped time.Time
}

type typingUser struct {
	ID   discord.UserID
	Name string
	Time time.Time
}

func NewState(s *state.State) *State {
	initHandler()

	t := &State{
		state: s,
	}

	// the breathing 3 dot thing
	breathing, _ := animations.NewBreathing()

	t.Label, _ = gtk.LabelNew("")
	t.Label.SetMarginStart(4)
	t.Label.SetSingleLineMode(true)
	t.Label.SetEllipsize(pango.ELLIPSIZE_END)

	t.Box, _ = gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	t.Box.SetHAlign(gtk.ALIGN_START)
	t.Box.SetVAlign(gtk.ALIGN_END)
	t.Box.SetHExpand(true)
	t.Box.Add(breathing)
	t.Box.Add(t.Label)
	t.Box.SetOpacity(0)

	gtkutils.Margin2(t.Box, 0, 10)

	return t
}

// Type is async. Kind of.
func (t *State) Type(chID discord.ChannelID) {
	now := time.Now()
	// if we sent a typing request the past $TypingTimeout:
	if now.Add(-TypingTimeout).Before(t.lastTyped) {
		return
	}
	t.lastTyped = now

	go func() {
		if err := t.state.Typing(chID); err != nil {
			log.Errorln("Failed to send typing to ch", chID, ":", err)
		}
	}()
}

func (t *State) isEmpty() bool {
	return len(t.Users) == 0
}

func (t *State) Reset() {
	t.lastTyped = time.Time{} // zero out
	t.Users = nil

	t.Label.SetText("")
	t.Box.SetOpacity(0)
}

func (t *State) Stop() {
	t.Reset()
}

func (t *State) Render() (empty bool) {
	semaphore.IdleMust(func() {
		empty = t.render()
	})
	return
}

func (t *State) render() bool {
	t.cleanUp()

	var text = ""

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
		text = humanize.Strings(t.users)

		if len(t.Users) == 1 {
			text += " is typing..."
		} else {
			text += " are typing..."
		}

	default:
		text = "Several people are typing..."
	}

	t.Label.SetMarkup(`<span size="smaller">` + text + "</span>")
	// Show or hide the breathing animation as well:
	if text == "" {
		t.Box.SetOpacity(0)
	} else {
		t.Box.SetOpacity(1)
	}

	return t.isEmpty()
}

func (t *State) Update() {
	t.sort()
	if len(t.Users) > 0 {
		t.shortest.Set(t.Users[0].Time)
	} else {
		t.shortest.Reset()
	}

	select {
	case typingHandler <- t:
	default:
	}
}

func (t *State) cleanUp() {
	// now - timeout
	now := time.Now().Add(-TypingTimeout)

	for i := 0; i < len(t.Users); i++ {
		if t.Users[i].Time.Before(now) {
			t.Users = append(t.Users[:i], t.Users[i+1:]...)
		}
	}

	if len(t.Users) == 0 {
		// GC the old slice.
		t.Users = nil
	}
}

func (t *State) sort() {
	sort.Slice(t.Users, func(i, j int) bool {
		// earliest first:
		return t.Users[i].Time.Before(t.Users[j].Time)
	})
}

func (t *State) Add(typing *gateway.TypingStartEvent) {
	if typing.UserID == t.state.Ready.User.ID {
		return
	}

	defer t.Update()

	// Check duplicates:
	for _, u := range t.Users {
		if u.ID == typing.UserID {
			u.Time = typing.Timestamp.Time()
			return
		}
	}

	var user = typingUser{
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
		m, err := t.state.Store.Member(typing.GuildID, typing.UserID)
		if err == nil {
			if m.Nick != "" {
				user.Name = m.Nick
			} else {
				user.Name = m.User.Username
			}
		}
	}

	// Attempt 3: Check the DM channel:
	if c, err := t.state.Store.Channel(typing.ChannelID); err == nil {
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
}

func (t *State) Remove(id discord.UserID) {
	defer t.Update()

	for i := range t.Users {
		if t.Users[i].ID == id {
			t.Users = append(t.Users[:i], t.Users[i+1:]...)
			return
		}
	}
}
