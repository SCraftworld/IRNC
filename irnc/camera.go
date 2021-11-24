package irnc

import (
	"context"
	"image"
	"time"
)

type Camera interface {
	VerifyConfiguration() []error
	Start(context.Context)
	Preview() (image.Image, error)
	SaveSnapshot(namePrefix string) error
	SaveVideo(namePrefix string, videoDuration time.Duration) error
}

type CameraDisposition struct {
	Width, Height uint
	RotationDegree int
}

func CreateCameraDisposition(config PhysicalDeviceConfig) (result CameraDisposition) {
	result.RotationDegree = config.RotationDegree
	result.Width = config.MaxRecordWidth
	result.Height = config.MaxRecordHeight
	if config.RotationDegree % 90 == 0 && config.RotationDegree % 180 != 0 {
		result.Width, result.Height = result.Height, result.Width
	}
	return
}
