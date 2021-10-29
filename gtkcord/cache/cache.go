package cache

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/diamondburned/gotk4/pkg/cairo"
	"github.com/diamondburned/gotk4/pkg/core/gioutil"
	"github.com/diamondburned/gotk4/pkg/gdk/v3"
	"github.com/diamondburned/gotk4/pkg/gdkpixbuf/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/pkg/errors"
	"golang.org/x/sync/semaphore"
	"golang.org/x/sync/singleflight"
)

var Client = http.Client{
	Timeout: 15 * time.Second,
}

var tmpPath = filepath.Join(os.TempDir(), "gtkcord3")

// TmpPath returns the temporary path.
func TmpPath() string {
	return tmpPath
}

var (
	gomaxprocs = int64(runtime.GOMAXPROCS(-1))

	// throttler is the global HTTP throttler to fetch assets.
	throttler = semaphore.NewWeighted(gomaxprocs * 4)
	// heavyThrottler is used for GIFs and such.
	heavyThrottler = semaphore.NewWeighted(gomaxprocs)
	// fileThrottler is used for cache files.
	fileThrottler = semaphore.NewWeighted(128)

	imageRegistry sync.Map // imageKey -> *imageData
	registryMutex sync.RWMutex
)

func imageKey(hash string, w, h int) string {
	if w == 0 && h == 0 {
		return hash
	}
	return fmt.Sprintf("%s#w=%d,h=%d", hash, w, h)
}

type imageData struct {
	surface   *cairo.Surface
	animation *gdkpixbuf.PixbufAnimation

	key       string
	reference int64
}

func (d *imageData) unref() {
	if atomic.AddInt64(&d.reference, -1) <= 0 {
		imageRegistry.Delete(d.key)
	}
}

var dlFlight singleflight.Group

func hashURL(s string) string {
	u, err := url.Parse(s)
	if err != nil {
		log.Errorf("invalid image URL %q", s)
		return rehashURL(s, "")
	}

	return rehashURL(s, path.Ext(u.Path))
}

func rehashURL(s, ext string) string {
	b := sha256.Sum224([]byte(s))
	return base64.URLEncoding.EncodeToString(b[:]) + ext
}

var statThrash sync.Map

func readFromFile(path string, dst io.Writer) error {
	fileThrottler.Acquire(context.Background(), 1)
	defer fileThrottler.Release(1)

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(dst, f)
	return err
}

func downloadToFile(url, dstFile string, dst io.Writer) error {
	_, ok := statThrash.Load(dstFile)
	if ok {
		return readFromFile(dstFile, dst)
	}

	if stat, err := os.Stat(dstFile); err == nil && !stat.IsDir() {
		statThrash.Store(dstFile, true)
		return readFromFile(dstFile, dst)
	}

	if err := os.MkdirAll(tmpPath, os.ModePerm); err != nil {
		return errors.Wrap(err, "failed to mkdir tmpPath")
	}

	f, err := os.CreateTemp(tmpPath, ".downloading-*")
	if err != nil {
		return errors.Wrap(err, "failed to create image file")
	}
	defer os.Remove(f.Name())
	defer f.Close()

	// Throttle.
	throttler.Acquire(context.Background(), 1)
	defer throttler.Release(1)

	r, err := Client.Get(url)
	if err != nil {
		return errors.Wrap(err, "failed to GET")
	}
	defer r.Body.Close()

	if r.StatusCode < 200 || r.StatusCode > 299 {
		return fmt.Errorf("bad status code %d for %s", r.StatusCode, url)
	}

	if _, err = io.Copy(f, r.Body); err != nil {
		return errors.Wrap(err, "failed to download")
	}

	if err := f.Close(); err != nil {
		return errors.Wrap(err, "failed to close tmpfile")
	}

	if err := os.Rename(f.Name(), dstFile); err != nil {
		return errors.Wrap(err, "failed to rename downloaded file")
	}

	statThrash.Store(dstFile, true)
	return readFromFile(dstFile, dst)
}

func get(url string, maxW, maxH, scale int, done func(*imageData)) {
	hash := hashURL(url)
	ikey := imageKey(hash, maxW, maxH)

	// TODO: see if anything explodes if I just don't account for scaling.

	if v, ok := imageRegistry.Load(ikey); ok {
		image := v.(*imageData)
		atomic.AddInt64(&image.reference, 1)
		done(image)
		return
	}

	go func() {
		// We have to rely on ourselves to know if the image is a GIF or not,
		// because we want to scale only if the image is not a GIF.
		isGIF := strings.Contains(hash, ".gif")

		fetch := func() (interface{}, error) {
			l := gdkpixbuf.NewPixbufLoader()
			if maxW > 0 && maxH > 0 {
				l.ConnectSizePrepared(func(w, h int) {
					l.SetSize(MaxSize(w, h, maxW, maxH))
				})
			}

			w := gioutil.PixbufLoaderWriter(l)

			dstFile := filepath.Join(tmpPath, hash)
			// Fetching a cached resource shouldn't be cancelled, since it'll
			// cascade onto other callers.
			if err := downloadToFile(url, dstFile, w); err != nil {
				l.Close()
				return nil, err
			}

			if err := l.Close(); err != nil {
				return nil, errors.Wrap(err, "failed to load pixbuf")
			}

			img := &imageData{reference: 1}
			if isGIF {
				img.animation = l.Animation()
			} else {
				img.surface = gdk.CairoSurfaceCreateFromPixbuf(l.Pixbuf(), scale, nil)
			}

			imageRegistry.Store(ikey, img)

			return img, nil
		}

		img, err, _ := dlFlight.Do(ikey, func() (interface{}, error) {
			v, err := fetch()
			if err != nil {
				log.Errorf("error caching image %q URL %q: %v", hash, url, err)
				return nil, err
			}
			return v, nil
		})

		if err == nil {
			glib.IdleAdd(func() { done(img.(*imageData)) })
		}
	}()
}

