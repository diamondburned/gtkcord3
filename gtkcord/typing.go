package gtkcord

import (
	"sort"
	"sync"
	"time"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/gtkcord3/gtkcord/animations"
	"github.com/diamondburned/gtkcord3/humanize"
	"github.com/gotk3/gotk3/gtk"
)

const TypingTimeout = 10 * time.Second

type TypingState struct {
	*gtk.Box
	Label *gtk.Label

	mu sync.Mutex

	Users []typingUser

	// updated by Render() only
	users []string
}

type typingUser struct {
	ID   discord.Snowflake
	Name string
	Time time.Time
}

func (m *Messages) loadTypingState() {
	if m.Typing == nil {
		m.Typing = &TypingState{}
	}

	t := m.Typing

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.Box != nil {
		return
	}

	must(func() {
		// the breathing 3 dot thing
		breathing, _ := animations.NewBreathing()

		t.Label, _ = gtk.LabelNew("")
		t.Label.SetSizeRequest(-1, 22) // 22px is a magic number
		t.Label.SetMarginStart(4)

		t.Box, _ = gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
		t.Box.SetHAlign(gtk.ALIGN_START)
		t.Box.SetVAlign(gtk.ALIGN_END)
		t.Box.Add(breathing)
		t.Box.Add(t.Label)
		t.Box.SetOpacity(0)

		margin2(t.Box, 2, AvatarPadding*2)
		t.Box.SetMarginTop(0)
	})

	return
}

func (t *TypingState) Empty() bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	return len(t.Users) == 0
}

func (t *TypingState) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.Users = t.Users[:0]

	must(func() {
		t.Label.SetText("")
		t.Box.SetOpacity(0)
	})
}

func (t *TypingState) Stop() {
	t.Reset()
}

func (t *TypingState) render() {
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
		for _, u := range t.Users {
			t.users = append(t.users, u.Name)
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

	must(func() {
		t.Label.SetMarkup(text)
		// Show or hide the breathing animation as well:
		if text == "" {
			t.Box.SetOpacity(0)
		} else {
			t.Box.SetOpacity(1)
		}
	})
}

func (t *TypingState) Update() {
	App.typingHandler <- t
}

func (t *TypingState) cleanUp() {
	// now - timeout
	now := time.Now().Add(-TypingTimeout)

	for i := 0; i < len(t.Users); i++ {
		if t.Users[i].Time.Before(now) {
			t.Users = append(t.Users[:i], t.Users[i+1:]...)
		}
	}
}

func (t *TypingState) sort() {
	sort.Slice(t.Users, func(i, j int) bool {
		// earliest first:
		return t.Users[i].Time.Before(t.Users[j].Time)
	})
}

func (t *TypingState) Add(typing *gateway.TypingStartEvent) {
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
		n, err := App.State.MemberDisplayName(typing.GuildID, typing.UserID)
		if err == nil {
			user.Name = n
		}
	}

	// Attempt 3: if we have to manually fetch the user from their ID
	if user.Name == "" {
		u, err := App.State.User(typing.UserID)
		if err == nil {
			user.Name = u.Username
		}
	}

	// Attempt 4: just use the ID
	if user.Name == "" {
		user.Name = typing.UserID.String()
	}

	// Escape and format the name:
	user.Name = bold(escape(user.Name))

	// Lock back the mutex:
	t.mu.Lock()

	t.Users = append(t.Users, user)
}

func (t *TypingState) Remove(id discord.Snowflake) {
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
