package cache

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/png"
	"io"

	_ "image/jpeg"

	"github.com/diamondburned/gtkcord3/gtkcord/icons"
	"github.com/disintegration/imaging"
	"github.com/pkg/errors"
)

type Processor func(image.Image) image.Image

func ProcessAnimationStream(r io.Reader, processors []Processor) ([]byte, error) {
	GIF, err := gif.DecodeAll(r)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to decode GIF")
	}

	// Add transparency:
	if p, ok := GIF.Config.ColorModel.(color.Palette); ok {
		GIF.Config.ColorModel = ensurePaletteTransparent(p)
	}

	// Encode the GIF frame-by-frame
	for _, frame := range GIF.Image {
		var img = image.Image(frame)
		for _, proc := range processors {
			img = proc(img)
		}

		frame.Rect = img.Bounds()

		if frame.Palette != nil {
			frame.Palette = ensurePaletteTransparent(frame.Palette)
		}

		for x := 0; x < frame.Rect.Dx(); x++ {
			for y := 0; y < frame.Rect.Dy(); y++ {
				frame.Set(x, y, img.At(x, y))
			}
		}
	}

	if len(GIF.Image) > 0 {
		bounds := GIF.Image[0].Bounds()
		GIF.Config.Width = bounds.Dx()
		GIF.Config.Height = bounds.Dy()
	}

	var buf = new(bytes.Buffer)

	if err := gif.EncodeAll(buf, GIF); err != nil {
		return nil, errors.Wrap(err, "Failed to encode GIF")
	}

	return buf.Bytes(), nil
}

func ensurePaletteTransparent(palette color.Palette) color.Palette {
	// TODO: properly quantize
	if len(palette) > 255 {
		palette = palette[:255]
	}
	palette = append(palette, color.Transparent)

	return palette
}

var pngEncoder = png.Encoder{
	// Prefer speed over compression, since cache is slightly more optimized
	// now.
	CompressionLevel: png.NoCompression,
}

func ProcessStream(r io.Reader, processors []Processor) ([]byte, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to decode")
	}

	for _, proc := range processors {
		img = proc(img)
	}

	var buf = new(bytes.Buffer)

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

		w, h := MaxSize(imgW, imgH, maxW, maxH)

		return imaging.Resize(img, w, h, imaging.Lanczos)
	}
}

func Round(img image.Image) image.Image {
	// Scale up
	oldbounds := img.Bounds()
	const scale = 2

	// only bother anti-aliasing if it's not a paletted image.
	var _, paletted = img.(*image.Paletted)
	if !paletted {
		img = imaging.Resize(img, oldbounds.Dx()*scale, oldbounds.Dy()*scale, imaging.Lanczos)
	}

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

	if paletted {
		return dst
	}

	return imaging.Resize(dst, oldbounds.Dx(), oldbounds.Dy(), imaging.Lanczos)
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
