package vpx

import (
	"testing"
)

// TestVP9EncodeBasic demonstrates basic VP9 encoding workflow.
// Note: VP9 encoder may buffer frames internally, requiring flush to get output.
func TestVP9EncodeBasic(t *testing.T) {
	const (
		width   = 320
		height  = 240
		bitrate = 200
	)

	ctx := NewCodecCtx()
	if ctx == nil {
		t.Fatal("failed to create codec context")
	}
	defer CodecDestroy(ctx)

	iface := EncoderIfaceVP9()
	if iface == nil {
		t.Fatal("failed to get VP9 encoder interface")
	}

	cfg := &CodecEncCfg{}
	if err := Error(CodecEncConfigDefault(iface, cfg, 0)); err != nil {
		t.Fatalf("failed to get default encoder config: %v", err)
	}
	cfg.Deref()

	cfg.GW = width
	cfg.GH = height
	cfg.GTimebase = Rational{Num: 1, Den: 30}
	cfg.RcTargetBitrate = bitrate
	cfg.GPass = RcOnePass
	cfg.RcEndUsage = Vbr
	cfg.GLagInFrames = 0

	if err := Error(CodecEncInitVer(ctx, iface, cfg, 0, EncoderABIVersion)); err != nil {
		t.Fatalf("failed to initialize encoder: %v", err)
	}

	img := ImageAlloc(nil, ImageFormatI420, width, height, 1)
	if img == nil {
		t.Fatal("failed to allocate image")
	}
	defer ImageFree(img)
	img.Deref()

	fillTestPattern(img, 0)

	if err := Error(CodecEncode(ctx, img, 0, 1, 0, DlGoodQuality)); err != nil {
		t.Fatalf("failed to encode frame: %v", err)
	}

	// VP9 requires flush to get buffered frames
	if err := Error(CodecEncode(ctx, nil, 0, 0, 0, DlGoodQuality)); err != nil {
		t.Fatalf("failed to flush encoder: %v", err)
	}

	var iter CodecIter
	pkt := CodecGetCxData(ctx, &iter)
	if pkt == nil {
		t.Fatal("no encoded packet returned after flush")
	}
	pkt.Deref()

	data := pkt.GetFrameData()
	if len(data) == 0 {
		t.Fatal("encoded data is empty")
	}

	t.Logf("VP9 encoded frame size: %d bytes, keyframe: %v", len(data), pkt.IsKeyframe())
}

// TestVP9EncodeMultipleFrames demonstrates VP9 encoding of multiple frames.
func TestVP9EncodeMultipleFrames(t *testing.T) {
	const (
		width      = 320
		height     = 240
		frameCount = 10
	)

	ctx := NewCodecCtx()
	defer CodecDestroy(ctx)

	iface := EncoderIfaceVP9()
	cfg := &CodecEncCfg{}
	CodecEncConfigDefault(iface, cfg, 0)
	cfg.Deref()

	cfg.GW = width
	cfg.GH = height
	cfg.GTimebase = Rational{Num: 1, Den: 30}
	cfg.RcTargetBitrate = 200
	cfg.GPass = RcOnePass
	cfg.GLagInFrames = 0

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

	// Flush remaining frames
	CodecEncode(ctx, nil, 0, 0, 0, DlGoodQuality)
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

	t.Logf("VP9: encoded %d frames, total %d bytes, %d keyframes", frameCount, totalBytes, keyframes)

	if keyframes == 0 {
		t.Error("expected at least one keyframe")
	}
}

// TestVP9EncodeHighQuality demonstrates VP9 encoding with best quality settings.
func TestVP9EncodeHighQuality(t *testing.T) {
	const (
		width  = 320
		height = 240
	)

	ctx := NewCodecCtx()
	defer CodecDestroy(ctx)

	iface := EncoderIfaceVP9()
	cfg := &CodecEncCfg{}
	CodecEncConfigDefault(iface, cfg, 0)
	cfg.Deref()

	cfg.GW = width
	cfg.GH = height
	cfg.GTimebase = Rational{Num: 1, Den: 30}
	cfg.RcTargetBitrate = 500
	cfg.GPass = RcOnePass
	cfg.RcEndUsage = Q           // Constant quality mode
	cfg.RcMinQuantizer = 0
	cfg.RcMaxQuantizer = 30      // Lower = higher quality
	cfg.GLagInFrames = 0

	if err := Error(CodecEncInitVer(ctx, iface, cfg, 0, EncoderABIVersion)); err != nil {
		t.Fatalf("failed to initialize encoder: %v", err)
	}

	img := ImageAlloc(nil, ImageFormatI420, width, height, 1)
	defer ImageFree(img)
	img.Deref()

	fillTestPattern(img, 0)

	// Use best quality deadline
	if err := Error(CodecEncode(ctx, img, 0, 1, 0, DlBestQuality)); err != nil {
		t.Fatalf("failed to encode: %v", err)
	}

	// Flush
	CodecEncode(ctx, nil, 0, 0, 0, DlBestQuality)

	var iter CodecIter
	pkt := CodecGetCxData(ctx, &iter)
	if pkt == nil {
		t.Fatal("no encoded packet")
	}
	pkt.Deref()

	data := pkt.GetFrameData()
	t.Logf("VP9 high quality frame size: %d bytes", len(data))
}

