package irnc

import (
	"context"
	"fmt"
	"image"
	"log"
	"os/exec"
	"time"
)

type NCamera struct {
	V4L2Camera
}

// Get normal/nightvision camera camera with provided configuration
func GetNCameraFromConfig(config *Config) *NCamera {
	camConfig := config.NConfig
	
	return &NCamera{
		V4L2Camera: V4L2Camera {
			bitrate: camConfig.Bitrate,
			decoder: &H264Decoder{},
			deviceNumber: camConfig.V4L2DeviceNumber,
			disposition: CreateCameraDisposition(camConfig.PhysicalConfig),
			frameReceivers: make(map[string]frameReceivingCommunicationPack),
			framerate: config.PreviewFramerate,
			lastImageCh: make(chan image.Image),
			previewWidth: config.PreviewWidth,
			previewHeight: config.PreviewHeight,
			previewPixelDensity: camConfig.PreviewPixelDensity,
			recordWidth: camConfig.RecordWidth,
			recordHeight: camConfig.RecordHeight,
		},
	}
}

func (nc *NCamera) Start(ctx context.Context) {
	nc.stateMtx.Lock()
	cmdCtx, _ := context.WithTimeout(ctx, GeneralExternalsExecutionTimeout)
	err := exec.CommandContext(cmdCtx, "v4l2-ctl", "-d", fmt.Sprintf("%d", nc.deviceNumber), fmt.Sprintf("--set-ctrl=rotate=%d", nc.disposition.RotationDegree), "-p", fmt.Sprintf("%d", nc.framerate)).Run()
	nc.stateMtx.Unlock()
	if err != nil { log.Panic("NCamera configuration error:", err) }
	nc.V4L2Camera.Start(ctx)
}

// Take a photo and save it as png file using raspistill call
func (nc *NCamera) savePngByRaspistill(filename string) error {
	nc.stateMtx.Lock()
	defer nc.stateMtx.Unlock()
	cmdCtx, _ := context.WithTimeout(context.Background(), GeneralExternalsExecutionTimeout)
	return exec.CommandContext(cmdCtx, "raspistill", "-n", "-rot", fmt.Sprintf("%d", nc.disposition.RotationDegree), "-e", "png", "-o", filename).Run()
}

// Take a photo and save it to file with given name prefix
func (nc *NCamera) SaveSnapshot(namePrefix string) error {
	filename := fmt.Sprintf("%s_n.png", namePrefix)
	log.Println("N snapshot in", filename)
	// alt: err := nc.savePngByRaspistill(filename)
	err := nc.SavePngPhotoFromV4L2(filename)
	if err == nil { log.Println("N shapshot saved") }
	return err
}

// Record video to h264 file via raspivid call
func (nc *NCamera) saveH264ByRaspivid(filename string, videoDuration time.Duration) error {
	nc.stateMtx.Lock()
	defer nc.stateMtx.Unlock()
	cmdCtx, _ := context.WithTimeout(context.Background(), videoDuration + GeneralExternalsExecutionTimeout)
	return exec.CommandContext(cmdCtx, "raspivid", "-n", "-rot", fmt.Sprintf("%d", nc.disposition.RotationDegree), "-t", fmt.Sprintf("%d", videoDuration.Milliseconds()), "-o", filename).Run()
}

// Record video to file with given name prefix
func (nc *NCamera) SaveVideo(namePrefix string, videoDuration time.Duration) error {
	filename := fmt.Sprintf("%s_n.h264", namePrefix)
	log.Println("N video in", filename)
	// alt: err := nc.saveH264ByRaspivid(filename, videoDuration)
	err := nc.SaveH264VideoFromV4L2(filename, videoDuration)
	if err == nil { log.Println("N video saved") }
	return err
}
