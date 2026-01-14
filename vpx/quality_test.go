package vpx

import (
	"math"
	"testing"
	"unsafe"
)

// calculatePSNR calculates Peak Signal-to-Noise Ratio between two byte slices.
// Returns PSNR in dB. Higher is better. Typical values:
// - 30-40 dB: Good quality
// - 40-50 dB: Very good quality
// - > 50 dB: Excellent quality
func calculatePSNR(original, decoded []byte) float64 {
	if len(original) != len(decoded) {
		return 0
	}
	if len(original) == 0 {
		return 0
	}

	var mse float64
	for i := range original {
		diff := float64(original[i]) - float64(decoded[i])
		mse += diff * diff
	}
	mse /= float64(len(original))

	if mse == 0 {
		return math.Inf(1) // Identical images
	}

	// PSNR = 10 * log10(MAX^2 / MSE) where MAX=255 for 8-bit
	return 10 * math.Log10(255*255/mse)
}

// calculateSSIM calculates a simplified Structural Similarity Index.
// Returns value between 0 and 1. Higher is better.
// This is a simplified version that compares local statistics.
func calculateSSIM(original, decoded []byte, width, height, stride int) float64 {
	if len(original) == 0 || len(decoded) == 0 {
		return 0
	}

	const (
		windowSize = 8
		c1         = 6.5025   // (0.01 * 255)^2
		c2         = 58.5225  // (0.03 * 255)^2
	)

	var ssimSum float64
	var count int

	for y := 0; y <= height-windowSize; y += windowSize {
		for x := 0; x <= width-windowSize; x += windowSize {
			var sumOrig, sumDec float64
			var sumOrigSq, sumDecSq, sumOrigDec float64

			for wy := 0; wy < windowSize; wy++ {
				for wx := 0; wx < windowSize; wx++ {
					idx := (y+wy)*stride + (x + wx)
					o := float64(original[idx])
					d := float64(decoded[idx])

					sumOrig += o
					sumDec += d
					sumOrigSq += o * o
					sumDecSq += d * d
					sumOrigDec += o * d
				}
			}

			n := float64(windowSize * windowSize)
			muOrig := sumOrig / n
			muDec := sumDec / n
			sigmaOrigSq := sumOrigSq/n - muOrig*muOrig
			sigmaDecSq := sumDecSq/n - muDec*muDec
			sigmaOrigDec := sumOrigDec/n - muOrig*muDec

			numerator := (2*muOrig*muDec + c1) * (2*sigmaOrigDec + c2)
			denominator := (muOrig*muOrig + muDec*muDec + c1) * (sigmaOrigSq + sigmaDecSq + c2)

			ssimSum += numerator / denominator
			count++
		}
	}

	if count == 0 {
		return 0
	}
	return ssimSum / float64(count)
}

// extractYPlane extracts Y plane data from Image as a contiguous slice.
func extractYPlane(img *Image) []byte {
	h := int(img.DH)
	w := int(img.DW)
	stride := int(img.Stride[PlaneY])

	result := make([]byte, w*h)
	src := (*(*[1 << 30]byte)(unsafe.Pointer(img.Planes[PlaneY])))[:stride*h]

	for row := 0; row < h; row++ {
		copy(result[row*w:(row+1)*w], src[row*stride:row*stride+w])
	}
	return result
}

