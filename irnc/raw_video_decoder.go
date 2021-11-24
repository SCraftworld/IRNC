package irnc

import (
	"image"
)

type RawRGBVideoDecoder struct {
	frameWidth uint
	frameHeight uint
}

func (decoder *RawRGBVideoDecoder) Init() error {
	return nil
}

func (decoder *RawRGBVideoDecoder) Decode(frame []byte) (image.Image, error) {
	rgbImage := RGBImage{
		data: make([]byte, len(frame)),
		dataWidth: decoder.frameWidth,
		rect: image.Rectangle {image.Point{0, 0}, image.Point{int(decoder.frameWidth), int(decoder.frameHeight)}},
	}
	copy(rgbImage.data, frame)
	return &rgbImage, nil
}

func (decoder *RawRGBVideoDecoder) Destroy() error {
	return nil
}