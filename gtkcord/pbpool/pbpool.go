package pbpool

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/gotk3/gotk3/gdk"
	"github.com/pkg/errors"
)

var pool sync.Map
var Client = &http.Client{
	Timeout: 5 * time.Second,
}

var MaxCacheSize = 1 * 1024 * 1024 // 4MB

func httpGet(url string) ([]byte, error) {
	r, err := Client.Get(url)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	if r.StatusCode < 200 || r.StatusCode > 299 {
		return nil, fmt.Errorf("Bad status code %d for %s", r.StatusCode, url)
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to download image")
	}

	if len(b) == 0 {
		return nil, errors.New("nil body")
	}

	return b, nil
}

func Get(url string, pp ...Processor) (*gdk.Pixbuf, error) {
	return getScaled(true, url, 0, 0, pp...)
}

// Caches
func GetScaled(url string, w, h int, pp ...Processor) (*gdk.Pixbuf, error) {
	return getScaled(true, url, w, h, pp...)
}

// Doesn't cache
func DownloadScaled(url string, w, h int, pp ...Processor) (*gdk.Pixbuf, error) {
	return getScaled(false, url, w, h, pp...)
}

func getScaled(
	cache bool, url string, w, h int, pp ...Processor) (*gdk.Pixbuf, error) {

	if cache {
		if v, ok := pool.Load(url); ok {
			pb, ok := v.(*gdk.Pixbuf)
			if !ok {
				return nil, errors.New("Image is not a pixbuf")
			}

			if w > 0 && h > 0 {
				pb, err := pb.ScaleSimple(w, h, gdk.INTERP_BILINEAR)
				return pb, errors.Wrap(err, "Failed to scale pixbuf")
			}

			return pb, nil
		}
	}

	b, err := httpGet(url)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to GET URL "+url)
	}

	if len(pp) > 0 {
		b = Process(b, pp...)
	}

	l, err := gdk.PixbufLoaderNew()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create a new pixbuf loader")
	}

	p, err := l.WriteAndReturnPixbuf(b)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to set image to pixbuf")
	}

	if len(b) <= MaxCacheSize && cache {
		pool.Store(url, p)
	}

	if w > 0 && h > 0 {
		p, err = p.ScaleSimple(w, h, gdk.INTERP_BILINEAR)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to scale pixbuf")
		}
	}

	return p, nil
}

func GetAnimation(url string, pp ...Processor) (*gdk.PixbufAnimation, error) {
	return GetAnimationScaled(url, 0, 0, pp...)
}

func GetAnimationScaled(url string, w, h int, pp ...Processor) (*gdk.PixbufAnimation, error) {
	return getAnimationScaled(true, url, w, h, pp...)
}

func DownloadAnimationScaled(url string, w, h int, pp ...Processor) (*gdk.PixbufAnimation, error) {
	return getAnimationScaled(false, url, w, h, pp...)
}

func getAnimationScaled(
	cache bool, url string, w, h int, pp ...Processor) (*gdk.PixbufAnimation, error) {

	// As PixbufAnimation doesn't allow resizing, we have to store it
	// separately.
	var key = fmt.Sprintf("%s#%d,%d", url, w, h)

	if cache {
		if v, ok := pool.Load(key); ok {
			pb, ok := v.(*gdk.PixbufAnimation)
			if !ok {
				return nil, errors.New("Image is not an animation")
			}

			return pb, nil
		}
	}

	b, err := httpGet(url)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to GET URL "+url)
	}

	if w > 0 && h > 0 {
		// We can resize the image using Go instead.
		pp = Prepend(Resize(w, h), pp...)
	}
	b = Process(b, pp...)

	l, err := gdk.PixbufLoaderNewWithType("gif")
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create a new pixbuf loader")
	}

	p, err := l.WriteAndReturnPixbufAnimation(b)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to set image to pixbuf")
	}

	if len(b) <= MaxCacheSize && cache {
		pool.Store(key, p)
	}

	return p, nil
}
