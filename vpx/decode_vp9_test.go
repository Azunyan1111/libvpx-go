package vpx

import (
	"testing"
)

// TestVP9DecodeBasic demonstrates basic VP9 decoding workflow.
func TestVP9DecodeBasic(t *testing.T) {
	const (
		width  = 320
		height = 240
	)

	// Encode a VP9 frame first
	encodedData := encodeVP9TestFrame(t, width, height)
	if len(encodedData) == 0 {
		t.Fatal("no encoded data")
	}

	// Create decoder context
	ctx := NewCodecCtx()
	if ctx == nil {
		t.Fatal("failed to create codec context")
	}
	defer CodecDestroy(ctx)

	// Get VP9 decoder interface
	iface := DecoderIfaceVP9()
	if iface == nil {
		t.Fatal("failed to get VP9 decoder interface")
	}

	// Initialize decoder
	if err := Error(CodecDecInitVer(ctx, iface, nil, 0, DecoderABIVersion)); err != nil {
		t.Fatalf("failed to initialize decoder: %v", err)
	}

	// Decode VP9 data
	if err := Error(CodecDecode(ctx, string(encodedData), uint32(len(encodedData)), nil, 0)); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	// Get decoded frame
	var iter CodecIter
	img := CodecGetFrame(ctx, &iter)
	if img == nil {
		t.Fatal("no decoded frame returned")
	}
	img.Deref()

	if img.DW != width || img.DH != height {
		t.Errorf("decoded frame size mismatch: got %dx%d, want %dx%d", img.DW, img.DH, width, height)
	}

	t.Logf("VP9 decoded frame: %dx%d, format=%v", img.DW, img.DH, img.Fmt)
}

// TestVP9DecodeMultipleFrames demonstrates decoding multiple VP9 frames.
func TestVP9DecodeMultipleFrames(t *testing.T) {
	const (
		width      = 320
		height     = 240
		frameCount = 5
	)

	packets := encodeVP9TestFrames(t, width, height, frameCount)

	ctx := NewCodecCtx()
	defer CodecDestroy(ctx)

	iface := DecoderIfaceVP9()
	if err := Error(CodecDecInitVer(ctx, iface, nil, 0, DecoderABIVersion)); err != nil {
		t.Fatalf("failed to initialize decoder: %v", err)
	}

	var decodedCount int
	for i, pkt := range packets {
		if err := Error(CodecDecode(ctx, string(pkt), uint32(len(pkt)), nil, 0)); err != nil {
			t.Fatalf("failed to decode packet %d: %v", i, err)
		}

		var iter CodecIter
		for img := CodecGetFrame(ctx, &iter); img != nil; img = CodecGetFrame(ctx, &iter) {
			img.Deref()
			decodedCount++
		}
	}

	if decodedCount != frameCount {
		t.Errorf("decoded %d frames, want %d", decodedCount, frameCount)
	}

	t.Logf("VP9: decoded %d frames", decodedCount)
}

// TestVP9DecodeWithFrameThreading demonstrates VP9 decoding with frame threading.
func TestVP9DecodeWithFrameThreading(t *testing.T) {
	const (
		width  = 320
		height = 240
	)

	encodedData := encodeVP9TestFrame(t, width, height)

	ctx := NewCodecCtx()
	defer CodecDestroy(ctx)

	iface := DecoderIfaceVP9()

	// Check if frame threading is supported
	caps := CodecGetCaps(iface)
	if caps&CodecCapFrameThreading == 0 {
		t.Skip("frame threading not supported")
	}

	cfg := &CodecDecCfg{
		Threads: 4,
		W:       width,
		H:       height,
	}

	// Initialize with frame threading flag
	if err := Error(CodecDecInitVer(ctx, iface, cfg, CodecUseFrameThreading, DecoderABIVersion)); err != nil {
		t.Fatalf("failed to initialize decoder with frame threading: %v", err)
	}

	if err := Error(CodecDecode(ctx, string(encodedData), uint32(len(encodedData)), nil, 0)); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	var iter CodecIter
	img := CodecGetFrame(ctx, &iter)
	if img == nil {
		t.Fatal("no decoded frame")
	}
	img.Deref()

	t.Logf("VP9 decoded with frame threading: %dx%d", img.DW, img.DH)
}

