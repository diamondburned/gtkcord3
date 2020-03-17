package ningen

import (
	"sync"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/gtkcord3/log"
)

type MemberListState struct {
	mu    sync.Mutex
	state map[discord.Snowflake]*MemberList

	// All mutex guarded
	OnOP   func(m *MemberList, guild discord.Snowflake, op gateway.GuildMemberListOp)
	OnSync func(m *MemberList, guild discord.Snowflake)
}

type MemberList struct {
	mu sync.Mutex

	ID          string
	MemberCount uint64
	OnlineCount uint64

	Groups []gateway.GuildMemberListGroup
	Items  []*gateway.GuildMemberListOpItem
}

func NewMemberListState() *MemberListState {
	return &MemberListState{
		state: map[discord.Snowflake]*MemberList{},
	}
}

func (m *MemberList) Acquire() func() {
	m.mu.Lock()
	return m.mu.Unlock
}

func (m *MemberListState) GetMemberList(guild discord.Snowflake) *MemberList {
	m.mu.Lock()
	defer m.mu.Unlock()

	ml, ok := m.state[guild]
	if !ok {
		return nil
	}

	return ml
}

func (m *MemberListState) handle(ev *gateway.GuildMemberListUpdate) {
	if ev.ID != "everyone" {
		log.Errorln("Invalid member list ID:", ev.ID)
		return
	}

	m.mu.Lock()

	ml, ok := m.state[ev.GuildID]
	if !ok {
		ml = &MemberList{}
		m.state[ev.GuildID] = ml
	}
	ml.mu.Lock()
	defer ml.mu.Unlock()

	m.mu.Unlock()

	ml.ID = ev.ID
	ml.MemberCount = ev.MemberCount
	ml.OnlineCount = ev.OnlineCount
	ml.Groups = ev.Groups

	synced := false

	for _, op := range ev.Ops {
		if op.Op == "SYNC" {
			start, end := op.Range[0], op.Range[1]
			length := end + 1
			growItems(&ml.Items, length)

			for i := 0; i < length-start && i < len(op.Items); i++ {
				ml.Items[start+i] = &op.Items[i]
			}

			synced = true
			continue
		}

		// https://github.com/golang/go/wiki/SliceTricks
		i := op.Index

		// Bounds check
		if len(ml.Items) > 0 && i != 0 {
			var length = len(ml.Items)
			if op.Op == "INSERT" {
				length++
			}

			if length <= i {
				log.Errorf(
					"Member %s: index out of range: len(ml.Items)=%d <= op.Index=%d\n",
					op.Op, len(ml.Items), i,
				)
				continue
			}
		}

		switch op.Op {
		case "UPDATE":
			ml.Items[i] = &op.Item

		case "DELETE":
			if i < len(ml.Items)-1 {
				copy(ml.Items[i:], ml.Items[i+1:])
			}
			ml.Items[len(ml.Items)-1] = nil
			ml.Items = ml.Items[:len(ml.Items)-1]

		case "INSERT":
			ml.Items = append(ml.Items, nil)
			copy(ml.Items[i+1:], ml.Items[i:])
			ml.Items[i] = &op.Item
		}

		if m.OnOP != nil {
			m.OnOP(ml, ev.GuildID, op)
		}
	}

	if synced {
		m.OnSync(ml, ev.GuildID)
	}
}

func growItems(items *[]*gateway.GuildMemberListOpItem, maxLen int) {
	if len(*items) >= maxLen {
		return
	}
	delta := maxLen - len(*items)
	*items = append(*items, make([]*gateway.GuildMemberListOpItem, delta)...)
}