// TestVP8ImageQuality verifies that VP8 encoding/decoding preserves image quality.
func TestVP8ImageQuality(t *testing.T) {
	const (
		width   = 320
		height  = 240
		minPSNR = 30.0 // Minimum acceptable PSNR in dB
		minSSIM = 0.85 // Minimum acceptable SSIM
	)

	// Create and fill original image
	origImg := ImageAlloc(nil, ImageFormatI420, width, height, 1)
	if origImg == nil {
		t.Fatal("failed to allocate original image")
	}
	defer ImageFree(origImg)
	origImg.Deref()

	fillTestPattern(origImg, 0)

	// Save original Y plane
	originalY := extractYPlane(origImg)

	// Encode
	encCtx := NewCodecCtx()
	defer CodecDestroy(encCtx)

	encIface := EncoderIfaceVP8()
	cfg := &CodecEncCfg{}
	CodecEncConfigDefault(encIface, cfg, 0)
	cfg.Deref()

	cfg.GW = width
	cfg.GH = height
	cfg.GTimebase = Rational{Num: 1, Den: 30}
	cfg.RcTargetBitrate = 500 // Higher bitrate for better quality
	cfg.GPass = RcOnePass

	if err := Error(CodecEncInitVer(encCtx, encIface, cfg, 0, EncoderABIVersion)); err != nil {
		t.Fatalf("failed to initialize VP8 encoder: %v", err)
	}

	if err := Error(CodecEncode(encCtx, origImg, 0, 1, 0, DlGoodQuality)); err != nil {
		t.Fatalf("failed to encode: %v", err)
	}

	var encIter CodecIter
	pkt := CodecGetCxData(encCtx, &encIter)
	if pkt == nil {
		t.Fatal("no encoded packet")
	}
	pkt.Deref()
	encodedData := pkt.GetFrameData()

	// Decode
	decCtx := NewCodecCtx()
	defer CodecDestroy(decCtx)

	decIface := DecoderIfaceVP8()
	if err := Error(CodecDecInitVer(decCtx, decIface, nil, 0, DecoderABIVersion)); err != nil {
		t.Fatalf("failed to initialize VP8 decoder: %v", err)
	}

	if err := Error(CodecDecode(decCtx, string(encodedData), uint32(len(encodedData)), nil, 0)); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	var decIter CodecIter
	decodedImg := CodecGetFrame(decCtx, &decIter)
	if decodedImg == nil {
		t.Fatal("no decoded frame")
	}
	decodedImg.Deref()

	// Extract decoded Y plane
	decodedY := extractYPlane(decodedImg)

	// Calculate quality metrics
	psnr := calculatePSNR(originalY, decodedY)
	ssim := calculateSSIM(originalY, decodedY, width, height, width)

	t.Logf("VP8 Quality - PSNR: %.2f dB, SSIM: %.4f", psnr, ssim)

	if psnr < minPSNR {
		t.Errorf("VP8 PSNR too low: %.2f dB < %.2f dB", psnr, minPSNR)
	}
	if ssim < minSSIM {
		t.Errorf("VP8 SSIM too low: %.4f < %.4f", ssim, minSSIM)
	}
}

// TestVP9ImageQuality verifies that VP9 encoding/decoding preserves image quality.
func TestVP9ImageQuality(t *testing.T) {
	const (
		width   = 320
		height  = 240
		minPSNR = 30.0
		minSSIM = 0.85
	)

	// Create and fill original image
	origImg := ImageAlloc(nil, ImageFormatI420, width, height, 1)
	if origImg == nil {
		t.Fatal("failed to allocate original image")
	}
	defer ImageFree(origImg)
	origImg.Deref()

	fillTestPattern(origImg, 0)

	// Save original Y plane
	originalY := extractYPlane(origImg)

	// Encode with VP9
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
		t.Fatalf("failed to initialize VP9 encoder: %v", err)
	}

	if err := Error(CodecEncode(encCtx, origImg, 0, 1, 0, DlGoodQuality)); err != nil {
		t.Fatalf("failed to encode: %v", err)
	}

	// VP9 requires flush
	CodecEncode(encCtx, nil, 0, 0, 0, DlGoodQuality)

	var encIter CodecIter
	pkt := CodecGetCxData(encCtx, &encIter)
	if pkt == nil {
		t.Fatal("no encoded packet")
	}
	pkt.Deref()
	encodedData := pkt.GetFrameData()

	// Decode
	decCtx := NewCodecCtx()
	defer CodecDestroy(decCtx)

	decIface := DecoderIfaceVP9()
	if err := Error(CodecDecInitVer(decCtx, decIface, nil, 0, DecoderABIVersion)); err != nil {
		t.Fatalf("failed to initialize VP9 decoder: %v", err)
	}

	if err := Error(CodecDecode(decCtx, string(encodedData), uint32(len(encodedData)), nil, 0)); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	var decIter CodecIter
	decodedImg := CodecGetFrame(decCtx, &decIter)
	if decodedImg == nil {
		t.Fatal("no decoded frame")
	}
	decodedImg.Deref()

	// Extract decoded Y plane
	decodedY := extractYPlane(decodedImg)

	// Calculate quality metrics
	psnr := calculatePSNR(originalY, decodedY)
	ssim := calculateSSIM(originalY, decodedY, width, height, width)

	t.Logf("VP9 Quality - PSNR: %.2f dB, SSIM: %.4f", psnr, ssim)

	if psnr < minPSNR {
		t.Errorf("VP9 PSNR too low: %.2f dB < %.2f dB", psnr, minPSNR)
	}
	if ssim < minSSIM {
		t.Errorf("VP9 SSIM too low: %.4f < %.4f", ssim, minSSIM)
	}
}

