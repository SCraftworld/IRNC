package irnc

import (
	"context"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

func nowAsString() string {
	return time.Now().Format("2006.01.02_15.04.05")
}

var logFile *os.File
var nCam, irCam Camera
var camReleaseFunc func()
var camInitMtx sync.Mutex

// Prepare to work: initialize hardware, open log
func Init() {
	camInitMtx.Lock()
	logFile, err := os.OpenFile(fmt.Sprintf("%s.log", nowAsString()), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil { log.Panic("Log file opening error:", err) }
	logMW := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(logMW)
	
	config := GetHardcodedConfig()
	nCam = GetNCameraFromConfig(config)
	irCam = GetIRCameraFromConfig(config)
	
	errs := nCam.VerifyConfiguration()
	if len(errs) > 0 {
		log.Panic("NCam configuration errors:", errs)
	}
	errs = irCam.VerifyConfiguration()
	if len(errs) > 0 {
		log.Panic("IRCam configuration errors:", errs)
	}
	
	var ctx context.Context
	ctx, camReleaseFunc = context.WithCancel(context.Background())
	go nCam.Start(ctx)
	go irCam.Start(ctx)
}

// Prepare to die: deallocate resources, close log
func Finish() {
	defer camInitMtx.Unlock()
	camReleaseFunc()
	logFile.Sync()
	logFile.Close()
}

// warning: non GC-managed memory bleeds constantly (approx. 1Mb in 6min)
// logic/camera/decoder/etc removal doesn't eliminate memleak
// seems to bleed faster when GUI updates frequently
// originating from Fyne communication with Raspbian?

// Run GUI, show main window
func RunGUI() {
	app := app.New()
	w := app.NewWindow("IRNC")
	
	buttonSize := float32(100)
	buttonPaddingSize := float32(10)
	photoButton := NewSquareIconStickyButton(buttonSize, buttonPaddingSize, rscPhotoPng, func(wg *sync.WaitGroup) {
		timestamp := nowAsString()
		wg.Add(1)
		go func() {
			err := nCam.SaveSnapshot(timestamp)
			if err != nil { log.Println("NCam snapshot saving error:", err) }
			wg.Done()
		}()
		go func() {
			err := irCam.SaveSnapshot(timestamp)
			if err != nil { log.Println("IRCam snapshot saving error:", err) }
			wg.Done()
		}()
	})
	recordButton := NewSquareIconStickyButton(buttonSize, buttonPaddingSize, rscVideoPng, func(wg *sync.WaitGroup) {
		timestamp := nowAsString()
		wg.Add(1)
		go func() {
			err := nCam.SaveVideo(timestamp, RecordedVideoSize)
			if err != nil { log.Println("NCam video saving error:", err) }
			wg.Done()
		}()
		go func() {
			err := irCam.SaveVideo(timestamp, RecordedVideoSize)
			if err != nil { log.Println("IRCam video saving error:", err) }
			wg.Done()
		}()
	})
	exitButton := NewSquareIconStickyButton(buttonSize, buttonPaddingSize, rscExitPng, func(*sync.WaitGroup) {
		os.Exit(0)
	})
	buttons := container.New(layout.NewVBoxLayout(), layout.NewSpacer(), photoButton, layout.NewSpacer(), recordButton, layout.NewSpacer(), exitButton, layout.NewSpacer())
	
	minPreviewSize := fyne.Size{Width: 100, Height: 100}
	nImageWidget := NewUpdateableImage(minPreviewSize)
	irImageWidget := NewUpdateableImage(minPreviewSize)
	w.SetContent(container.New(&irncLayout{}, irImageWidget, buttons, nImageWidget))
	
	config := GetHardcodedConfig()
	for _, cameraWidgetPair := range [][]interface{}{{nCam, nImageWidget}, {irCam, irImageWidget}} {
		go func(camWidgetPair []interface{}) {
			camera := camWidgetPair[0].(Camera)
			widget := camWidgetPair[1].(*UpdateableImage)
			
			for {
				preview, err := camera.Preview()
				if err == nil {
					widget.Update(preview)
				} else {
					log.Println("Preview image retrieval error:", err)
				}
				// warning: sleep-less cycle prevents other widgets update which is suboptimal. runtime.Gosched() is not sufficient.
				time.Sleep(time.Second / time.Duration(config.PreviewFramerate))
			}
		}(cameraWidgetPair)
	}
	w.SetFullScreen(true)
	w.ShowAndRun()
}
