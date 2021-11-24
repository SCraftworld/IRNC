package irnc

import (
	"image"
	"image/color"
)

type RGBImage struct {
	data []byte
	dataWidth uint
	rect image.Rectangle
}

func (rgb *RGBImage) ColorModel() color.Model {
	return color.NRGBAModel
}

func (rgb *RGBImage) Bounds() image.Rectangle {
	return rgb.rect
}

func (rgb *RGBImage) At(x, y int) color.Color {
	offset := (x + y * int(rgb.dataWidth))*3
	return color.NRGBA{rgb.data[offset], rgb.data[offset + 1], rgb.data[offset + 2], 255}
}

func (rgb *RGBImage) SubImage(rect image.Rectangle) image.Image {
	rect = rect.Intersect(rgb.rect)
	if rect.Empty() {
		return &RGBImage{}
	}
	return &RGBImage{
		data: rgb.data[:],
		dataWidth: rgb.dataWidth,
		rect: rect,
	}
}

func (rgb *RGBImage) GetOriginalData() ([]byte, uint) {
	return rgb.data, rgb.dataWidth
}
