package vpx

import (
	"testing"
	"unsafe"
)

// TestDecoderIntegrity verifies decoder integrity by checking various edge cases.
func TestDecoderIntegrity(t *testing.T) {
	t.Run("VP8", func(t *testing.T) {
		testDecoderIntegrityCodec(t, true)
	})
	t.Run("VP9", func(t *testing.T) {
		testDecoderIntegrityCodec(t, false)
	})
}

func testDecoderIntegrityCodec(t *testing.T, isVP8 bool) {
	codecName := "VP9"
	if isVP8 {
		codecName = "VP8"
	}

	// Test 1: Decode same data twice, should get identical results
	t.Run("DecodeConsistency", func(t *testing.T) {
		const (
			width  = 320
			height = 240
		)

		// Create and encode
		origImg := ImageAlloc(nil, ImageFormatI420, width, height, 1)
		defer ImageFree(origImg)
		origImg.Deref()
		fillTestPattern(origImg, 0)

		var encodedData []byte
		if isVP8 {
			encodedData = encodeVP8Frame(t, origImg)
		} else {
			encodedData = encodeVP9Frame(t, origImg)
		}

		// Decode twice with separate contexts
		decoded1 := decodeFrame(t, encodedData, isVP8)
		decoded2 := decodeFrame(t, encodedData, isVP8)

		y1 := extractYPlane(decoded1)
		y2 := extractYPlane(decoded2)

		if len(y1) != len(y2) {
			t.Fatalf("%s: decoded frame sizes differ: %d vs %d", codecName, len(y1), len(y2))
		}

		for i := range y1 {
			if y1[i] != y2[i] {
				t.Fatalf("%s: decoded frames differ at byte %d: %d vs %d", codecName, i, y1[i], y2[i])
			}
		}
		t.Logf("%s: decode consistency OK - identical results", codecName)
	})

	// Test 2: Verify frame dimensions after decode
	t.Run("FrameDimensions", func(t *testing.T) {
		const (
			width  = 320
			height = 240
		)

		encodedData := encodeFrameWithNewImage(t, width, height, isVP8)
		if len(encodedData) == 0 {
			t.Fatalf("%s: no encoded data", codecName)
		}

		decoded := decodeFrame(t, encodedData, isVP8)
		if decoded.DW != width || decoded.DH != height {
			t.Errorf("%s: resolution mismatch: got %dx%d, want %dx%d",
				codecName, decoded.DW, decoded.DH, width, height)
		}

		// Verify format
		if decoded.Fmt != ImageFormatI420 {
			t.Errorf("%s: unexpected format: %v", codecName, decoded.Fmt)
		}

		t.Logf("%s: frame dimensions OK - %dx%d, format=%v", codecName, decoded.DW, decoded.DH, decoded.Fmt)
	})

	// Test 3: Sequential frame decoding (P-frames depend on previous frames)
	t.Run("SequentialFrames", func(t *testing.T) {
		const (
			width      = 320
			height     = 240
			frameCount = 10
		)

		// Encode multiple frames
		packets := encodeMultipleFrames(t, width, height, frameCount, isVP8)

		// Decode all frames in sequence
		decCtx := NewCodecCtx()
		defer CodecDestroy(decCtx)

		var decIface *CodecIface
		if isVP8 {
			decIface = DecoderIfaceVP8()
		} else {
			decIface = DecoderIfaceVP9()
		}

		if err := Error(CodecDecInitVer(decCtx, decIface, nil, 0, DecoderABIVersion)); err != nil {
			t.Fatalf("failed to init decoder: %v", err)
		}

		var decodedCount int
		var prevY []byte

		for i, pkt := range packets {
			if err := Error(CodecDecode(decCtx, string(pkt), uint32(len(pkt)), nil, 0)); err != nil {
				t.Fatalf("%s: failed to decode packet %d: %v", codecName, i, err)
			}

			var iter CodecIter
			for img := CodecGetFrame(decCtx, &iter); img != nil; img = CodecGetFrame(decCtx, &iter) {
				img.Deref()

				if img.DW != width || img.DH != height {
					t.Errorf("%s frame %d: wrong size %dx%d", codecName, decodedCount, img.DW, img.DH)
				}

				currentY := extractYPlane(img)

				// Frames should be different (test pattern changes each frame)
				if prevY != nil && decodedCount > 0 {
					same := true
					for j := range currentY {
						if currentY[j] != prevY[j] {
							same = false
							break
						}
					}
					if same {
						t.Errorf("%s: frame %d identical to previous frame", codecName, decodedCount)
					}
				}

				prevY = make([]byte, len(currentY))
				copy(prevY, currentY)
				decodedCount++
			}
		}

		if decodedCount != frameCount {
			t.Errorf("%s: expected %d frames, got %d", codecName, frameCount, decodedCount)
		}
		t.Logf("%s: sequential decode OK - %d frames", codecName, decodedCount)
	})

	// Test 4: Verify decoder doesn't crash on reuse
	t.Run("DecoderReuse", func(t *testing.T) {
		const (
			width  = 320
			height = 240
		)

		decCtx := NewCodecCtx()
		defer CodecDestroy(decCtx)

		var decIface *CodecIface
		if isVP8 {
			decIface = DecoderIfaceVP8()
		} else {
			decIface = DecoderIfaceVP9()
		}

		CodecDecInitVer(decCtx, decIface, nil, 0, DecoderABIVersion)

		// Decode multiple different frames with same decoder
		for i := 0; i < 5; i++ {
			origImg := ImageAlloc(nil, ImageFormatI420, width, height, 1)
			origImg.Deref()
			fillTestPattern(origImg, i*10) // Different patterns

			var encodedData []byte
			if isVP8 {
				encodedData = encodeVP8Frame(t, origImg)
			} else {
				encodedData = encodeVP9Frame(t, origImg)
			}
			ImageFree(origImg)

			CodecDecode(decCtx, string(encodedData), uint32(len(encodedData)), nil, 0)

			var iter CodecIter
			img := CodecGetFrame(decCtx, &iter)
			if img == nil {
				t.Fatalf("%s: no frame on iteration %d", codecName, i)
			}
			img.Deref()

			if img.DW != width || img.DH != height {
				t.Errorf("%s iteration %d: wrong size", codecName, i)
			}
		}
		t.Logf("%s: decoder reuse OK", codecName)
	})

	// Test 5: Verify all YUV planes are valid
	t.Run("YUVPlaneValidity", func(t *testing.T) {
		const (
			width  = 320
			height = 240
		)

		origImg := ImageAlloc(nil, ImageFormatI420, width, height, 1)
		defer ImageFree(origImg)
		origImg.Deref()
		fillTestPattern(origImg, 0)

		var encodedData []byte
		if isVP8 {
			encodedData = encodeVP8Frame(t, origImg)
		} else {
			encodedData = encodeVP9Frame(t, origImg)
		}

		decoded := decodeFrame(t, encodedData, isVP8)

		// Check Y plane
		y, u, v := decoded.GetYUVData()
		if y == nil || u == nil || v == nil {
			t.Fatalf("%s: YUV planes are nil", codecName)
		}

		expectedYSize := int(decoded.Stride[PlaneY]) * int(decoded.DH)
		expectedUVSize := int(decoded.Stride[PlaneU]) * int(decoded.DH) / 2

		if len(y) != expectedYSize {
			t.Errorf("%s: Y plane size wrong: got %d, want %d", codecName, len(y), expectedYSize)
		}
		if len(u) != expectedUVSize {
			t.Errorf("%s: U plane size wrong: got %d, want %d", codecName, len(u), expectedUVSize)
		}
		if len(v) != expectedUVSize {
			t.Errorf("%s: V plane size wrong: got %d, want %d", codecName, len(v), expectedUVSize)
		}

		// Check that planes contain valid data (not all zeros or all same value)
		if allSameValue(y) {
			t.Errorf("%s: Y plane has all same values", codecName)
		}
		if allSameValue(u) {
			t.Errorf("%s: U plane has all same values", codecName)
		}
		if allSameValue(v) {
			t.Errorf("%s: V plane has all same values", codecName)
		}

		t.Logf("%s: YUV planes valid - Y:%d U:%d V:%d bytes", codecName, len(y), len(u), len(v))
	})
}

