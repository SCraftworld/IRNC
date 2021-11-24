package irnc

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"sync"
	"time"
	v4l2 "github.com/thinkski/go-v4l2"
)

// TODO: capture video without reencoding

// since v4l2 provides copy-free buffers with manual release function it's imperative to know when the frame is no longer needed
type frameWithWg struct {
	Frame v4l2.Buffer
	FrameProcessed *sync.WaitGroup
}

type frameReceivingCommunicationPack struct {
	// channel for frames
	FrameCh chan frameWithWg
	// channel to singal end of receiving
	ReceivingDoneCh <-chan struct{}
}

type V4L2Camera struct {
	bitrate uint
	decoder VideoDecoder
	device *v4l2.Device
	deviceNumber uint
	disposition CameraDisposition
	frameReceivers map[string]frameReceivingCommunicationPack
	framerate uint
	lastImageCh chan image.Image
	previewWidth, previewHeight uint
	previewPixelDensity uint
	recordWidth, recordHeight uint
	stateMtx sync.Mutex
}

// Do basic consistency checks for configuration values (camera/tool-specific)
func (v4l2c *V4L2Camera) VerifyConfiguration() (res []error) {
	if v4l2c.decoder == nil {
		res = append(res, errors.New("V4L2Camera requires initialized video decoder"))
	}
	if v4l2c.disposition.RotationDegree % 90 != 0 {
		res = append(res, errors.New("Rotation must be integer divisible by 90"))
	}
	if v4l2c.disposition.Width < v4l2c.recordWidth {
		res = append(res, errors.New(fmt.Sprintf("Max device width exceeded by record width: %d < %d", v4l2c.disposition.Width, v4l2c.recordWidth)))
	}
	if v4l2c.disposition.Width < v4l2c.previewWidth * v4l2c.previewPixelDensity {
		res = append(res, errors.New(fmt.Sprintf("Max device width exceeded by preview width: %d < %d * %d", v4l2c.disposition.Width, v4l2c.previewWidth, v4l2c.previewPixelDensity)))
	}
	if v4l2c.disposition.Height < v4l2c.recordHeight {
		res = append(res, errors.New(fmt.Sprintf("Max device height exceeded by record height: %d < %d", v4l2c.disposition.Height, v4l2c.recordHeight)))
	}
	if v4l2c.disposition.Height < v4l2c.previewHeight * v4l2c.previewPixelDensity {
		res = append(res, errors.New(fmt.Sprintf("Max device height exceeded by preview height: %d < %d * %d", v4l2c.disposition.Height, v4l2c.previewHeight, v4l2c.previewPixelDensity)))
	}
	if v4l2c.framerate == 0 {
		res = append(res, errors.New("Framerate must be positive"))
	}
	return
}

// Configure channel which will inexhaustibly return last video decode result as image
func (v4l2c *V4L2Camera) setupImageChannel(imageCh chan<- image.Image, ctx context.Context) {
	frameCh := make(chan frameWithWg)
	v4l2c.frameReceivers["lastImage"] = frameReceivingCommunicationPack{FrameCh: frameCh, ReceivingDoneCh: ctx.Done()}
	
	err := v4l2c.decoder.Init()
	if err != nil { log.Panic("Decoder initialization error:", err) }
	updatedImageCh := make(chan image.Image)
	go func() {
		var img image.Image
		select {
			case <-ctx.Done():
				return
			case img = <-updatedImageCh:
		}
		for {
			select {
				case <-ctx.Done():
					return
				case img = <-updatedImageCh:
				case imageCh<- img:
			}
		}
	}()
	
	go func() {
		defer v4l2c.decoder.Destroy()
		for {
			select {
				case <-ctx.Done():
					return
				case frame := <-frameCh:
					img, err := v4l2c.decoder.Decode(frame.Frame.Data)
					frame.FrameProcessed.Done()
					if _, noPicture := err.(NoPictureError); noPicture {
						continue
					}
					if err == nil {
						updatedImageCh<- img
					} else {
						log.Println("LastImage update error:", err)
					}
			}
		}
	}()
}

// Add frames handler to receivers list
func (v4l2c *V4L2Camera) addFrameReceiver(id string, frameCh chan frameWithWg, receivingDoneCh <-chan struct{}) {
	v4l2c.stateMtx.Lock()
	defer v4l2c.stateMtx.Unlock()
	v4l2c.frameReceivers[id] = frameReceivingCommunicationPack{FrameCh: frameCh, ReceivingDoneCh: receivingDoneCh}
}

// Remove frames handler from receivers list (done signal must be sent by receiver themself)
func (v4l2c *V4L2Camera) removeFrameReceiver(id string) {
	v4l2c.stateMtx.Lock()
	defer v4l2c.stateMtx.Unlock()
	delete(v4l2c.frameReceivers, id)
}

