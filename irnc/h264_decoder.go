package irnc

/*
#include <libavcodec/avcodec.h>
#include <libavformat/avformat.h>
#include <libavutil/avutil.h>

typedef struct {
	AVCodec *codec;
	AVCodecContext *context;
	AVFrame *frame;
	AVPacket *packet;
} h264decoder_t ;

#cgo pkg-config: libavcodec libavutil libavformat
*/
import "C"

import (
	"errors"
	"image"
	"fmt"
	"unsafe"
)

type H264Decoder struct {
	decoderImpl C.h264decoder_t
}

// Initialize decoder
func (decoder *H264Decoder) Init() error {
	decoder.decoderImpl.codec = C.avcodec_find_decoder(C.AV_CODEC_ID_H264)
	if decoder.decoderImpl.codec == nil {
		return errors.New("Can't find codec for decoder")
	}
	decoder.decoderImpl.context = C.avcodec_alloc_context3(decoder.decoderImpl.codec)
	if decoder.decoderImpl.context == nil {
		return errors.New("Can't allocate context for decoder")
	}
	decoder.decoderImpl.frame = C.av_frame_alloc()
	if decoder.decoderImpl.frame == nil {
		return errors.New("Can't allocate frame for decoder")
	}
	decoder.decoderImpl.packet = C.av_packet_alloc()
	if decoder.decoderImpl.packet == nil {
		return errors.New("Can't allocate packet for decoder")
	}
	ret := C.avcodec_open2(decoder.decoderImpl.context, decoder.decoderImpl.codec, nil)
	if ret < 0 {
		return errors.New(fmt.Sprintf("Open codec for decoder failed with code %d", ret))
	}
	return nil
}

// Decode NAL and return frame as image (if possible)
func (decoder *H264Decoder) Decode(nal []byte) (frame image.Image, err error) {
	decoder.decoderImpl.packet.data = (*C.uint8_t)(unsafe.Pointer(&nal[0]))
	decoder.decoderImpl.packet.size = C.int(len(nal))
	ret := C.avcodec_receive_frame(decoder.decoderImpl.context, decoder.decoderImpl.frame)
	hasPicture := (ret == 0)
	if ret == 0 || ret == -C.EAGAIN {
		ret = C.avcodec_send_packet(decoder.decoderImpl.context, decoder.decoderImpl.packet)
		if ret < 0 {
			err = errors.New(fmt.Sprintf("Decode (send_packet) failed with code %d", ret))
			return
		}
	} else {
		err = errors.New(fmt.Sprintf("Decode (receive_frame) failed with code %d", ret))
		return
	}
	if !hasPicture {
		err = NoPictureError{}
		return
	}
	
	frameWidth := int(decoder.decoderImpl.frame.width)
	frameHeight := int(decoder.decoderImpl.frame.height)
	yStride := int(decoder.decoderImpl.frame.linesize[0])
	cStride := int(decoder.decoderImpl.frame.linesize[1])

	frame = &image.YCbCr{
		Y: CPtr2UIntSlice(unsafe.Pointer(decoder.decoderImpl.frame.data[0]), yStride*frameHeight),
		Cb: CPtr2UIntSlice(unsafe.Pointer(decoder.decoderImpl.frame.data[1]), cStride*frameHeight/2),
		Cr: CPtr2UIntSlice(unsafe.Pointer(decoder.decoderImpl.frame.data[2]), cStride*frameHeight/2),
		YStride: yStride,
		CStride: cStride,
		SubsampleRatio: image.YCbCrSubsampleRatio420,
		Rect: image.Rect(0, 0, frameWidth, frameHeight),
	}

	return
}

// Deallocate resources
func (decoder *H264Decoder) Destroy() error {
	C.avcodec_free_context(&decoder.decoderImpl.context)
	C.av_frame_free(&decoder.decoderImpl.frame)
	C.av_packet_free(&decoder.decoderImpl.packet)
	return nil
}
