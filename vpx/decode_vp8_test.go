package vpx

import (
	"testing"
	"unsafe"
)

// TestVP8DecodeBasic demonstrates basic VP8 decoding workflow.
// This test shows how to:
// 1. Encode a frame to get VP8 data
// 2. Initialize decoder
// 3. Decode VP8 data back to YUV frame
func TestVP8DecodeBasic(t *testing.T) {
	const (
		width  = 320
		height = 240
	)

	// First, encode a frame to get VP8 data
	encodedData := encodeTestFrame(t, width, height)
	if len(encodedData) == 0 {
		t.Fatal("no encoded data")
	}

	// Step 1: Create decoder context
	ctx := NewCodecCtx()
	if ctx == nil {
		t.Fatal("failed to create codec context")
	}
	defer CodecDestroy(ctx)

	// Step 2: Get VP8 decoder interface
	iface := DecoderIfaceVP8()
	if iface == nil {
		t.Fatal("failed to get VP8 decoder interface")
	}

	// Step 3: Initialize decoder
	if err := Error(CodecDecInitVer(ctx, iface, nil, 0, DecoderABIVersion)); err != nil {
		t.Fatalf("failed to initialize decoder: %v", err)
	}

	// Step 4: Decode VP8 data
	if err := Error(CodecDecode(ctx, string(encodedData), uint32(len(encodedData)), nil, 0)); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	// Step 5: Get decoded frame
	var iter CodecIter
	img := CodecGetFrame(ctx, &iter)
	if img == nil {
		t.Fatal("no decoded frame returned")
	}
	img.Deref()

	if img.DW != width || img.DH != height {
		t.Errorf("decoded frame size mismatch: got %dx%d, want %dx%d", img.DW, img.DH, width, height)
	}

	t.Logf("decoded frame: %dx%d, format=%v", img.DW, img.DH, img.Fmt)
}

// TestVP8DecodeMultipleFrames demonstrates decoding multiple frames.
func TestVP8DecodeMultipleFrames(t *testing.T) {
	const (
		width      = 320
		height     = 240
		frameCount = 5
	)

	// Encode multiple frames
	packets := encodeTestFrames(t, width, height, frameCount)

	// Initialize decoder
	ctx := NewCodecCtx()
	defer CodecDestroy(ctx)

	iface := DecoderIfaceVP8()
	if err := Error(CodecDecInitVer(ctx, iface, nil, 0, DecoderABIVersion)); err != nil {
		t.Fatalf("failed to initialize decoder: %v", err)
	}

	// Decode each packet
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

	t.Logf("decoded %d frames", decodedCount)
}

