package typing

import (
	"html"
	"sort"
	"sync"
	"time"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/arikawa/state"
	"github.com/diamondburned/gtkcord3/gtkcord/components/animations"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/humanize"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/gotk3/gotk3/gtk"
)

const TypingTimeout = 8 * time.Second

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
		for tOld == nil {
			tOld = <-typingHandler
			// stop until we get something
		}

		// Block until a tick or a typing state
		select {
		case <-tick.C:
		case t := <-typingHandler:
			// Drain the ticker if it's not drained:
			if !tick.Stop() {
				<-tick.C
			}

			if t != nil {
				// Reset the timer to the shortest tick:
				if T := t.Shortest(); !T.IsZero() {
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
		tOld.render()

		// If there's nothing left to update, mark it.
		if tOld.Empty() {
			tOld = nil
		}
	}
}

type State struct {
	*gtk.Box
	Label *gtk.Label

	mu sync.Mutex

	Users []typingUser

	// updated by Render() only
	users []string

	state *state.State

	lastTyped time.Time
}

type typingUser struct {
	ID   discord.Snowflake
	Name string
	Time time.Time
}

func NewState(s *state.State) *State {
	initHandler()

	t := &State{
		state: s,
	}

	semaphore.IdleMust(func() {
		// the breathing 3 dot thing
		breathing, _ := animations.NewBreathing()

		t.Label, _ = gtk.LabelNew("")
		t.Label.SetMarginStart(4)
		t.Label.SetSingleLineMode(true)

		t.Box, _ = gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
		t.Box.SetHAlign(gtk.ALIGN_START)
		t.Box.SetVAlign(gtk.ALIGN_END)
		t.Box.Add(breathing)
		t.Box.Add(t.Label)
		t.Box.SetOpacity(0)

		gtkutils.Margin2(t.Box, 2, 20)
		t.Box.SetMarginTop(0)
	})

	return t
}

// Type is async. Kind of.
func (t *State) Type(chID discord.Snowflake) {
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

func (t *State) Empty() bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	return len(t.Users) == 0
}

func (t *State) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.Users = t.Users[:0]

	semaphore.IdleMust(func() {
		t.Label.SetText("")
		t.Box.SetOpacity(0)
	})
}

func (t *State) Stop() {
	t.Reset()
}

func (t *State) render() {
	t.mu.Lock()

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

	t.mu.Unlock()

	log.Println("Rendered", text)

	semaphore.IdleMust(func() {
		t.Label.SetMarkup(text)
		// Show or hide the breathing animation as well:
		if text == "" {
			t.Box.SetOpacity(0)
		} else {
			t.Box.SetOpacity(1)
		}
	})
}

func (t *State) Update() {
	select {
	case typingHandler <- t:
	default:
		log.Println("Not updating typing indicator.")
	}
}

func (t *State) Shortest() time.Time {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.Users) == 0 {
		return time.Time{}
	}

	t.sort()
	return t.Users[0].Time.Add(-1)
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
	if typing.UserID == t.state.Ready.User.ID {
		return
	}

	defer t.Update()

	t.mu.Lock()
	defer t.mu.Unlock()

	// Check duplicates:
	for _, u := range t.Users {
		if u.ID == typing.UserID {
			u.Time = typing.Timestamp.Time()
			t.sort()
			return
		}
	}

	// Temporarily unlock the mutex:
	t.mu.Unlock()

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
	if user.Name == "" && typing.GuildID.Valid() {
		n, err := t.state.MemberDisplayName(typing.GuildID, typing.UserID)
		if err == nil {
			user.Name = n
		}
	}

	// Attempt 3: if we have to manually fetch the user from their ID
	if user.Name == "" {
		u, err := t.state.User(typing.UserID)
		if err == nil {
			user.Name = u.Username
		}
	}

	// Attempt 4: just use the ID
	if user.Name == "" {
		user.Name = typing.UserID.String()
	}

	// Escape and format the name:
	user.Name = `<span weight="bold">` + html.EscapeString(user.Name) + `</span>`

	// Lock back the mutex:
	t.mu.Lock()

	t.Users = append(t.Users, user)
}

func (t *State) Remove(id discord.Snowflake) {
	defer t.Update()

	t.mu.Lock()
	defer t.mu.Unlock()

	for i := range t.Users {
		if t.Users[i].ID == id {
			t.Users = append(t.Users[:i], t.Users[i+1:]...)
			return
		}
	}
}
