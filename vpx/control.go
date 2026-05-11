package vpx

/*
#cgo CFLAGS: -I${SRCDIR}/../include
#cgo LDFLAGS: -L${SRCDIR}/../lib -lvpx
#include <vpx/vpx_codec.h>
#include <vpx/vp8cx.h>

static vpx_codec_err_t vpx_codec_control_int(vpx_codec_ctx_t *ctx, int ctrl_id, int value) {
	return vpx_codec_control_(ctx, ctrl_id, value);
}

static vpx_codec_err_t vpx_codec_control_uint(vpx_codec_ctx_t *ctx, int ctrl_id, unsigned int value) {
	return vpx_codec_control_(ctx, ctrl_id, value);
}
*/
import "C"
import "unsafe"

const (
	VP8ESetCPUUsed            = int(C.VP8E_SET_CPUUSED)
	VP8ESetStaticThreshold    = int(C.VP8E_SET_STATIC_THRESHOLD)
	VP8ESetMaxIntraBitratePct = int(C.VP8E_SET_MAX_INTRA_BITRATE_PCT)

	VP9ESetTileColumns             = int(C.VP9E_SET_TILE_COLUMNS)
	VP9ESetTileRows                = int(C.VP9E_SET_TILE_ROWS)
	VP9ESetFrameParallelDecoding   = int(C.VP9E_SET_FRAME_PARALLEL_DECODING)
	VP9ESetRowMT                   = int(C.VP9E_SET_ROW_MT)
	VP9ESetMaxInterBitratePct      = int(C.VP9E_SET_MAX_INTER_BITRATE_PCT)
	VP9ESetGFCBRBoostPct           = int(C.VP9E_SET_GF_CBR_BOOST_PCT)
	VP9ESetDisableOvershootMaxQCBR = int(C.VP9E_SET_DISABLE_OVERSHOOT_MAXQ_CBR)
	VP9ESetDisableLoopFilter       = int(C.VP9E_SET_DISABLE_LOOPFILTER)
)

// CodecControlInt applies an int-valued encoder control to an initialized codec context.
//
// VP9ESetTileColumns uses libvpx's log2 tile column value:
// 0 means 1 column, 1 means 2 columns, 2 means 4 columns.
func CodecControlInt(ctx *CodecCtx, ctrlID int, value int) CodecErr {
	cctx, _ := (*C.vpx_codec_ctx_t)(unsafe.Pointer(ctx)), cgoAllocsUnknown
	cctrlID, _ := (C.int)(ctrlID), cgoAllocsUnknown
	cvalue, _ := (C.int)(value), cgoAllocsUnknown
	__ret := C.vpx_codec_control_int(cctx, cctrlID, cvalue)
	__v := (CodecErr)(__ret)
	return __v
}

// CodecControlUint applies an unsigned int-valued encoder control to an initialized codec context.
func CodecControlUint(ctx *CodecCtx, ctrlID int, value uint32) CodecErr {
	cctx, _ := (*C.vpx_codec_ctx_t)(unsafe.Pointer(ctx)), cgoAllocsUnknown
	cctrlID, _ := (C.int)(ctrlID), cgoAllocsUnknown
	cvalue, _ := (C.uint)(value), cgoAllocsUnknown
	__ret := C.vpx_codec_control_uint(cctx, cctrlID, cvalue)
	__v := (CodecErr)(__ret)
	return __v
}