// Imager describes the image type.
type Imager interface {
	gtk.Widgetter
	ScaleFactor() int
	SetFromIconName(string, int)
	SetFromPixbuf(*gdkpixbuf.Pixbuf)
	SetFromAnimation(*gdkpixbuf.PixbufAnimation)
	SetFromSurface(*cairo.Surface)
}

var _ Imager = (*gtk.Image)(nil)

func SetImageURL(img Imager, url string) {
	SetImageURLScaled(img, url, 0, 0)
}

func SetImageURLScaled(img Imager, url string, w, h int) {
	SetImageURLScaledContext(context.Background(), img, url, w, h)
}

func SetImageURLScaledContext(ctx context.Context, img Imager, url string, w, h int) {
	img.SetFromPixbuf(nil)

	get(url, w, h, img.ScaleFactor(), func(data *imageData) {
		select {
		case <-ctx.Done():
			return
		default:
			// ok
		}

		switch {
		case data.surface != nil:
			img.SetFromSurface(data.surface)
		case data.animation != nil:
			img.SetFromAnimation(data.animation)
		default:
		}

		// On image destroy, unreference the image.
		img.Connect("destroy", data.unref)
	})
}

// SetImageStreamed is async and not cached.
func SetImageStreamed(img Imager, url string, w, h int) {
	SetImageStreamedContext(context.Background(), img, url, w, h)
}

// SetImageStreamedContext is the ctx variant of SetImageStreamed.
func SetImageStreamedContext(ctx context.Context, img Imager, url string, w, h int) {
	widget := gtk.BaseWidget(img)

	baseCtx := ctx

	ctx, cancel := context.WithCancel(ctx)
	widget.ConnectUnmap(func() {
		cancel()
	})

	if !widget.Mapped() {
		widget.ConnectMap(func() {
			ctx, cancel = context.WithCancel(baseCtx)
			setImageStreamedContext(ctx, img, url, w, h)
		})
	} else {
		setImageStreamedContext(ctx, img, url, w, h)
	}
}

func setImageStreamedContext(ctx context.Context, img Imager, url string, maxW, maxH int) {
	go func() {
		gone := func(err error) {
			log.Printf("cannot stream image %s: %v", url, err)
			glib.IdleAdd(func() {
				img.SetFromIconName("image-missing", 0)
				w := gtk.BaseWidget(img)
				w.SetTooltipText(err.Error())
			})
		}

		// Throttle.
		if err := heavyThrottler.Acquire(ctx, 1); err != nil {
			gone(err)
			return
		}
		defer heavyThrottler.Release(1)

		request, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			gone(err)
			return
		}

		r, err := Client.Do(request)
		if err != nil {
			gone(err)
			return
		}
		defer r.Body.Close()

		if r.StatusCode < 200 || r.StatusCode > 299 {
			gone(fmt.Errorf("bad status code %d", r.StatusCode))
			return
		}

		loader := gdkpixbuf.NewPixbufLoader()
		if maxW > 0 && maxH > 0 {
			loader.ConnectSizePrepared(func(w, h int) {
				loader.SetSize(MaxSize(w, h, maxW, maxH))
			})
		}

		if _, err := io.Copy(gioutil.PixbufLoaderWriter(loader), r.Body); err != nil {
			gone(err)
			return
		}

		if err := loader.Close(); err != nil {
			gone(errors.Wrap(err, "pixbuf load error"))
			return
		}

		glib.IdleAdd(func() {
			animation := loader.Animation()
			if animation.IsStaticImage() {
				img.SetFromPixbuf(animation.StaticImage())
			} else {
				img.SetFromAnimation(animation)
			}
		})
	}()
}

func SizeToURL(urlstr string, w, h int) string {
	u, err := url.Parse(urlstr)
	if err != nil {
		return urlstr
	}

	val := u.Query()
	val.Set("width", strconv.Itoa(w))
	val.Set("height", strconv.Itoa(h))
	u.RawQuery = val.Encode()

	return u.String()
}

func MaxSize(w, h, maxW, maxH int) (int, int) {
	if w < maxW && h < maxH {
		return w, h
	}

	if w > h {
		h = h * maxW / w
		w = maxW
	} else {
		w = w * maxH / h
		h = maxH
	}

	return w, h
}
