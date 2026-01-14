package vpx

import (
	"testing"
	"unsafe"
)

// TestVP8EncodeBasic demonstrates basic VP8 encoding workflow.
// This test shows how to:
// 1. Initialize encoder with VP8 codec
// 2. Configure encoding parameters
// 3. Encode YUV frames to VP8 compressed data
// 4. Retrieve encoded packets
func TestVP8EncodeBasic(t *testing.T) {
	const (
		width   = 320
		height  = 240
		bitrate = 200
	)

	// Step 1: Create codec context
	ctx := NewCodecCtx()
	if ctx == nil {
		t.Fatal("failed to create codec context")
	}
	defer CodecDestroy(ctx)

	// Step 2: Get VP8 encoder interface
	iface := EncoderIfaceVP8()
	if iface == nil {
		t.Fatal("failed to get VP8 encoder interface")
	}

	// Step 3: Get default encoder configuration
	cfg := &CodecEncCfg{}
	if err := Error(CodecEncConfigDefault(iface, cfg, 0)); err != nil {
		t.Fatalf("failed to get default encoder config: %v", err)
	}
	cfg.Deref()

	// Step 4: Configure encoder parameters
	cfg.GW = width
	cfg.GH = height
	cfg.GTimebase = Rational{Num: 1, Den: 30} // 30 fps
	cfg.RcTargetBitrate = bitrate
	cfg.GPass = RcOnePass
	cfg.RcEndUsage = Vbr
	cfg.KfMode = KfAuto
	cfg.KfMaxDist = 30
	cfg.GThreads = 1

	// Step 5: Initialize encoder
	if err := Error(CodecEncInitVer(ctx, iface, cfg, 0, EncoderABIVersion)); err != nil {
		t.Fatalf("failed to initialize encoder: %v", err)
	}

	// Step 6: Create test image
	img := ImageAlloc(nil, ImageFormatI420, width, height, 1)
	if img == nil {
		t.Fatal("failed to allocate image")
	}
	defer ImageFree(img)
	img.Deref()

	// Fill with test pattern
	fillTestPattern(img, 0)

	// Step 7: Encode frame
	if err := Error(CodecEncode(ctx, img, 0, 1, 0, DlGoodQuality)); err != nil {
		t.Fatalf("failed to encode frame: %v", err)
	}

	// Step 8: Get encoded data
	var iter CodecIter
	pkt := CodecGetCxData(ctx, &iter)
	if pkt == nil {
		t.Fatal("no encoded packet returned")
	}
	pkt.Deref()

	if pkt.Kind != CodecCxFramePkt {
		t.Fatalf("unexpected packet kind: %v", pkt.Kind)
	}

	data := pkt.GetFrameData()
	if len(data) == 0 {
		t.Fatal("encoded data is empty")
	}

	t.Logf("encoded frame size: %d bytes", len(data))
	t.Logf("is keyframe: %v", pkt.IsKeyframe())
}

// TestVP8EncodeMultipleFrames demonstrates encoding multiple frames.
func TestVP8EncodeMultipleFrames(t *testing.T) {
	const (
		width      = 320
		height     = 240
		frameCount = 10
	)

	ctx := NewCodecCtx()
	defer CodecDestroy(ctx)

	iface := EncoderIfaceVP8()
	cfg := &CodecEncCfg{}
	CodecEncConfigDefault(iface, cfg, 0)
	cfg.Deref()

	cfg.GW = width
	cfg.GH = height
	cfg.GTimebase = Rational{Num: 1, Den: 30}
	cfg.RcTargetBitrate = 200
	cfg.GPass = RcOnePass

	if err := Error(CodecEncInitVer(ctx, iface, cfg, 0, EncoderABIVersion)); err != nil {
		t.Fatalf("failed to initialize encoder: %v", err)
	}

	img := ImageAlloc(nil, ImageFormatI420, width, height, 1)
	defer ImageFree(img)
	img.Deref()

	var totalBytes int
	var keyframes int

	for i := 0; i < frameCount; i++ {
		fillTestPattern(img, i)

		if err := Error(CodecEncode(ctx, img, CodecPts(i), 1, 0, DlGoodQuality)); err != nil {
			t.Fatalf("failed to encode frame %d: %v", i, err)
		}

		var iter CodecIter
		for pkt := CodecGetCxData(ctx, &iter); pkt != nil; pkt = CodecGetCxData(ctx, &iter) {
			pkt.Deref()
			if pkt.Kind == CodecCxFramePkt {
				data := pkt.GetFrameData()
				totalBytes += len(data)
				if pkt.IsKeyframe() {
					keyframes++
				}
			}
		}
	}

	t.Logf("encoded %d frames, total %d bytes, %d keyframes", frameCount, totalBytes, keyframes)

	if keyframes == 0 {
		t.Error("expected at least one keyframe")
	}
}

