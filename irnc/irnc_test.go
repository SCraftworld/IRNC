package irnc

import (
	"github.com/liyue201/goqr"
	"testing"
)

func initHardware() (nCam, irCam Camera, stopCamFn func()) {
	nCam = GetHardcodedNCamera()
	irCam = GetHardcodedIRCamera()
	
	errs := nCam.VerifyConfiguration()
	if len(errs) > 0 {
		panic("NCam configuration errors:", errs)
	}
	errs = irCam.VerifyConfiguration()
	if len(errs) > 0 {
		panic("IRCam configuration errors:", errs)
	}
	
	var ctx context.Context
	ctx, stopCamFn = context.WithCancel(context.Background())
	go nCam.Start(ctx)
	go irCam.Start(ctx)
}

func TestHardwareQR(t *testing.T) {
	nCam, irCam, stopCamFn := initHardware()
	t.Cleanup(stopCamFn)
	nPrev, err := nCam.Preview()
	if err != nil {
		t.Fatal("Normal/nightvision camera preview error", err)
	}
	irPrev, err := irCam.Preview()
	if err != nil {
		t.Fatal("Infrared camera preview error", err)
	}
	nCodes, err := goqr.Recognize(nPrev)
	if err != nil {
		t.Fatal("QR recognition failed for NCam", err)
	}
	irCodes, err := goqr.Recognize(irPrev)
	if err != nil {
		t.Fatal("QR recognition failed for IRCam", err)
	}
	if len(nCodes) > 1 || len(irCodes) > 1 {
		t.Fatal("Multiple QR codes")
	}
	nCode := string(nCodes[0].Payload)
	irCode := string(irCodes[0].Payload)
	if nCode != irCode {
		t.Fatalf("QR codes mismatch, NCam \"%s\" VS IRCam \"%s\"", nCode, irCode)
	}
}
