package irnc

/*
#include <libavcodec/avcodec.h>
#include <libavformat/avformat.h>
#include <libavutil/avutil.h>
#include <libavutil/avconfig.h>

typedef struct {
	AVCodec *codec;
	AVCodecContext *context;
	AVFrame *frame;
	AVPacket *packet;
} h264encoder_t;

*/
import "C"

import (
	"errors"
	"fmt"
	"image"
	"log"
	"unsafe"
)

type H264Encoder struct {
	encoderImpl C.h264encoder_t
	bitrate, framerate uint
}

// Initialize encoder based on sample image format
func (encoder *H264Encoder) InitBySample(img image.Image) (header []byte, err error) {
	width := C.int(img.Bounds().Dx())
	height := C.int(img.Bounds().Dy())
	var pix_fmt int32
	switch imgOfType := img.(type) {
		case *image.YCbCr:
			pix_fmt = C.AV_PIX_FMT_YUV420P
			encoder.encoderImpl.codec = C.avcodec_find_encoder(C.AV_CODEC_ID_H264)
		case *RGBImage:
			pix_fmt = C.AV_PIX_FMT_RGB24
			libName := C.CString("libx264rgb")
			defer C.free(unsafe.Pointer(libName))
			encoder.encoderImpl.codec = C.avcodec_find_encoder_by_name(libName)
		default:
			// low tolerance for unnoticed unimplemented cases
			log.Panicf("H264 encoder for image type %T is not implemented", imgOfType)
	}
	if encoder.encoderImpl.codec == nil {
		err = errors.New("Can't find codec for encoder")
		return
	}
	encoder.encoderImpl.context = C.avcodec_alloc_context3(encoder.encoderImpl.codec)
	if encoder.encoderImpl.context == nil {
		err = errors.New("Can't allocate context for encoder")
		return
	}
	encoder.encoderImpl.context.width = width
	encoder.encoderImpl.context.height = height
	encoder.encoderImpl.context.bit_rate = C.longlong(encoder.bitrate)
	encoder.encoderImpl.context.time_base = C.av_make_q(1, C.int(encoder.framerate))
	encoder.encoderImpl.context.pix_fmt = pix_fmt
	encoder.encoderImpl.context.flags |= C.AV_CODEC_FLAG_GLOBAL_HEADER
	encoder.encoderImpl.frame = C.av_frame_alloc()
	if encoder.encoderImpl.frame == nil {
		err = errors.New("Can't allocate frame for encoder")
		return
	}
	encoder.encoderImpl.frame.width = width
	encoder.encoderImpl.frame.height = height
	encoder.encoderImpl.frame.format = C.int(pix_fmt)
	encoder.encoderImpl.packet = C.av_packet_alloc()
	if encoder.encoderImpl.packet == nil {
		err = errors.New("Can't allocate packet for encoder")
		return
	}
	ret := C.avcodec_open2(encoder.encoderImpl.context, encoder.encoderImpl.codec, nil)
	if ret < 0 {
		err = errors.New(fmt.Sprintf("Open codec for encoder failed with code %d", ret))
		return
	}
	header = CPtr2UIntSlice(unsafe.Pointer(encoder.encoderImpl.context.extradata), int(encoder.encoderImpl.context.extradata_size))
	return
}

// Encode video frame and return NAL (if possible)
func (encoder *H264Encoder) Encode(img image.Image) (out []byte, err error) {
	var frame *C.AVFrame
	if img != nil {
		frame = encoder.encoderImpl.frame
		frame.pts = C.longlong(encoder.encoderImpl.context.frame_number)
		switch imgOfType := img.(type) {
			case *image.YCbCr:
				frame.data[0] = (*C.uint8_t)(unsafe.Pointer(&imgOfType.Y[0]))
				frame.data[1] = (*C.uint8_t)(unsafe.Pointer(&imgOfType.Cb[0]))
				frame.data[2] = (*C.uint8_t)(unsafe.Pointer(&imgOfType.Cr[0]))
				frame.linesize[0] = C.int(imgOfType.YStride)
				frame.linesize[1] = C.int(imgOfType.CStride)
				frame.linesize[2] = C.int(imgOfType.CStride)
			case *RGBImage:
				data, size := imgOfType.GetOriginalData()
				frame.data[0] = (*C.uint8_t)(unsafe.Pointer(&data[0]))
				frame.linesize[0] = C.int(size) * 3
			default:
				log.Panicf("H264 encoder for image type %T is not implemented", imgOfType)
		}
	}
	ret := C.avcodec_send_frame(encoder.encoderImpl.context, frame)
	if ret < 0 {
		err = errors.New(fmt.Sprintf("Encode (send_frame) failed with code %d", ret))
		return
	}
	ret = C.avcodec_receive_packet(encoder.encoderImpl.context, encoder.encoderImpl.packet)
	if ret == -C.EAGAIN {
		err = TryAgainError{}
		return
	}
	if ret == C.AVERROR_EOF {
		err = EOFError{}
		return
	}
	if ret < 0 {
		err = errors.New(fmt.Sprintf("Encode (receive_packet) failed with code %d", ret))
		return
	}
	out = make([]byte, encoder.encoderImpl.packet.size)
	C.memcpy(
		unsafe.Pointer(&out[0]),
		unsafe.Pointer(encoder.encoderImpl.packet.data),
		C.size_t(encoder.encoderImpl.packet.size),
	)
	return
}

// Deallocate resources
func (encoder *H264Encoder) Destroy() error {
	C.avcodec_free_context(&encoder.encoderImpl.context)
	C.av_frame_free(&encoder.encoderImpl.frame)
	C.av_packet_free(&encoder.encoderImpl.packet)
	return nil
}