func encodeVP8Frame(t *testing.T, img *Image) []byte {
	t.Helper()
	return encodeVP8FrameWithSize(t, img, img.DW, img.DH)
}

func encodeVP8FrameWithSize(t *testing.T, img *Image, width, height uint32) []byte {
	t.Helper()

	encCtx := NewCodecCtx()
	defer CodecDestroy(encCtx)

	encIface := EncoderIfaceVP8()
	cfg := &CodecEncCfg{}
	CodecEncConfigDefault(encIface, cfg, 0)
	cfg.Deref()

	cfg.GW = width
	cfg.GH = height
	cfg.GTimebase = Rational{Num: 1, Den: 30}
	cfg.RcTargetBitrate = 500
	cfg.GPass = RcOnePass

	if err := Error(CodecEncInitVer(encCtx, encIface, cfg, 0, EncoderABIVersion)); err != nil {
		t.Logf("VP8 encode init failed for %dx%d: %v", width, height, err)
		return nil
	}

	if err := Error(CodecEncode(encCtx, img, 0, 1, 0, DlGoodQuality)); err != nil {
		t.Logf("VP8 encode failed: %v", err)
		return nil
	}

	var iter CodecIter
	pkt := CodecGetCxData(encCtx, &iter)
	if pkt == nil {
		t.Logf("VP8: no packet for %dx%d", width, height)
		return nil
	}
	pkt.Deref()

	data := make([]byte, len(pkt.GetFrameData()))
	copy(data, pkt.GetFrameData())
	return data
}

