package logo

import (
	"io/ioutil"

	"github.com/gotk3/gotk3/gdk"
	"github.com/markbates/pkger"
	"github.com/pkg/errors"
)

func PNG() ([]byte, error) {
	f, err := pkger.Open("/gtkcord/components/logo/logo.png")
	if err != nil {
		return nil, errors.Wrap(err, "Failed to open logo")
	}

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to read logo")
	}

	return b, nil
}

func Pixbuf(sz int) (*gdk.Pixbuf, error) {
	b, err := PNG()
	if err != nil {
		return nil, err
	}

	l, err := gdk.PixbufLoaderNew()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create a pixbuf loader")
	}

	if sz > 0 {
		l.SetSize(sz, sz)
	}

	p, err := l.WriteAndReturnPixbuf(b)
	return p, errors.Wrap(err, "Failed to write to pixbuf")
}
