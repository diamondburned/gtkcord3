package icons

import "image"

func Solid(color uint32, w, h int) image.Image {
	copy := &image.NRGBA{
		Pix:    append([]uint8{}, folderIcon.Pix...),
		Stride: folderIcon.Stride,
		Rect:   folderIcon.Rect,
	}

	var (
		r = uint8((color >> 16) & 255)
		g = uint8((color >> 8) & 255)
		b = uint8(color & 255)
	)

	for i := 0; i < len(copy.Pix); i += 4 {
		copy.Pix[i+0] = r
		copy.Pix[i+1] = g
		copy.Pix[i+2] = b
		copy.Pix[i+3] = 255
	}

	return copy
}
