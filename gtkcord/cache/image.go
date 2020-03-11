package cache

import (
	"bytes"
	"image"
	"image/draw"
	"image/gif"
	"image/png"
	"io"
	"sync"

	_ "image/jpeg"

	"github.com/diamondburned/gtkcord3/gtkcord/icons"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/disintegration/imaging"
	"github.com/pkg/errors"
)

var bufPool = sync.Pool{
	New: func() interface{} { return new(bytes.Buffer) },
}

type Processor func(image.Image) image.Image

func ProcessAnimation(data []byte, processors ...Processor) []byte {
	GIF, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		log.Errorln("Go: Failed to decode GIF:", err)
		return data
	}

	// Encode the GIF frame-by-frame
	for _, frame := range GIF.Image {
		var img = image.Image(frame)
		for _, proc := range processors {
			img = proc(img)
		}

		frame.Rect = img.Bounds()

		for x := 0; x < frame.Rect.Dx(); x++ {
			for y := 0; y < frame.Rect.Dy(); y++ {
				frame.Set(x, y, img.At(x, y))
			}
		}
	}

	buf := bufPool.Get().(*bytes.Buffer)
	defer bufPool.Put(buf)
	defer buf.Reset()

	if err := gif.EncodeAll(buf, GIF); err != nil {
		log.Errorln("Go: Failed to encode GIF:", err)
		return data
	}

	return buf.Bytes()
}

var pngEncoder = png.Encoder{
	// Prefer compression over speed, as the cache is retained for a long time.
	CompressionLevel: png.BestCompression,
}

func Process(data []byte, processors []Processor) ([]byte, error) {
	return ProcessStream(bytes.NewReader(data), processors)
}

func ProcessStream(r io.Reader, processors []Processor) ([]byte, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to decode")
	}

	for _, proc := range processors {
		img = proc(img)
	}

	buf := bufPool.Get().(*bytes.Buffer)
	defer bufPool.Put(buf)
	defer buf.Reset()

	if err := pngEncoder.Encode(buf, img); err != nil {
		return nil, errors.Wrap(err, "Failed to encode")
	}

	return buf.Bytes(), nil
}

func Prepend(p1 Processor, pN []Processor) []Processor {
	return append([]Processor{p1}, pN...)
}

func Resize(maxW, maxH int) Processor {
	return func(img image.Image) image.Image {
		bounds := img.Bounds()
		imgW, imgH := bounds.Dx(), bounds.Dy()

		w, h := maxSize(imgW, imgH, maxW, maxH)

		return imaging.Resize(img, w, h, imaging.Linear)
	}
}

func Round(img image.Image) image.Image {
	// Scale up
	oldbounds := img.Bounds()
	const scale = 2

	img = imaging.Resize(img, oldbounds.Dx()*scale, oldbounds.Dy()*scale, imaging.Lanczos)

	r := img.Bounds().Dx() / 2

	var dst draw.Image

	switch img.(type) {
	// alpha-supported:
	case *image.RGBA, *image.RGBA64, *image.NRGBA, *image.NRGBA64:
		dst = img.(draw.Image)
	default:
		dst = image.NewRGBA(image.Rect(
			0, 0,
			r*2, r*2,
		))
	}

	roundTo(img, dst, r)

	return imaging.Resize(img, oldbounds.Dx(), oldbounds.Dy(), imaging.Lanczos)
}

// RoundTo round-crops an image
func roundTo(src image.Image, dst draw.Image, r int) {
	draw.DrawMask(
		dst,
		src.Bounds(),
		src,
		image.ZP,
		icons.NewCircle(r),
		image.ZP,
		draw.Src,
	)
}

func min(i, j int) int {
	if i < j {
		return i
	}
	return j
}