// TestVP9EncodeRealtime demonstrates VP9 encoding optimized for real-time.
func TestVP9EncodeRealtime(t *testing.T) {
	const (
		width  = 320
		height = 240
	)

	ctx := NewCodecCtx()
	defer CodecDestroy(ctx)

	iface := EncoderIfaceVP9()
	cfg := &CodecEncCfg{}
	CodecEncConfigDefault(iface, cfg, 0)
	cfg.Deref()

	cfg.GW = width
	cfg.GH = height
	cfg.GTimebase = Rational{Num: 1, Den: 30}
	cfg.RcTargetBitrate = 200
	cfg.GPass = RcOnePass
	cfg.RcEndUsage = Cbr         // Constant bitrate for real-time
	cfg.GLagInFrames = 0         // No frame lag for real-time
	cfg.GErrorResilient = ErrorResilientDefault

	if err := Error(CodecEncInitVer(ctx, iface, cfg, 0, EncoderABIVersion)); err != nil {
		t.Fatalf("failed to initialize encoder: %v", err)
	}

	img := ImageAlloc(nil, ImageFormatI420, width, height, 1)
	defer ImageFree(img)
	img.Deref()

	fillTestPattern(img, 0)

	// Use realtime deadline
	if err := Error(CodecEncode(ctx, img, 0, 1, 0, DlRealtime)); err != nil {
		t.Fatalf("failed to encode: %v", err)
	}

	// Flush
	CodecEncode(ctx, nil, 0, 0, 0, DlRealtime)

	var iter CodecIter
	pkt := CodecGetCxData(ctx, &iter)
	if pkt == nil {
		t.Fatal("no encoded packet")
	}
	pkt.Deref()

	data := pkt.GetFrameData()
	t.Logf("VP9 realtime frame size: %d bytes", len(data))
}

// TestVP9CompareWithVP8 compares VP9 and VP8 encoding efficiency.
func TestVP9CompareWithVP8(t *testing.T) {
	const (
		width      = 320
		height     = 240
		frameCount = 10
		bitrate    = 200
	)

	img := ImageAlloc(nil, ImageFormatI420, width, height, 1)
	defer ImageFree(img)
	img.Deref()

	// Encode with VP8
	vp8Ctx := NewCodecCtx()
	defer CodecDestroy(vp8Ctx)

	vp8Iface := EncoderIfaceVP8()
	vp8Cfg := &CodecEncCfg{}
	CodecEncConfigDefault(vp8Iface, vp8Cfg, 0)
	vp8Cfg.Deref()
	vp8Cfg.GW = width
	vp8Cfg.GH = height
	vp8Cfg.GTimebase = Rational{Num: 1, Den: 30}
	vp8Cfg.RcTargetBitrate = bitrate
	vp8Cfg.GPass = RcOnePass
	CodecEncInitVer(vp8Ctx, vp8Iface, vp8Cfg, 0, EncoderABIVersion)

	var vp8Bytes int
	for i := 0; i < frameCount; i++ {
		fillTestPattern(img, i)
		CodecEncode(vp8Ctx, img, CodecPts(i), 1, 0, DlGoodQuality)
		var iter CodecIter
		for pkt := CodecGetCxData(vp8Ctx, &iter); pkt != nil; pkt = CodecGetCxData(vp8Ctx, &iter) {
			pkt.Deref()
			if pkt.Kind == CodecCxFramePkt {
				vp8Bytes += len(pkt.GetFrameData())
			}
		}
	}

	// Encode with VP9
	vp9Ctx := NewCodecCtx()
	defer CodecDestroy(vp9Ctx)

	vp9Iface := EncoderIfaceVP9()
	vp9Cfg := &CodecEncCfg{}
	CodecEncConfigDefault(vp9Iface, vp9Cfg, 0)
	vp9Cfg.Deref()
	vp9Cfg.GW = width
	vp9Cfg.GH = height
	vp9Cfg.GTimebase = Rational{Num: 1, Den: 30}
	vp9Cfg.RcTargetBitrate = bitrate
	vp9Cfg.GPass = RcOnePass
	vp9Cfg.GLagInFrames = 0
	CodecEncInitVer(vp9Ctx, vp9Iface, vp9Cfg, 0, EncoderABIVersion)

	var vp9Bytes int
	for i := 0; i < frameCount; i++ {
		fillTestPattern(img, i)
		CodecEncode(vp9Ctx, img, CodecPts(i), 1, 0, DlGoodQuality)
		var iter CodecIter
		for pkt := CodecGetCxData(vp9Ctx, &iter); pkt != nil; pkt = CodecGetCxData(vp9Ctx, &iter) {
			pkt.Deref()
			if pkt.Kind == CodecCxFramePkt {
				vp9Bytes += len(pkt.GetFrameData())
			}
		}
	}

	// Flush VP9
	CodecEncode(vp9Ctx, nil, 0, 0, 0, DlGoodQuality)
	var iter CodecIter
	for pkt := CodecGetCxData(vp9Ctx, &iter); pkt != nil; pkt = CodecGetCxData(vp9Ctx, &iter) {
		pkt.Deref()
		if pkt.Kind == CodecCxFramePkt {
			vp9Bytes += len(pkt.GetFrameData())
		}
	}

	t.Logf("VP8: %d bytes, VP9: %d bytes for %d frames", vp8Bytes, vp9Bytes, frameCount)
	if vp8Bytes > 0 {
		t.Logf("VP9 compression ratio vs VP8: %.2f%%", float64(vp9Bytes)/float64(vp8Bytes)*100)
	}
}
