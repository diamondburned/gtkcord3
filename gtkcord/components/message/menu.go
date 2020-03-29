package message

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/components/window"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/gotk3/gotk3/gtk"
)

func (m *Messages) menuAddAdmin(msg *Message, menu gtkutils.Container) {
	var canDelete = msg.AuthorID == m.c.Ready.User.ID
	if !canDelete {
		p, err := m.c.Permissions(m.GetChannelID(), m.c.Ready.User.ID)
		if err != nil {
			log.Errorln("Failed to get permissions:", err)
			return
		}

		canDelete = p.Has(discord.PermissionManageMessages)
	}

	if canDelete {
		iDel, _ := gtk.MenuItemNewWithLabel("Delete Message")
		iDel.Connect("activate", func() {
			go func() {
				if err := m.c.DeleteMessage(m.GetChannelID(), msg.ID); err != nil {
					log.Errorln("Error deleting message:", err)
				}
			}()
		})
		iDel.Show()
		menu.Add(iDel)
	}

	if msg.AuthorID == m.c.Ready.User.ID {
		iEdit, _ := gtk.MenuItemNewWithLabel("Edit Message")
		iEdit.Connect("activate", func() {
			go func() {
				if err := m.Input.editMessage(msg.ID); err != nil {
					log.Errorln("Error editing message:", err)
				}
			}()
		})
		iEdit.Show()
		menu.Add(iEdit)
	}
}

func (m *Messages) menuAddDebug(msg *Message, menu gtkutils.Container) {
	cpmsgID, _ := gtk.MenuItemNewWithLabel("Copy Message ID")
	cpmsgID.Connect("activate", func() {
		window.Window.Clipboard.SetText(msg.ID.String())
	})
	cpmsgID.Show()
	menu.Add(cpmsgID)

	cpchID, _ := gtk.MenuItemNewWithLabel("Copy Channel ID")
	cpchID.Connect("activate", func() {
		window.Window.Clipboard.SetText(m.GetChannelID().String())
	})
	cpchID.Show()
	menu.Add(cpchID)

	if m.GetGuildID().Valid() {
		cpgID, _ := gtk.MenuItemNewWithLabel("Copy Guild ID")
		cpgID.Connect("activate", func() {
			window.Window.Clipboard.SetText(m.GetGuildID().String())
		})
		cpgID.Show()
		menu.Add(cpgID)
	}
}
