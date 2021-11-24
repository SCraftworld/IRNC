package irnc

import (
	"image"
	"reflect"
	"unsafe"
)

type NoPictureError struct { }

func (e NoPictureError) Error() string {
	return "No picture"
}

type TryAgainError struct { }

func (e TryAgainError) Error() string {
	return "Try again"
}

type EOFError struct { }

func (e EOFError) Error() string {
	return "EOF"
}

type Encoded struct {
	Result []byte
	Error error
}

type VideoEncoder interface {
	InitBySample(image.Image) ([]byte, error)
	Encode(image.Image) ([]byte, error)
	Destroy() error
}

// Read and encode images from channel until exhausted
func SetupChannelEncoder(encoder VideoEncoder, imageCh <-chan image.Image) <-chan Encoded {
	resCh := make(chan Encoded)
	go func() {
		initialized := false
		defer close(resCh)
		for img := range imageCh {
			if !initialized {
				initialized = true
				header, err := encoder.InitBySample(img)
				if err == nil {
					resCh<- Encoded{header, nil}
				} else {
					resCh<- Encoded{nil, err}
					return
				}
			}
			nal, err := encoder.Encode(img)
			if _, tryAgain := err.(TryAgainError); tryAgain { continue }
			if _, finished := err.(EOFError); finished { break }
			resCh<- Encoded{nal, err}
		}
		for {
			nal, err := encoder.Encode(nil)
			if err != nil { break }
			resCh<- Encoded{nal, nil}
		}
		err := encoder.Destroy()
		if err != nil {
			resCh<- Encoded{nil, err}
		}
	}()
	return resCh
}

type Decoded struct {
	Result image.Image
	Error error
}

type VideoDecoder interface {
	Init() error
	Decode([]byte) (image.Image, error)
	Destroy() error
}

// Read and decode video from channel until exhausted
func SetupChannelDecoder(decoder VideoDecoder, nalCh <-chan []byte) <-chan Decoded {
	resCh := make(chan Decoded)
	go func() {
		defer close(resCh)
		err := decoder.Init()
		if err == nil {
			for nal := range nalCh {
				img, err := decoder.Decode(nal)
				if _, noPicture := err.(NoPictureError); noPicture { continue }
				resCh<- Decoded{img, err}
			}
		} else {
			resCh<- Decoded{nil, err}
			return
		}
		err = decoder.Destroy()
		if err != nil {
			resCh<- Decoded{nil, err}
		}
	}()
	return resCh
}

// Convert unsafe pointer from C to byte slice
func CPtr2UIntSlice(buf unsafe.Pointer, size int) (res []uint8) {
	sliceHeader := (*reflect.SliceHeader)(unsafe.Pointer(&res))
	sliceHeader.Cap = size
	sliceHeader.Len = size
	sliceHeader.Data = uintptr(buf)
	return
}