func encodeVP9Frame(t *testing.T, img *Image) []byte {
	t.Helper()
	return encodeVP9FrameWithSize(t, img, img.DW, img.DH)
}

func encodeVP9FrameWithSize(t *testing.T, img *Image, width, height uint32) []byte {
	t.Helper()

	encCtx := NewCodecCtx()
	defer CodecDestroy(encCtx)

	encIface := EncoderIfaceVP9()
	cfg := &CodecEncCfg{}
	CodecEncConfigDefault(encIface, cfg, 0)
	cfg.Deref()

	cfg.GW = width
	cfg.GH = height
	cfg.GTimebase = Rational{Num: 1, Den: 30}
	cfg.RcTargetBitrate = 500
	cfg.GPass = RcOnePass
	cfg.GLagInFrames = 0

	if err := Error(CodecEncInitVer(encCtx, encIface, cfg, 0, EncoderABIVersion)); err != nil {
		t.Logf("VP9 encode init failed for %dx%d: %v", width, height, err)
		return nil
	}

	if err := Error(CodecEncode(encCtx, img, 0, 1, 0, DlGoodQuality)); err != nil {
		t.Logf("VP9 encode failed: %v", err)
		return nil
	}

	CodecEncode(encCtx, nil, 0, 0, 0, DlGoodQuality)

	var iter CodecIter
	pkt := CodecGetCxData(encCtx, &iter)
	if pkt == nil {
		t.Logf("VP9: no packet for %dx%d", width, height)
		return nil
	}
	pkt.Deref()

	data := make([]byte, len(pkt.GetFrameData()))
	copy(data, pkt.GetFrameData())
	return data
}

func decodeFrame(t *testing.T, data []byte, isVP8 bool) *Image {
	t.Helper()

	decCtx := NewCodecCtx()
	defer CodecDestroy(decCtx)

	var decIface *CodecIface
	if isVP8 {
		decIface = DecoderIfaceVP8()
	} else {
		decIface = DecoderIfaceVP9()
	}

	CodecDecInitVer(decCtx, decIface, nil, 0, DecoderABIVersion)
	CodecDecode(decCtx, string(data), uint32(len(data)), nil, 0)

	var iter CodecIter
	img := CodecGetFrame(decCtx, &iter)
	if img == nil {
		t.Fatal("no decoded frame")
	}
	img.Deref()

	return img
}

