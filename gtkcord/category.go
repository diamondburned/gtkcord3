package gtkcord

import (
	"sort"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/state"
	"github.com/pkg/errors"
)

type _sortStructure struct {
	parent    discord.Channel
	hasParent bool
	children  []discord.Channel
}

func filterChannels(s *state.State, chs []discord.Channel) []discord.Channel {
	filtered := chs[:0]

	u, err := s.Me()
	if err != nil {
		return chs
	}

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

			break

		default:
			continue
		}

		filtered = append(filtered, ch)
	}

	return filtered
}

func transformChannels(widget *Channels, chs []discord.Channel) error {
	var tree = map[discord.Snowflake]*_sortStructure{}

	for _, ch := range chs {
		if ch.Type == discord.GuildCategory {
			v, ok := tree[ch.ID]
			if ok {
				v.parent = ch
				v.hasParent = true
			} else {
				tree[ch.ID] = &_sortStructure{
					parent:    ch,
					hasParent: true,
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
			parent:    ch,
			hasParent: true,
		}
	}

	var list = make([]*_sortStructure, 0, len(tree))

	for _, v := range tree {
		if v.children != nil {
			sort.Slice(v.children, func(i, j int) bool {
				return v.children[i].Position < v.children[j].Position
			})
		}

		list = append(list, v)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].parent.Position < list[j].parent.Position
	})

	widget.Channels = make([]*Channel, 0, len(chs))

	for _, sch := range list {
		if sch.hasParent {
			w, err := newChannel(sch.parent)
			if err != nil {
				return errors.Wrap(err, "Failed to create sch.parent")
			}

			widget.Channels = append(widget.Channels, w)
		}

		for _, ch := range sch.children {
			w, err := newChannel(ch)
			if err != nil {
				return errors.Wrap(err, "Failed to create children channel")
			}

			widget.Channels = append(widget.Channels, w)
		}
	}

	return nil
}
