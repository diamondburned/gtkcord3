package members

import (
	"fmt"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/components/user"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/gotk3/gotk3/gtk"
)

const MemberAvatarSize = 32

type Section struct {
	*gtk.ListBoxRow
	Name  string
	Count uint64

	Label *gtk.Label
}

func NewSection(name string, count uint64) *Section {
	r, _ := gtk.ListBoxRowNew()
	r.Show()
	r.SetSelectable(false)
	r.SetActivatable(false)
	gtkutils.InjectCSSUnsafe(r, "section", "")

	l, _ := gtk.LabelNew("")
	l.Show()
	l.SetHAlign(gtk.ALIGN_START)
	gtkutils.Margin4(l, 16, 2, 8, 8)
	r.Add(l)

	s := &Section{
		ListBoxRow: r,
		Label:      l,
	}
	s.UpdateUnsafe(name, count)

	return s
}

func (s *Section) shouldUpdate(name string, count uint64) bool {
	if (name != "" && s.Name == name) && (count > 0 && s.Count == count) {
		return false
	}

	if name != "" {
		s.Name = name
	}
	if count > 0 {
		s.Count = count
	}
	return true
}

func (s *Section) Update(name string, count uint64) {
	if !s.shouldUpdate(name, count) {
		return
	}

	semaphore.IdleMust(s.Label.SetMarkup,
		fmt.Sprintf(`<span size="smaller" weight="bold">%s—%d</span>`, s.Name, s.Count))
}

func (s *Section) UpdateUnsafe(name string, count uint64) {
	if !s.shouldUpdate(name, count) {
		return
	}

	s.Label.SetMarkup(
		fmt.Sprintf(`<span size="smaller" weight="bold">%s—%d</span>`, s.Name, s.Count))
}

type Member struct {
	*gtk.ListBoxRow
	User *user.Container

	// user
	ID discord.Snowflake
}

func NewMember(m discord.Member, p discord.Presence, guild discord.Guild) *Member {
	body := user.New()
	body.UpdateMember(m, guild)
	body.UpdateActivity(p.Game)
	body.UpdateStatus(p.Status)

	if m.User.Avatar != "" {
		body.UpdateAvatar(m.User.AvatarURL() + "?size=32")
	}

	return newMember(body, m.User.ID)
}

func NewMemberUnavailable() *Member {
	body := user.New()

	return newMember(body, 0)
}

func newMember(body *user.Container, uID discord.Snowflake) *Member {
	r, _ := gtk.ListBoxRowNew()
	r.Show()
	r.Add(body)

	return &Member{
		ListBoxRow: r,
		User:       body,
		ID:         uID,
	}
}