// TestVP8EncodeForceKeyframe demonstrates forcing a keyframe.
func TestVP8EncodeForceKeyframe(t *testing.T) {
	const (
		width  = 320
		height = 240
	)

	ctx := NewCodecCtx()
	defer CodecDestroy(ctx)

	iface := EncoderIfaceVP8()
	cfg := &CodecEncCfg{}
	CodecEncConfigDefault(iface, cfg, 0)
	cfg.Deref()

	cfg.GW = width
	cfg.GH = height
	cfg.GTimebase = Rational{Num: 1, Den: 30}
	cfg.RcTargetBitrate = 200
	cfg.GPass = RcOnePass
	cfg.KfMode = KfDisabled // Disable auto keyframes

	if err := Error(CodecEncInitVer(ctx, iface, cfg, 0, EncoderABIVersion)); err != nil {
		t.Fatalf("failed to initialize encoder: %v", err)
	}

	img := ImageAlloc(nil, ImageFormatI420, width, height, 1)
	defer ImageFree(img)
	img.Deref()

	// Encode first frame (always keyframe)
	fillTestPattern(img, 0)
	CodecEncode(ctx, img, 0, 1, 0, DlGoodQuality)

	// Encode second frame without force keyframe
	fillTestPattern(img, 1)
	CodecEncode(ctx, img, 1, 1, 0, DlGoodQuality)

	// Encode third frame with force keyframe flag
	fillTestPattern(img, 2)
	if err := Error(CodecEncode(ctx, img, 2, 1, EflagForceKf, DlGoodQuality)); err != nil {
		t.Fatalf("failed to encode with force keyframe: %v", err)
	}

	var iter CodecIter
	for pkt := CodecGetCxData(ctx, &iter); pkt != nil; pkt = CodecGetCxData(ctx, &iter) {
		pkt.Deref()
		if pkt.Kind == CodecCxFramePkt {
			t.Logf("frame pts=%d, keyframe=%v", pkt.GetFramePts(), pkt.IsKeyframe())
		}
	}
}

// TestVP8EncodeFlush demonstrates flushing remaining frames from encoder.
func TestVP8EncodeFlush(t *testing.T) {
	const (
		width      = 320
		height     = 240
		frameCount = 5
	)

	ctx := NewCodecCtx()
	defer CodecDestroy(ctx)

	iface := EncoderIfaceVP8()
	cfg := &CodecEncCfg{}
	CodecEncConfigDefault(iface, cfg, 0)
	cfg.Deref()

	cfg.GW = width
	cfg.GH = height
	cfg.GTimebase = Rational{Num: 1, Den: 30}
	cfg.RcTargetBitrate = 200
	cfg.GPass = RcOnePass
	cfg.GLagInFrames = 5 // Enable frame lagging

	if err := Error(CodecEncInitVer(ctx, iface, cfg, 0, EncoderABIVersion)); err != nil {
		t.Fatalf("failed to initialize encoder: %v", err)
	}

	img := ImageAlloc(nil, ImageFormatI420, width, height, 1)
	defer ImageFree(img)
	img.Deref()

	var packetsBeforeFlush int
	for i := 0; i < frameCount; i++ {
		fillTestPattern(img, i)
		CodecEncode(ctx, img, CodecPts(i), 1, 0, DlGoodQuality)

		var iter CodecIter
		for pkt := CodecGetCxData(ctx, &iter); pkt != nil; pkt = CodecGetCxData(ctx, &iter) {
			pkt.Deref()
			if pkt.Kind == CodecCxFramePkt {
				packetsBeforeFlush++
			}
		}
	}

	// Flush encoder by passing nil image
	if err := Error(CodecEncode(ctx, nil, 0, 0, 0, DlGoodQuality)); err != nil {
		t.Fatalf("failed to flush encoder: %v", err)
	}

	var packetsAfterFlush int
	var iter CodecIter
	for pkt := CodecGetCxData(ctx, &iter); pkt != nil; pkt = CodecGetCxData(ctx, &iter) {
		pkt.Deref()
		if pkt.Kind == CodecCxFramePkt {
			packetsAfterFlush++
		}
	}

	t.Logf("packets before flush: %d, after flush: %d", packetsBeforeFlush, packetsAfterFlush)
}

// fillTestPattern fills image with a simple test pattern.
func fillTestPattern(img *Image, frameNum int) {
	h := int(img.DH)
	w := int(img.DW)
	yStride := int(img.Stride[PlaneY])
	uStride := int(img.Stride[PlaneU])
	vStride := int(img.Stride[PlaneV])

	yPlane := (*(*[1 << 30]byte)(unsafe.Pointer(img.Planes[PlaneY])))[:yStride*h]
	uPlane := (*(*[1 << 30]byte)(unsafe.Pointer(img.Planes[PlaneU])))[:uStride*h/2]
	vPlane := (*(*[1 << 30]byte)(unsafe.Pointer(img.Planes[PlaneV])))[:vStride*h/2]

	offset := (frameNum * 8) % 256
	for row := 0; row < h; row++ {
		for col := 0; col < w; col++ {
			yPlane[row*yStride+col] = byte((row + col + offset) % 256)
		}
	}

	for row := 0; row < h/2; row++ {
		for col := 0; col < w/2; col++ {
			uPlane[row*uStride+col] = byte((128 + row + offset/2) % 256)
			vPlane[row*vStride+col] = byte((128 + col + offset/2) % 256)
		}
	}
}
