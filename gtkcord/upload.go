package gtkcord

import (
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

type Uploader struct {
	*gtk.FileChooserNativeDialog
	callback func(string)
}

func NewUploader(callback func(path string)) (*Uploader, error) {
	dialog := must(
		gtk.FileChooserNativeDialogNew, "Upload File",
		App.Window, gtk.FILE_CHOOSER_ACTION_OPEN, "Upload", "Cancel").(*gtk.FileChooserNativeDialog)
	must(dialog.SetCurrentFolder, glib.GetUserDataDir())
	must(dialog.Connect, "response", func() {
		dialog.Destroy()
	})

	return &Uploader{
		FileChooserNativeDialog: dialog,
		callback:                callback,
	}, nil
}

// Spawn should be running in a goroutine.
func (u *Uploader) Spawn() {
	must(u.Show)

	resCode := must(u.Run).(int)
	if gtk.ResponseType(resCode) != gtk.RESPONSE_ACCEPT {
		return
	}

}
