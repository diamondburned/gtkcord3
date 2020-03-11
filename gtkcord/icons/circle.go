package icons

import (
	"image"
	"image/color"
)

type Circle struct {
	r int
}

func NewCircle(radius int) Circle {
	return Circle{
		r: radius,
	}
}

func (c Circle) ColorModel() color.Model {
	return color.AlphaModel
}

func (c Circle) Bounds() image.Rectangle {
	return image.Rect(0, 0, c.r*2, c.r*2)
}

func (c Circle) At(x, y int) color.Color {
	xx, yy, rr := float64(x-c.r)+0.5, float64(y-c.r)+0.5, float64(c.r)

	delta := (xx*xx + yy*yy) - rr*rr
	if delta <= 0 {
		return color.Alpha{255}
	}

	return color.Alpha{0}
}