func encodeMultipleFrames(t *testing.T, width, height uint32, count int, isVP8 bool) [][]byte {
	t.Helper()

	encCtx := NewCodecCtx()
	defer CodecDestroy(encCtx)

	var encIface *CodecIface
	if isVP8 {
		encIface = EncoderIfaceVP8()
	} else {
		encIface = EncoderIfaceVP9()
	}

	cfg := &CodecEncCfg{}
	CodecEncConfigDefault(encIface, cfg, 0)
	cfg.Deref()

	cfg.GW = width
	cfg.GH = height
	cfg.GTimebase = Rational{Num: 1, Den: 30}
	cfg.RcTargetBitrate = 500
	cfg.GPass = RcOnePass
	if !isVP8 {
		cfg.GLagInFrames = 0
	}

	CodecEncInitVer(encCtx, encIface, cfg, 0, EncoderABIVersion)

	img := ImageAlloc(nil, ImageFormatI420, width, height, 1)
	defer ImageFree(img)
	img.Deref()

	var packets [][]byte
	for i := 0; i < count; i++ {
		fillTestPattern(img, i)
		CodecEncode(encCtx, img, CodecPts(i), 1, 0, DlGoodQuality)

		var iter CodecIter
		for pkt := CodecGetCxData(encCtx, &iter); pkt != nil; pkt = CodecGetCxData(encCtx, &iter) {
			pkt.Deref()
			if pkt.Kind == CodecCxFramePkt {
				data := make([]byte, len(pkt.GetFrameData()))
				copy(data, pkt.GetFrameData())
				packets = append(packets, data)
			}
		}
	}

	// Flush
	if !isVP8 {
		CodecEncode(encCtx, nil, 0, 0, 0, DlGoodQuality)
		var iter CodecIter
		for pkt := CodecGetCxData(encCtx, &iter); pkt != nil; pkt = CodecGetCxData(encCtx, &iter) {
			pkt.Deref()
			if pkt.Kind == CodecCxFramePkt {
				data := make([]byte, len(pkt.GetFrameData()))
				copy(data, pkt.GetFrameData())
				packets = append(packets, data)
			}
		}
	}

	return packets
}

func allSameValue(data []byte) bool {
	if len(data) == 0 {
		return true
	}
	first := data[0]
	for _, b := range data {
		if b != first {
			return false
		}
	}
	return true
}

func encodeFrameWithNewImage(t *testing.T, width, height uint32, isVP8 bool) []byte {
	t.Helper()

	encCtx := NewCodecCtx()
	defer CodecDestroy(encCtx)

	var encIface *CodecIface
	if isVP8 {
		encIface = EncoderIfaceVP8()
	} else {
		encIface = EncoderIfaceVP9()
	}

	cfg := &CodecEncCfg{}
	CodecEncConfigDefault(encIface, cfg, 0)
	cfg.Deref()

	cfg.GW = width
	cfg.GH = height
	cfg.GTimebase = Rational{Num: 1, Den: 30}
	cfg.RcTargetBitrate = 500
	cfg.GPass = RcOnePass
	if !isVP8 {
		cfg.GLagInFrames = 0
	}

	if err := Error(CodecEncInitVer(encCtx, encIface, cfg, 0, EncoderABIVersion)); err != nil {
		t.Logf("encode init failed for %dx%d: %v", width, height, err)
		return nil
	}

	img := ImageAlloc(nil, ImageFormatI420, width, height, 1)
	if img == nil {
		t.Logf("failed to allocate image %dx%d", width, height)
		return nil
	}
	defer ImageFree(img)
	img.Deref()

	fillTestPattern(img, 0)

	if err := Error(CodecEncode(encCtx, img, 0, 1, 0, DlGoodQuality)); err != nil {
		t.Logf("encode failed: %v", err)
		return nil
	}

	if !isVP8 {
		CodecEncode(encCtx, nil, 0, 0, 0, DlGoodQuality)
	}

	var iter CodecIter
	pkt := CodecGetCxData(encCtx, &iter)
	if pkt == nil {
		t.Logf("no packet for %dx%d", width, height)
		return nil
	}
	pkt.Deref()

	data := make([]byte, len(pkt.GetFrameData()))
	copy(data, pkt.GetFrameData())
	return data
}

func fillTestPatternForSize(img *Image, frameNum int, width, height uint32) {
	h := int(height)
	w := int(width)
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
