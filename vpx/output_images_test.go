package vpx

import (
	"image"
	"image/png"
	"os"
	"testing"
)

// TestOutputImages outputs original and decoded images as PNG files for visual comparison.
func TestOutputImages(t *testing.T) {
	const (
		width  = 320
		height = 240
	)

	// Create original image
	origImg := ImageAlloc(nil, ImageFormatI420, width, height, 1)
	if origImg == nil {
		t.Fatal("failed to allocate original image")
	}
	defer ImageFree(origImg)
	origImg.Deref()

	fillTestPattern(origImg, 0)

	// Save original image
	origRGBA := origImg.ImageRGBA()
	if err := savePNG("/tmp/original.png", origRGBA); err != nil {
		t.Fatalf("failed to save original image: %v", err)
	}
	t.Logf("saved: /tmp/original.png")

	// VP8: Encode and decode
	vp8Decoded := encodeDecodeVP8ForOutput(t, origImg)
	vp8RGBA := vp8Decoded.ImageRGBA()
	if err := savePNG("/tmp/vp8_decoded.png", vp8RGBA); err != nil {
		t.Fatalf("failed to save VP8 decoded image: %v", err)
	}
	t.Logf("saved: /tmp/vp8_decoded.png")

	// VP9: Encode and decode
	vp9Decoded := encodeDecodeVP9ForOutput(t, origImg)
	vp9RGBA := vp9Decoded.ImageRGBA()
	if err := savePNG("/tmp/vp9_decoded.png", vp9RGBA); err != nil {
		t.Fatalf("failed to save VP9 decoded image: %v", err)
	}
	t.Logf("saved: /tmp/vp9_decoded.png")

	// Calculate and log quality metrics
	originalY := extractYPlane(origImg)
	vp8Y := extractYPlane(vp8Decoded)
	vp9Y := extractYPlane(vp9Decoded)

	vp8PSNR := calculatePSNR(originalY, vp8Y)
	vp9PSNR := calculatePSNR(originalY, vp9Y)

	t.Logf("VP8 PSNR: %.2f dB", vp8PSNR)
	t.Logf("VP9 PSNR: %.2f dB", vp9PSNR)

	t.Log("")
	t.Log("Output files:")
	t.Log("  /tmp/original.png     - Original image")
	t.Log("  /tmp/vp8_decoded.png  - VP8 encoded/decoded")
	t.Log("  /tmp/vp9_decoded.png  - VP9 encoded/decoded")
}

func savePNG(path string, img *image.RGBA) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func encodeDecodeVP8ForOutput(t *testing.T, origImg *Image) *Image {
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
	cfg.RcTargetBitrate = 500
	cfg.GPass = RcOnePass

	CodecEncInitVer(encCtx, encIface, cfg, 0, EncoderABIVersion)
	CodecEncode(encCtx, origImg, 0, 1, 0, DlGoodQuality)

	var encIter CodecIter
	pkt := CodecGetCxData(encCtx, &encIter)
	if pkt == nil {
		t.Fatal("VP8: no encoded packet")
	}
	pkt.Deref()

	encodedData := make([]byte, len(pkt.GetFrameData()))
	copy(encodedData, pkt.GetFrameData())

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

func encodeDecodeVP9ForOutput(t *testing.T, origImg *Image) *Image {
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
	cfg.RcTargetBitrate = 500
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

	encodedData := make([]byte, len(pkt.GetFrameData()))
	copy(encodedData, pkt.GetFrameData())

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
