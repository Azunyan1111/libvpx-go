package vpx

/*
#cgo CFLAGS: -I${SRCDIR}/../include
#cgo LDFLAGS: -L${SRCDIR}/../lib -lvpx
#include <vpx/vpx_encoder.h>
#include <stdlib.h>
#include <string.h>

// Helper functions to access union fields in vpx_codec_cx_pkt_t
static void* get_cx_pkt_frame_buf(const vpx_codec_cx_pkt_t* pkt) {
	return pkt->data.frame.buf;
}

static size_t get_cx_pkt_frame_sz(const vpx_codec_cx_pkt_t* pkt) {
	return pkt->data.frame.sz;
}

static vpx_codec_pts_t get_cx_pkt_frame_pts(const vpx_codec_cx_pkt_t* pkt) {
	return pkt->data.frame.pts;
}

static unsigned long get_cx_pkt_frame_duration(const vpx_codec_cx_pkt_t* pkt) {
	return pkt->data.frame.duration;
}

static vpx_codec_frame_flags_t get_cx_pkt_frame_flags(const vpx_codec_cx_pkt_t* pkt) {
	return pkt->data.frame.flags;
}
*/
import "C"
import "unsafe"

// GetFrameData returns the compressed frame data from CodecCxPkt.
// Returns nil if the packet is nil or not a frame packet.
func (pkt *CodecCxPkt) GetFrameData() []byte {
	if pkt == nil || pkt.refa671fc83 == nil {
		return nil
	}
	buf := C.get_cx_pkt_frame_buf(pkt.refa671fc83)
	sz := C.get_cx_pkt_frame_sz(pkt.refa671fc83)
	if buf == nil || sz == 0 {
		return nil
	}
	return C.GoBytes(buf, C.int(sz))
}

// GetFramePts returns the presentation timestamp of the frame.
func (pkt *CodecCxPkt) GetFramePts() CodecPts {
	if pkt == nil || pkt.refa671fc83 == nil {
		return 0
	}
	return CodecPts(C.get_cx_pkt_frame_pts(pkt.refa671fc83))
}

// GetFrameDuration returns the duration of the frame.
func (pkt *CodecCxPkt) GetFrameDuration() uint {
	if pkt == nil || pkt.refa671fc83 == nil {
		return 0
	}
	return uint(C.get_cx_pkt_frame_duration(pkt.refa671fc83))
}

// GetFrameFlags returns the frame flags (e.g., keyframe flag).
func (pkt *CodecCxPkt) GetFrameFlags() CodecFrameFlags {
	if pkt == nil || pkt.refa671fc83 == nil {
		return 0
	}
	return CodecFrameFlags(C.get_cx_pkt_frame_flags(pkt.refa671fc83))
}

// IsKeyframe returns true if the frame is a keyframe.
func (pkt *CodecCxPkt) IsKeyframe() bool {
	return pkt.GetFrameFlags()&FrameIsKey != 0
}

// SetImageData sets YUV plane data to the Image structure.
// This is a helper function for setting raw YUV data.
func (img *Image) SetImageData(y, u, v []byte) {
	if img == nil {
		return
	}
	if len(y) > 0 {
		img.Planes[PlaneY] = &y[0]
	}
	if len(u) > 0 {
		img.Planes[PlaneU] = &u[0]
	}
	if len(v) > 0 {
		img.Planes[PlaneV] = &v[0]
	}
}

// GetYUVData extracts YUV plane data from the Image.
// Returns Y, U, V byte slices.
func (img *Image) GetYUVData() (y, u, v []byte) {
	if img == nil {
		return nil, nil, nil
	}

	h := int(img.DH)
	yStride := int(img.Stride[PlaneY])
	uStride := int(img.Stride[PlaneU])
	vStride := int(img.Stride[PlaneV])

	ySz := yStride * h
	uvH := h / 2
	uSz := uStride * uvH
	vSz := vStride * uvH

	if img.Planes[PlaneY] != nil {
		y = (*(*[1 << 30]byte)(unsafe.Pointer(img.Planes[PlaneY])))[:ySz:ySz]
	}
	if img.Planes[PlaneU] != nil {
		u = (*(*[1 << 30]byte)(unsafe.Pointer(img.Planes[PlaneU])))[:uSz:uSz]
	}
	if img.Planes[PlaneV] != nil {
		v = (*(*[1 << 30]byte)(unsafe.Pointer(img.Planes[PlaneV])))[:vSz:vSz]
	}

	return y, u, v
}
