package gtkcord

import (
	"github.com/diamondburned/gtkcord3/gtkcord/components/message"
	"github.com/diamondburned/gtkcord3/gtkcord/components/preferences"
	"github.com/diamondburned/gtkcord3/gtkcord/components/window"
	"github.com/diamondburned/gtkcord3/gtkcord/config"
	"github.com/diamondburned/gtkcord3/gtkcord/md"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/diamondburned/handy"
	"github.com/gotk3/gotk3/gtk"
)

const SettingsFile = "settings.json"

type Settings struct {
	*handy.PreferencesWindow `json:"-"`

	General struct {
		*handy.PreferencesPage `json:"-"`

		Behavior struct {
			*handy.PreferencesGroup `json:"-"`

			// Embed message.Opts, which contains zerowidth and ontyping.
			message.Opts
		}

		Customization struct {
			*handy.PreferencesGroup `json:"-"`

			// only 1 file since files can import others.
			CSSFile string `json:"css_files"`

			// https://xyproto.github.io/splash/docs/all.html
			HighlightStyle string `json:"highlight_style"`

			// TODO: dark/light theme switch
		} `json:"customization"`
	} `json:"general"`

	Integrations struct {
		*handy.PreferencesPage `json:"-"`

		RichPresence struct {
			*handy.PreferencesGroup `json:"-"`

			MPRIS bool `json:"mpris"`
			// RichPresence *bool `json:"rich_presence"`
		} `json:"rich_presence"`

		// Ignore plugins in config
		Plugins struct {
			*handy.PreferencesGroup `json:"-"`
		} `json:"-"`
	} `json:"integrations"`
}

func (s *Settings) initWidgets(a *Application) {
	// Main window
	s.PreferencesWindow = handy.PreferencesWindowNew()
	s.PreferencesWindow.SetTitle("Preferences")
	s.PreferencesWindow.SetModal(true)
	s.PreferencesWindow.SetTransientFor(window.Window)
	s.PreferencesWindow.SetDefaultSize(500, 400)

	// Start connecting:
	s.PreferencesWindow.Connect("delete-event", func() bool {
		if err := config.MarshalToFile(SettingsFile, s); err != nil {
			log.Errorln("Failed to save config:", err)
		}

		// Manually handle hiding the dialog:
		s.Hide()
		return true
	})

	{
		p := &s.General

		p.PreferencesPage = handy.PreferencesPageNew()
		p.PreferencesPage.SetIconName("preferences-system-symbolic")
		p.PreferencesPage.SetTitle("General")

		{
			g := &p.Behavior

			g.PreferencesGroup = handy.PreferencesGroupNew()
			g.PreferencesGroup.SetTitle("Behavior")

			ontp, _ := gtk.SwitchNew()
			preferences.BindSwitch(ontp, &g.InputOnTyping, func() {
				// This might be called before Ready, so we should have this
				// check.
				if a.Messages != nil {
					a.Messages.InputOnTyping = g.InputOnTyping
				}
			})

			g.Add(preferences.Row(
				"Send typing events",
				"Announce that you're typing in a channel.",
				ontp,
			))

			zwsp, _ := gtk.SwitchNew()
			preferences.BindSwitch(zwsp, &g.InputZeroWidth, func() {
				if a.Messages != nil {
					a.Messages.InputZeroWidth = g.InputZeroWidth
				}
			})

			g.Add(preferences.Row(
				"Insert zero-width spaces",
				"\"Obfuscate\" sent messages with zero-width spaces to reduce telemetry.",
				zwsp,
			))
		}

		{
			g := &p.Customization

			g.PreferencesGroup = handy.PreferencesGroupNew()
			g.PreferencesGroup.SetTitle("Customization")

			// Permit only CSS files by MIME type.
			cssFilter, _ := gtk.FileFilterNew()
			cssFilter.SetName("CSS Files")
			cssFilter.AddMimeType("text/css")

			cfileW, _ := gtk.FileChooserButtonNew("Select CSS", gtk.FILE_CHOOSER_ACTION_OPEN)
			cfileW.AddFilter(cssFilter)
			preferences.BindFileChooser(cfileW, &g.CSSFile, func() {
				window.FileCSS = s.General.Customization.CSSFile
				window.ReloadCSS()
			})

			const cssInfo = `Refer to the <a href="` +
				"https://developer.gnome.org/gtk3/stable/chap-css-overview.html" + `">` +
				"Gtk+ CSS Overview" + "</a>" + "."

			g.Add(preferences.Row(
				"Custom CSS File",
				cssInfo,
				cfileW,
			))

			hlEntry, _ := gtk.EntryNew()
			hlEntry.SetHExpand(true)
			hlEntry.SetPlaceholderText("Fallback highlighting")
			preferences.BindEntry(hlEntry, &g.HighlightStyle, func() {
				if err := md.ChangeStyle(g.HighlightStyle); err != nil {
					// shitty error icon at the end
					hlEntry.SetIconFromIconName(gtk.ENTRY_ICON_SECONDARY, "dialog-error")
					hlEntry.SetIconTooltipText(gtk.ENTRY_ICON_SECONDARY, err.Error())
				} else {
					hlEntry.RemoveIcon(gtk.ENTRY_ICON_SECONDARY)
				}
			})

			const schemeInfo = `Refer to the <a href="` +
				"https://xyproto.github.io/splash/docs/all.html" + `">` +
				"Chroma Style Gallery" + "</a>" + "."

			g.Add(preferences.Row(
				"Highlight color scheme",
				schemeInfo,
				hlEntry,
			))
		}

		p.Add(p.Behavior)
		p.Add(p.Customization)
	}

	{
		p := &s.Integrations

		p.PreferencesPage = handy.PreferencesPageNew()
		p.PreferencesPage.SetIconName("package-x-generic-symbolic")
		p.PreferencesPage.SetTitle("Integrations")

		{
			g := &p.RichPresence

			g.PreferencesGroup = handy.PreferencesGroupNew()
			g.PreferencesGroup.SetTitle("Rich Presence")

			mpris, _ := gtk.SwitchNew()
			mpris.SetHAlign(gtk.ALIGN_END)
			preferences.BindSwitch(mpris, &g.MPRIS, func() {
				a.MPRIS.SetEnabled(g.MPRIS)
			})

			g.Add(preferences.Row(
				"MPRIS D-Bus",
				"Broadcast currently played songs to Discord.",
				mpris,
			))
		}

		{
			g := &p.Plugins

			g.PreferencesGroup = handy.PreferencesGroupNew()
			g.PreferencesGroup.SetTitle("Plugins")
			g.PreferencesGroup.ShowAll()
		}

		p.Add(p.RichPresence)
		p.Add(p.Plugins)
	}

	s.Add(s.General)
	s.Add(s.Integrations)

	// Just for sure:
	s.General.ShowAll()
	s.Integrations.ShowAll()
}

func (a *Application) makeSettings() *Settings {
	s := &Settings{}
	s.General.Behavior.InputOnTyping = true
	s.General.Customization.HighlightStyle = "monokai"
	s.Integrations.RichPresence.MPRIS = true

	if err := config.UnmarshalFromFile(SettingsFile, s); err != nil {
		log.Errorln("Failed to load settings, using default. Error:", err)
	}

	s.initWidgets(a)
	return s
}