// TestVP9TranscodeToVP8 demonstrates transcoding VP9 to VP8.
func TestVP9TranscodeToVP8(t *testing.T) {
	const (
		width  = 320
		height = 240
	)

	// Encode as VP9
	vp9Data := encodeVP9TestFrame(t, width, height)
	t.Logf("VP9 encoded size: %d bytes", len(vp9Data))

	// Decode VP9
	decCtx := NewCodecCtx()
	defer CodecDestroy(decCtx)

	decIface := DecoderIfaceVP9()
	CodecDecInitVer(decCtx, decIface, nil, 0, DecoderABIVersion)
	CodecDecode(decCtx, string(vp9Data), uint32(len(vp9Data)), nil, 0)

	var decIter CodecIter
	decodedImg := CodecGetFrame(decCtx, &decIter)
	if decodedImg == nil {
		t.Fatal("failed to decode VP9 frame")
	}
	decodedImg.Deref()

	// Re-encode as VP8
	encCtx := NewCodecCtx()
	defer CodecDestroy(encCtx)

	encIface := EncoderIfaceVP8()
	cfg := &CodecEncCfg{}
	CodecEncConfigDefault(encIface, cfg, 0)
	cfg.Deref()
	cfg.GW = width
	cfg.GH = height
	cfg.GTimebase = Rational{Num: 1, Den: 30}
	cfg.RcTargetBitrate = 200
	cfg.GPass = RcOnePass
	CodecEncInitVer(encCtx, encIface, cfg, 0, EncoderABIVersion)

	// Create new image for encoding (decoder image data is owned by decoder)
	encImg := ImageAlloc(nil, ImageFormatI420, width, height, 1)
	defer ImageFree(encImg)
	encImg.Deref()

	// Copy decoded data to encoder image
	copyImageData(encImg, decodedImg)

	CodecEncode(encCtx, encImg, 0, 1, 0, DlGoodQuality)

	var encIter CodecIter
	pkt := CodecGetCxData(encCtx, &encIter)
	if pkt == nil {
		t.Fatal("failed to encode to VP8")
	}
	pkt.Deref()

	vp8Data := pkt.GetFrameData()
	t.Logf("VP8 encoded size: %d bytes", len(vp8Data))
	t.Logf("transcoding ratio: VP9 %d -> VP8 %d bytes", len(vp9Data), len(vp8Data))
}

// encodeVP9TestFrame encodes a single test frame using VP9.
func encodeVP9TestFrame(t *testing.T, width, height uint32) []byte {
	t.Helper()

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
		t.Fatalf("failed to initialize VP9 encoder: %v", err)
	}

	img := ImageAlloc(nil, ImageFormatI420, width, height, 1)
	defer ImageFree(img)
	img.Deref()

	fillTestPattern(img, 0)

	if err := Error(CodecEncode(ctx, img, 0, 1, 0, DlGoodQuality)); err != nil {
		t.Fatalf("failed to encode: %v", err)
	}

	// VP9 requires flush
	CodecEncode(ctx, nil, 0, 0, 0, DlGoodQuality)

	var iter CodecIter
	pkt := CodecGetCxData(ctx, &iter)
	if pkt == nil {
		t.Fatal("no encoded packet")
	}
	pkt.Deref()

	return pkt.GetFrameData()
}

// encodeVP9TestFrames encodes multiple test frames using VP9.
func encodeVP9TestFrames(t *testing.T, width, height uint32, count int) [][]byte {
	t.Helper()

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
		t.Fatalf("failed to initialize VP9 encoder: %v", err)
	}

	img := ImageAlloc(nil, ImageFormatI420, width, height, 1)
	defer ImageFree(img)
	img.Deref()

	var packets [][]byte
	for i := 0; i < count; i++ {
		fillTestPattern(img, i)
		CodecEncode(ctx, img, CodecPts(i), 1, 0, DlGoodQuality)

		var iter CodecIter
		for pkt := CodecGetCxData(ctx, &iter); pkt != nil; pkt = CodecGetCxData(ctx, &iter) {
			pkt.Deref()
			if pkt.Kind == CodecCxFramePkt {
				data := pkt.GetFrameData()
				cpy := make([]byte, len(data))
				copy(cpy, data)
				packets = append(packets, cpy)
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
			cpy := make([]byte, len(data))
			copy(cpy, data)
			packets = append(packets, cpy)
		}
	}

	return packets
}
