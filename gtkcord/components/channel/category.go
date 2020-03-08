package channel

import (
	"sort"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/state"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/log"
)

type _sortStructure struct {
	parent   discord.Channel
	children []discord.Channel
}

func filterChannels(s *state.State, chs []discord.Channel) []discord.Channel {
	filtered := make([]discord.Channel, 0, len(chs))
	u := s.Ready.User

	for _, ch := range chs {
		p, err := s.Permissions(ch.ID, u.ID)
		if err != nil {
			continue
		}

		if !p.Has(discord.PermissionReadMessageHistory) {
			continue
		}

		switch ch.Type {
		case discord.DirectMessage,
			discord.GuildText,
			discord.GuildCategory,
			discord.GroupDM:

		default:
			continue
		}

		filtered = append(filtered, ch)
	}

	return filtered
}

func transformChannels(s *ningen.State, chs []discord.Channel) []*Channel {
	var tree = map[discord.Snowflake]*_sortStructure{}

	for _, ch := range chs {
		if ch.Type == discord.GuildCategory {
			v, ok := tree[ch.ID]
			if ok {
				v.parent = ch
			} else {
				tree[ch.ID] = &_sortStructure{
					parent: ch,
				}
			}

			continue
		}

		if ch.CategoryID.Valid() {
			v, ok := tree[ch.CategoryID]
			if ok {
				v.children = append(v.children, ch)
			} else {
				tree[ch.CategoryID] = &_sortStructure{
					children: []discord.Channel{ch},
				}
			}

			continue
		}

		tree[ch.ID] = &_sortStructure{
			parent: ch,
		}
	}

	var list = make([]*_sortStructure, 0, len(tree))

	for _, v := range tree {
		if v.children != nil {
			sort.SliceStable(v.children, func(i, j int) bool {
				return v.children[i].Position < v.children[j].Position
			})
		}

		list = append(list, v)
	}

	sort.SliceStable(list, func(i, j int) bool {
		return list[i].parent.Position < list[j].parent.Position
	})

	sort.SliceStable(list, func(i, j int) bool {
		return list[i].children == nil
	})

	var channels = make([]*Channel, 0, len(chs))

	semaphore.IdleMust(func() {
		for _, sch := range list {
			sch := sch

			log.Println("Category:", sch.parent.Name)

			if sch.parent.ID.Valid() {
				channels = append(channels, createChannelRead(&sch.parent, s))
			}

			for i := range sch.children {
				log.Println("Channel:", sch.children[i].Name)
				channels = append(channels, createChannelRead(&sch.children[i], s))
			}
		}
	})

	return channels
}