// TestVP8DecodeWithConfig demonstrates decoding with custom configuration.
func TestVP8DecodeWithConfig(t *testing.T) {
	const (
		width  = 320
		height = 240
	)

	encodedData := encodeTestFrame(t, width, height)

	ctx := NewCodecCtx()
	defer CodecDestroy(ctx)

	iface := DecoderIfaceVP8()

	// Create decoder configuration
	cfg := &CodecDecCfg{
		Threads: 2,
		W:       width,
		H:       height,
	}

	if err := Error(CodecDecInitVer(ctx, iface, cfg, 0, DecoderABIVersion)); err != nil {
		t.Fatalf("failed to initialize decoder with config: %v", err)
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

	t.Logf("decoded with multi-threaded config: %dx%d", img.DW, img.DH)
}

// TestVP8DecodeExtractYUVData demonstrates extracting YUV data from decoded frame.
func TestVP8DecodeExtractYUVData(t *testing.T) {
	const (
		width  = 320
		height = 240
	)

	encodedData := encodeTestFrame(t, width, height)

	ctx := NewCodecCtx()
	defer CodecDestroy(ctx)

	iface := DecoderIfaceVP8()
	CodecDecInitVer(ctx, iface, nil, 0, DecoderABIVersion)
	CodecDecode(ctx, string(encodedData), uint32(len(encodedData)), nil, 0)

	var iter CodecIter
	img := CodecGetFrame(ctx, &iter)
	if img == nil {
		t.Fatal("no decoded frame")
	}
	img.Deref()

	// Extract YUV planes
	y, u, v := img.GetYUVData()
	if y == nil || u == nil || v == nil {
		t.Fatal("failed to get YUV data")
	}

	expectedYSize := int(img.Stride[PlaneY]) * int(img.DH)
	expectedUVSize := int(img.Stride[PlaneU]) * int(img.DH) / 2

	if len(y) != expectedYSize {
		t.Errorf("Y plane size mismatch: got %d, want %d", len(y), expectedYSize)
	}
	if len(u) != expectedUVSize {
		t.Errorf("U plane size mismatch: got %d, want %d", len(u), expectedUVSize)
	}
	if len(v) != expectedUVSize {
		t.Errorf("V plane size mismatch: got %d, want %d", len(v), expectedUVSize)
	}

	t.Logf("extracted YUV data: Y=%d bytes, U=%d bytes, V=%d bytes", len(y), len(u), len(v))
}

// TestVP8DecodeStreamInfo demonstrates getting stream info before full decode.
func TestVP8DecodeStreamInfo(t *testing.T) {
	const (
		width  = 320
		height = 240
	)

	encodedData := encodeTestFrame(t, width, height)

	iface := DecoderIfaceVP8()

	// Peek stream info without full decode
	si := &CodecStreamInfo{
		Sz: 32, // Size of vpx_codec_stream_info_t
	}

	if err := Error(CodecPeekStreamInfo(iface, string(encodedData), uint32(len(encodedData)), si)); err != nil {
		t.Fatalf("failed to peek stream info: %v", err)
	}
	si.Deref()

	t.Logf("stream info: %dx%d, is_kf=%v", si.W, si.H, si.IsKf != 0)

	if si.W != width || si.H != height {
		t.Errorf("stream info size mismatch: got %dx%d, want %dx%d", si.W, si.H, width, height)
	}
}

// encodeTestFrame encodes a single test frame and returns the encoded data.
func encodeTestFrame(t *testing.T, width, height uint32) []byte {
	t.Helper()

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

	fillTestPattern(img, 0)

	if err := Error(CodecEncode(ctx, img, 0, 1, 0, DlGoodQuality)); err != nil {
		t.Fatalf("failed to encode: %v", err)
	}

	var iter CodecIter
	pkt := CodecGetCxData(ctx, &iter)
	if pkt == nil {
		t.Fatal("no encoded packet")
	}
	pkt.Deref()

	return pkt.GetFrameData()
}

// encodeTestFrames encodes multiple test frames and returns the encoded packets.
func encodeTestFrames(t *testing.T, width, height uint32, count int) [][]byte {
	t.Helper()

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

	return packets
}

// copyDecodedFrame creates a copy of decoded frame data.
func copyDecodedFrame(img *Image) *Image {
	h := int(img.DH)
	ySize := int(img.Stride[PlaneY]) * h
	uvH := h / 2
	uSize := int(img.Stride[PlaneU]) * uvH
	vSize := int(img.Stride[PlaneV]) * uvH

	imgCopy := &Image{
		Fmt:          img.Fmt,
		W:            img.W,
		H:            img.H,
		DW:           img.DW,
		DH:           img.DH,
		XChromaShift: img.XChromaShift,
		YChromaShift: img.YChromaShift,
		Stride:       img.Stride,
	}

	imgCopy.ImgData = make([]byte, ySize+uSize+vSize)

	ySrc := (*(*[1 << 30]byte)(unsafe.Pointer(img.Planes[PlaneY])))[:ySize]
	uSrc := (*(*[1 << 30]byte)(unsafe.Pointer(img.Planes[PlaneU])))[:uSize]
	vSrc := (*(*[1 << 30]byte)(unsafe.Pointer(img.Planes[PlaneV])))[:vSize]

	copy(imgCopy.ImgData[0:ySize], ySrc)
	copy(imgCopy.ImgData[ySize:ySize+uSize], uSrc)
	copy(imgCopy.ImgData[ySize+uSize:], vSrc)

	imgCopy.Planes[PlaneY] = &imgCopy.ImgData[0]
	imgCopy.Planes[PlaneU] = &imgCopy.ImgData[ySize]
	imgCopy.Planes[PlaneV] = &imgCopy.ImgData[ySize+uSize]

	return imgCopy
}
