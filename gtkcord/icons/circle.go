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
	if CircleAt(c.r, x, y) {
		return color.Alpha{255}
	}
	return color.Alpha{0}
}

func CircleAt(radius, x, y int) bool {
	xx, yy, rr := float64(x-radius)+0.5, float64(y-radius)+0.5, float64(radius)
	if xx*xx+yy*yy < rr*rr {
		return true
	}
	return false
}