// Configure and open camera device
func (v4l2c *V4L2Camera) Start(ctx context.Context) {
	v4l2c.stateMtx.Lock()
	defer v4l2c.stateMtx.Unlock()
	
	v4l2c.setupImageChannel(v4l2c.lastImageCh, ctx)
	var err error
	v4l2c.device, err = v4l2.Open(fmt.Sprintf("/dev/video%d", v4l2c.deviceNumber))
	if err != nil { log.Panic("V4L2 device opening error:", err) }
	
	v4l2c.device.SetPixelFormat(int(v4l2c.recordWidth), int(v4l2c.recordHeight), v4l2.V4L2_PIX_FMT_H264)
	v4l2c.device.SetBitrate(int32(v4l2c.bitrate))
	v4l2c.device.Start()
	
	go func() {
		for {
			var frame v4l2.Buffer
			select {
				case <-ctx.Done():
					v4l2c.stateMtx.Lock()
					defer v4l2c.stateMtx.Unlock()
					err := v4l2c.device.Close()
					if err != nil { log.Println("V4L2 device closing error:", err) }
					return
				case frame = <-v4l2c.device.C:
			}
			
			var frameHandlersWg sync.WaitGroup
			v4l2c.stateMtx.Lock()
			frameHandlersWg.Add(len(v4l2c.frameReceivers))
			for _, frcp := range v4l2c.frameReceivers {
				go func(frcp frameReceivingCommunicationPack, frame v4l2.Buffer) {
					select {
						case <-frcp.ReceivingDoneCh:
							frameHandlersWg.Done()
						case frcp.FrameCh<- frameWithWg{frame, &frameHandlersWg}:
					}
				}(frcp, frame)
			}
			v4l2c.stateMtx.Unlock()
			
			// wait for handlers to finish to avoid congestion and safely release shared memory in frame buffer
			frameHandlersWg.Wait()
			frame.Release()
		}
	}()
}

// Get cropped photo suitable for preview
func (v4l2c *V4L2Camera) Preview() (preview image.Image, err error) {
	originalImage := <-v4l2c.lastImageCh
	if originalImage == nil {
		err = errors.New("No preview available")
		return
	}
	originalBounds := originalImage.Bounds()
	pw := v4l2c.previewWidth * v4l2c.previewPixelDensity
	ph := v4l2c.previewHeight * v4l2c.previewPixelDensity
	minPreviewX := (originalBounds.Max.X - originalBounds.Min.X - int(pw))/2 + originalBounds.Min.X
	minPreviewY := (originalBounds.Max.Y - originalBounds.Min.Y - int(ph))/2 + originalBounds.Min.Y
	
	previewBounds := image.Rect(minPreviewX, minPreviewY, minPreviewX + int(pw), minPreviewY + int(ph))
	switch originalImage.(type) {
		case *image.YCbCr:
			preview = originalImage.(*image.YCbCr).SubImage(previewBounds)
		case *RGBImage:
			preview = originalImage.(*RGBImage).SubImage(previewBounds)
		default:
			// low tolerance for unnoticed unimplemented cases
			log.Panicf("Preview for image type %T not implemented", originalImage)
	}
	return
}

// Take single image from v4l2 video stream and save it to png file
func (v4l2c *V4L2Camera) SavePngPhotoFromV4L2(filename string) error {
	outputFile, err := os.Create(filename)
	if err != nil { return err }
	defer func() {
		err = outputFile.Close()
		if err != nil { log.Println("Snapshot file closing error:", err) }
	}()
	img := <-v4l2c.lastImageCh
	if img == nil { return errors.New("No image available.") }
	return png.Encode(outputFile, img)
}

// Take a photo and save it to file with given name prefix
func (v4l2c *V4L2Camera) SaveSnapshot(namePrefix string) error {
	return errors.New("*V4L2Camera.SaveSnapshot is unimplemented. Use SavePngPhotoFromV4L2 in embedders")
}

// Record video from v4l2 video device to h264 file by encoding snapshot sequence
func (v4l2c *V4L2Camera) SaveH264VideoFromV4L2(filename string, videoDuration time.Duration) error {
	outputFile, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil { return err }
	defer func() {
		err = outputFile.Close()
		if err != nil { log.Println("Video file closing error:", err) }
	}()
	
	imagesToEncodeCh := make(chan image.Image)
	encodedCh := SetupChannelEncoder(&H264Encoder{bitrate: v4l2c.bitrate, framerate: v4l2c.framerate}, imagesToEncodeCh)
	go func() {
		videoEndCh := time.After(videoDuration)
		for {
			select {
				case img := <-v4l2c.lastImageCh:
					imagesToEncodeCh<- img
					time.Sleep(time.Second / time.Duration(v4l2c.framerate))
				case <-videoEndCh:
					close(imagesToEncodeCh)
					return
			}
		}
	}()
	for encoded := range encodedCh {
		err := encoded.Error
		if err == nil {
			_, err = outputFile.Write(encoded.Result)
		}
		if err != nil { log.Println("Video file writing error:", err) }
	}
	return nil
}

// Record video to file with given name prefix
func (v4l2c *V4L2Camera) SaveVideo(namePrefix string, videoDuration time.Duration) error {
	return errors.New("*V4L2Camera.SaveVideo is unimplemented. Use SaveH264VideoFromV4L2 in embedders")
}
