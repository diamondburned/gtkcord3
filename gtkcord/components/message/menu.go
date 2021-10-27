package message

import (
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gtkcord3/gtkcord/components/window"
	"github.com/diamondburned/gtkcord3/internal/log"
)

func (m *Messages) menuAddAdmin(msg *Message, menuContainer gtk.Containerer) {
	menu := menuContainer.BaseContainer()
	me, _ := m.c.Me()

	var canDelete = msg.AuthorID == me.ID
	if !canDelete {
		p, err := m.c.Permissions(m.ChannelID(), me.ID)
		if err != nil {
			log.Errorln("failed to get permissions:", err)
			return
		}

		canDelete = p.Has(discord.PermissionManageMessages)
	}

	if canDelete {
		iDel := gtk.NewMenuItemWithLabel("Delete Message")
		iDel.Connect("activate", func() {
			go func() {
				if err := m.c.DeleteMessage(m.ChannelID(), msg.ID); err != nil {
					log.Errorln("error deleting message:", err)
				}
			}()
		})
		iDel.Show()
		menu.Add(iDel)
	}

	if msg.AuthorID == me.ID {
		iEdit := gtk.NewMenuItemWithLabel("Edit Message")
		iEdit.Connect("activate", func() {
			m.Input.editMessage(msg.ID)
		})
		iEdit.Show()
		menu.Add(iEdit)
	}
}

func (m *Messages) menuAddDebug(msg *Message, menuContainer gtk.Containerer) {
	menu := menuContainer.BaseContainer()

	cpmsgID := gtk.NewMenuItemWithLabel("Copy Message ID")
	cpmsgID.Connect("activate", func() {
		window.Window.Clipboard.SetText(msg.ID.String(), -1)
	})
	cpmsgID.Show()
	menu.Add(cpmsgID)

	cpchID := gtk.NewMenuItemWithLabel("Copy Channel ID")
	cpchID.Connect("activate", func() {
		window.Window.Clipboard.SetText(m.ChannelID().String(), -1)
	})
	cpchID.Show()
	menu.Add(cpchID)

	if guildID := m.GuildID(); guildID.IsValid() {
		cpgID := gtk.NewMenuItemWithLabel("Copy Guild ID")
		cpgID.Connect("activate", func() {
			window.Window.Clipboard.SetText(guildID.String(), -1)
		})
		cpgID.Show()
		menu.Add(cpgID)
	}
}