// TestVP8VSP9Quality compares VP8 and VP9 quality at same bitrate.
func TestVP8VSP9Quality(t *testing.T) {
	const (
		width   = 320
		height  = 240
		bitrate = 300
	)

	// Create original image
	origImg := ImageAlloc(nil, ImageFormatI420, width, height, 1)
	if origImg == nil {
		t.Fatal("failed to allocate original image")
	}
	defer ImageFree(origImg)
	origImg.Deref()

	fillTestPattern(origImg, 0)
	originalY := extractYPlane(origImg)

	// Encode and decode with VP8
	vp8Decoded := encodeDecodeVP8(t, origImg, bitrate)
	vp8Y := extractYPlane(vp8Decoded)
	vp8PSNR := calculatePSNR(originalY, vp8Y)
	vp8SSIM := calculateSSIM(originalY, vp8Y, width, height, width)

	// Encode and decode with VP9
	vp9Decoded := encodeDecodeVP9(t, origImg, bitrate)
	vp9Y := extractYPlane(vp9Decoded)
	vp9PSNR := calculatePSNR(originalY, vp9Y)
	vp9SSIM := calculateSSIM(originalY, vp9Y, width, height, width)

	t.Logf("VP8 at %d kbps - PSNR: %.2f dB, SSIM: %.4f", bitrate, vp8PSNR, vp8SSIM)
	t.Logf("VP9 at %d kbps - PSNR: %.2f dB, SSIM: %.4f", bitrate, vp9PSNR, vp9SSIM)

	// Both should have reasonable quality
	if vp8PSNR < 25.0 {
		t.Errorf("VP8 quality too low: PSNR %.2f dB", vp8PSNR)
	}
	if vp9PSNR < 25.0 {
		t.Errorf("VP9 quality too low: PSNR %.2f dB", vp9PSNR)
	}
}

// TestMultiFrameQuality verifies quality across multiple frames.
func TestMultiFrameQuality(t *testing.T) {
	const (
		width      = 320
		height     = 240
		frameCount = 5
		minPSNR    = 28.0
	)

	// Test VP8
	t.Run("VP8", func(t *testing.T) {
		testMultiFrameQualityCodec(t, true, width, height, frameCount, minPSNR)
	})

	// Test VP9
	t.Run("VP9", func(t *testing.T) {
		testMultiFrameQualityCodec(t, false, width, height, frameCount, minPSNR)
	})
}

func testMultiFrameQualityCodec(t *testing.T, isVP8 bool, width, height uint32, frameCount int, minPSNR float64) {
	t.Helper()

	codecName := "VP9"
	if isVP8 {
		codecName = "VP8"
	}

	// Setup encoder
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
		t.Fatalf("failed to initialize %s encoder: %v", codecName, err)
	}

	// Create test image and save all original frames first
	img := ImageAlloc(nil, ImageFormatI420, width, height, 1)
	defer ImageFree(img)
	img.Deref()

	originalFrames := make([][]byte, frameCount)
	encodedPackets := make([][]byte, 0, frameCount)

	// Phase 1: Encode all frames and save originals
	for i := 0; i < frameCount; i++ {
		fillTestPattern(img, i)
		originalFrames[i] = extractYPlane(img)

		CodecEncode(encCtx, img, CodecPts(i), 1, 0, DlGoodQuality)

		var encIter CodecIter
		for pkt := CodecGetCxData(encCtx, &encIter); pkt != nil; pkt = CodecGetCxData(encCtx, &encIter) {
			pkt.Deref()
			if pkt.Kind == CodecCxFramePkt {
				data := pkt.GetFrameData()
				cpy := make([]byte, len(data))
				copy(cpy, data)
				encodedPackets = append(encodedPackets, cpy)
			}
		}
	}

	// Flush for VP9
	if !isVP8 {
		CodecEncode(encCtx, nil, 0, 0, 0, DlGoodQuality)
		var encIter CodecIter
		for pkt := CodecGetCxData(encCtx, &encIter); pkt != nil; pkt = CodecGetCxData(encCtx, &encIter) {
			pkt.Deref()
			if pkt.Kind == CodecCxFramePkt {
				data := pkt.GetFrameData()
				cpy := make([]byte, len(data))
				copy(cpy, data)
				encodedPackets = append(encodedPackets, cpy)
			}
		}
	}

	// Phase 2: Decode all packets
	decCtx := NewCodecCtx()
	defer CodecDestroy(decCtx)

	var decIface *CodecIface
	if isVP8 {
		decIface = DecoderIfaceVP8()
	} else {
		decIface = DecoderIfaceVP9()
	}

	if err := Error(CodecDecInitVer(decCtx, decIface, nil, 0, DecoderABIVersion)); err != nil {
		t.Fatalf("failed to initialize %s decoder: %v", codecName, err)
	}

	var decodedFrames [][]byte
	for _, pktData := range encodedPackets {
		CodecDecode(decCtx, string(pktData), uint32(len(pktData)), nil, 0)

		var decIter CodecIter
		for decImg := CodecGetFrame(decCtx, &decIter); decImg != nil; decImg = CodecGetFrame(decCtx, &decIter) {
			decImg.Deref()
			decodedFrames = append(decodedFrames, extractYPlane(decImg))
		}
	}

	if len(decodedFrames) == 0 {
		t.Fatalf("%s: no frames decoded", codecName)
	}

	// Phase 3: Compare quality
	var totalPSNR float64
	compareCount := len(decodedFrames)
	if compareCount > len(originalFrames) {
		compareCount = len(originalFrames)
	}

	for i := 0; i < compareCount; i++ {
		psnr := calculatePSNR(originalFrames[i], decodedFrames[i])
		totalPSNR += psnr
		t.Logf("%s frame %d: PSNR %.2f dB", codecName, i, psnr)
	}

	avgPSNR := totalPSNR / float64(compareCount)
	t.Logf("%s average PSNR: %.2f dB over %d frames", codecName, avgPSNR, compareCount)

	if avgPSNR < minPSNR {
		t.Errorf("%s average PSNR too low: %.2f dB < %.2f dB", codecName, avgPSNR, minPSNR)
	}
}

