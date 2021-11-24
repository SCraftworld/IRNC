package irnc

import (
	"time"
)

const GeneralExternalsExecutionTimeout = 15 * time.Second
const RecordedVideoSize = 60 * time.Second

type PhysicalDeviceConfig struct {
	MaxRecordWidth, MaxRecordHeight uint
	RotationDegree int
}

type CameraConfig struct {
	Bitrate uint
	ColorSchemeNumber uint
	PhysicalConfig PhysicalDeviceConfig
	PreviewPixelDensity uint
	RecordWidth, RecordHeight uint
	V4L2DeviceNumber uint
}

type Config struct {
	NConfig, IRConfig CameraConfig
	PreviewWidth, PreviewHeight, PreviewFramerate uint
}

// Get application specific settings for preview and cameras
func GetHardcodedConfig() *Config {
	return &Config {
		NConfig: CameraConfig {
			Bitrate: 17000000,
			PhysicalConfig: PhysicalDeviceConfig {
				MaxRecordWidth: 1920,
				MaxRecordHeight: 1080,
				RotationDegree: 0,
			},
			PreviewPixelDensity: 2,
			RecordWidth: 190,
			RecordHeight: 320,
			V4L2DeviceNumber: 0,
		},
		IRConfig: CameraConfig {
			Bitrate: 17000000,
			ColorSchemeNumber: 11,
			PhysicalConfig: PhysicalDeviceConfig {
				MaxRecordWidth: 320,
				MaxRecordHeight: 240,
				RotationDegree: 90,
			},
			PreviewPixelDensity: 1,
			RecordWidth: 190,
			RecordHeight: 320,
			V4L2DeviceNumber: 1,
		},
		PreviewWidth: 190,
		PreviewHeight: 320, // actually it's 189.57031 x 312/318
		PreviewFramerate: 15,
	}
}
