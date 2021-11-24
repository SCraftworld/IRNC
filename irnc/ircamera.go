package irnc

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"image"
	"log"
	"os/exec"
	"syscall"
	"time"
)

type IRCamera struct {
	colorSchemeNumber uint
	seekRedirectActive bool
	V4L2Camera
}

// Get infrared camera with provided configuration
func GetIRCameraFromConfig(config *Config) *IRCamera {
	camConfig := config.IRConfig
	
	deviceDisposition := CreateCameraDisposition(camConfig.PhysicalConfig)
	return &IRCamera{
		colorSchemeNumber: camConfig.ColorSchemeNumber,
		seekRedirectActive: false,
		V4L2Camera: V4L2Camera {
			bitrate: camConfig.Bitrate,
			decoder: &RawRGBVideoDecoder{deviceDisposition.Width, deviceDisposition.Height},
			deviceNumber: camConfig.V4L2DeviceNumber,
			disposition: deviceDisposition,
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

// Do basic consistency checks for configuration values (camera/tool-specific)
func (irc *IRCamera) VerifyConfiguration() (res []error) {
	if irc.colorSchemeNumber > 21 {
		res = append(res, errors.New("Color scheme number must be between 0 and 21"))
	}
	
	res = append(res, irc.V4L2Camera.VerifyConfiguration()...)
	return
}

// Start seek_viewer so IR cam output can be redirected to v4l2 device
func (irc *IRCamera) setupIRV4L2(ctx context.Context) bool {
	irc.stateMtx.Lock()
	defer irc.stateMtx.Unlock()
	if irc.seekRedirectActive { return true }
	
	viewerCmd := exec.CommandContext(ctx, "seek_viewer", "--camtype=seekpro", fmt.Sprintf("--colormap=%d", irc.colorSchemeNumber), fmt.Sprintf("--rotate=%d", irc.disposition.RotationDegree), "--mode=v4l2", fmt.Sprintf("--output=/dev/video%d", irc.deviceNumber))
	viewerCmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGINT}
	viewerStdout, err := viewerCmd.StdoutPipe()
	if err == nil { err = viewerCmd.Start() }
	if err != nil {
		log.Println("Seek_viewer start error:", err)
		return false
	}
	deviceReady := make(chan bool)
	startupTimeout := GeneralExternalsExecutionTimeout
	go func() {
		viewerOutputScanner := bufio.NewScanner(viewerStdout)
		for viewerOutputScanner.Scan() {
			line := viewerOutputScanner.Text()
			if line == "Opened v4l2 device" {
				deviceReady<- true
				break
			}
		}
	}()
	select {
		case ready := <-deviceReady:
			if ready {
				irc.seekRedirectActive = true
				return true
			}
		case <-ctx.Done():
		case <-time.After(startupTimeout):
	}
	log.Println("Failed to start seek_viewer")
	err = viewerCmd.Process.Signal(syscall.SIGINT)
	if err != nil { log.Println("Failed seek_viewer kill error:", err) }
	return false
}

// Configure and open camera device
func (irc *IRCamera) Start(ctx context.Context) {
	// seek_viewer tends to segfault itself out of existence on my device which is suboptimal
	// restart in loop until ^ fixed; delete loop afterwards
	for !irc.setupIRV4L2(ctx) { }
	irc.V4L2Camera.Start(ctx)
}

// Take a photo and save it as png by seek_snapshot call
func (irc *IRCamera) savePngBySeekSnapshot(filename string) error {
	irc.stateMtx.Lock()
	defer irc.stateMtx.Unlock()
	cmdCtx, _ := context.WithTimeout(context.Background(), GeneralExternalsExecutionTimeout)
	return exec.CommandContext(cmdCtx, "seek_snapshot", "-t", "seekpro", "-c", fmt.Sprintf("%d", irc.colorSchemeNumber), "-r", fmt.Sprintf("%d", irc.disposition.RotationDegree), "-o", filename).Run()
}

// Take a photo and save it to file with given name prefix
func (irc *IRCamera) SaveSnapshot(namePrefix string) error {
	filename := fmt.Sprintf("%s_ir.png", namePrefix)
	log.Println("IR snapshot in", filename)
	// alt: err := irc.savePngBySeekSnapshot(filename)
	err := irc.SavePngPhotoFromV4L2(filename)
	if err == nil { log.Println("IR snapshot saved") }
	return err
}

// Record video to avi file by seek_viewer call
func (irc *IRCamera) saveAviBySeekViewer(filename string, videoDuration time.Duration) error {
	irc.stateMtx.Lock()
	defer irc.stateMtx.Unlock()
	cmdCtx, _ := context.WithTimeout(context.Background(), videoDuration + GeneralExternalsExecutionTimeout)
	cmd := exec.CommandContext(cmdCtx, "seek_viewer", "-t", "seekpro", "-c", fmt.Sprintf("%d", irc.colorSchemeNumber), "-r", fmt.Sprintf("%d", irc.disposition.RotationDegree), "-m", "file", "-o", filename)
	err := cmd.Start()
	if err != nil { return err }
	time.Sleep(videoDuration)
	return cmd.Process.Signal(syscall.SIGINT)
}

// Record video to file with given name prefix
func (irc *IRCamera) SaveVideo(namePrefix string, videoDuration time.Duration) error {
	filename := fmt.Sprintf("%s_ir.h264", namePrefix)
	log.Println("IR video in", filename)
	// alt: err := irc.saveAviBySeekViewer(filename, videoDuration)
	err := irc.SaveH264VideoFromV4L2(filename, videoDuration)
	if err == nil {	log.Println("IR video saved") }
	return err
}