// encodeDecodeVP8 encodes and decodes an image using VP8, returns decoded image.
func encodeDecodeVP8(t *testing.T, origImg *Image, bitrate uint32) *Image {
	t.Helper()

	width := origImg.DW
	height := origImg.DH

	// Encode
	encCtx := NewCodecCtx()
	defer CodecDestroy(encCtx)

	encIface := EncoderIfaceVP8()
	cfg := &CodecEncCfg{}
	CodecEncConfigDefault(encIface, cfg, 0)
	cfg.Deref()

	cfg.GW = width
	cfg.GH = height
	cfg.GTimebase = Rational{Num: 1, Den: 30}
	cfg.RcTargetBitrate = bitrate
	cfg.GPass = RcOnePass

	CodecEncInitVer(encCtx, encIface, cfg, 0, EncoderABIVersion)
	CodecEncode(encCtx, origImg, 0, 1, 0, DlGoodQuality)

	var encIter CodecIter
	pkt := CodecGetCxData(encCtx, &encIter)
	if pkt == nil {
		t.Fatal("VP8: no encoded packet")
	}
	pkt.Deref()
	encodedData := pkt.GetFrameData()

	// Decode
	decCtx := NewCodecCtx()
	defer CodecDestroy(decCtx)

	decIface := DecoderIfaceVP8()
	CodecDecInitVer(decCtx, decIface, nil, 0, DecoderABIVersion)
	CodecDecode(decCtx, string(encodedData), uint32(len(encodedData)), nil, 0)

	var decIter CodecIter
	decodedImg := CodecGetFrame(decCtx, &decIter)
	if decodedImg == nil {
		t.Fatal("VP8: no decoded frame")
	}
	decodedImg.Deref()

	return decodedImg
}

// encodeDecodeVP9 encodes and decodes an image using VP9, returns decoded image.
func encodeDecodeVP9(t *testing.T, origImg *Image, bitrate uint32) *Image {
	t.Helper()

	width := origImg.DW
	height := origImg.DH

	// Encode
	encCtx := NewCodecCtx()
	defer CodecDestroy(encCtx)

	encIface := EncoderIfaceVP9()
	cfg := &CodecEncCfg{}
	CodecEncConfigDefault(encIface, cfg, 0)
	cfg.Deref()

	cfg.GW = width
	cfg.GH = height
	cfg.GTimebase = Rational{Num: 1, Den: 30}
	cfg.RcTargetBitrate = bitrate
	cfg.GPass = RcOnePass
	cfg.GLagInFrames = 0

	CodecEncInitVer(encCtx, encIface, cfg, 0, EncoderABIVersion)
	CodecEncode(encCtx, origImg, 0, 1, 0, DlGoodQuality)
	CodecEncode(encCtx, nil, 0, 0, 0, DlGoodQuality) // Flush

	var encIter CodecIter
	pkt := CodecGetCxData(encCtx, &encIter)
	if pkt == nil {
		t.Fatal("VP9: no encoded packet")
	}
	pkt.Deref()
	encodedData := pkt.GetFrameData()

	// Decode
	decCtx := NewCodecCtx()
	defer CodecDestroy(decCtx)

	decIface := DecoderIfaceVP9()
	CodecDecInitVer(decCtx, decIface, nil, 0, DecoderABIVersion)
	CodecDecode(decCtx, string(encodedData), uint32(len(encodedData)), nil, 0)

	var decIter CodecIter
	decodedImg := CodecGetFrame(decCtx, &decIter)
	if decodedImg == nil {
		t.Fatal("VP9: no decoded frame")
	}
	decodedImg.Deref()

	return decodedImg
}
