package vpx

import (
	"image"
	"image/color"
	"testing"
)

// TestYUVToRGBA demonstrates converting YUV frame to RGBA image.
func TestYUVToRGBA(t *testing.T) {
	const (
		width  = 320
		height = 240
	)

	// Encode and decode a frame to get real YUV data
	encodedData := encodeTestFrame(t, width, height)

	ctx := NewCodecCtx()
	defer CodecDestroy(ctx)

	iface := DecoderIfaceVP8()
	CodecDecInitVer(ctx, iface, nil, 0, DecoderABIVersion)
	CodecDecode(ctx, string(encodedData), uint32(len(encodedData)), nil, 0)

	var iter CodecIter
	img := CodecGetFrame(ctx, &iter)
	if img == nil {
		t.Fatal("failed to decode frame")
	}
	img.Deref()

	// Convert YUV to RGBA
	rgba := img.ImageRGBA()
	if rgba == nil {
		t.Fatal("ImageRGBA returned nil")
	}

	// Verify dimensions
	if rgba.Bounds().Dx() != int(width) || rgba.Bounds().Dy() != int(height) {
		t.Errorf("RGBA size mismatch: got %dx%d, want %dx%d",
			rgba.Bounds().Dx(), rgba.Bounds().Dy(), width, height)
	}

	// Verify pixel format (RGBA)
	if len(rgba.Pix) != int(width)*int(height)*4 {
		t.Errorf("RGBA pixel data size mismatch: got %d, want %d",
			len(rgba.Pix), width*height*4)
	}

	// Check alpha channel is fully opaque
	for y := 0; y < int(height); y++ {
		for x := 0; x < int(width); x++ {
			_, _, _, a := rgba.At(x, y).RGBA()
			if a != 0xFFFF {
				t.Errorf("alpha at (%d,%d) = %d, want 65535", x, y, a)
			}
		}
	}

	t.Logf("converted to RGBA: %dx%d, %d bytes", rgba.Bounds().Dx(), rgba.Bounds().Dy(), len(rgba.Pix))
}

// TestYUVToYCbCr demonstrates converting to Go's standard YCbCr format.
func TestYUVToYCbCr(t *testing.T) {
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
		t.Fatal("failed to decode frame")
	}
	img.Deref()

	// Convert to YCbCr
	ycbcr := img.ImageYCbCr()
	if ycbcr == nil {
		t.Fatal("ImageYCbCr returned nil")
	}

	// Verify dimensions
	if ycbcr.Bounds().Dx() != int(width) || ycbcr.Bounds().Dy() != int(height) {
		t.Errorf("YCbCr size mismatch: got %dx%d, want %dx%d",
			ycbcr.Bounds().Dx(), ycbcr.Bounds().Dy(), width, height)
	}

	// Verify subsample ratio
	if ycbcr.SubsampleRatio != image.YCbCrSubsampleRatio420 {
		t.Errorf("subsample ratio = %v, want YCbCrSubsampleRatio420", ycbcr.SubsampleRatio)
	}

	// Verify Y plane size
	expectedYSize := ycbcr.YStride * int(height)
	if len(ycbcr.Y) != expectedYSize {
		t.Errorf("Y plane size mismatch: got %d, want %d", len(ycbcr.Y), expectedYSize)
	}

	t.Logf("converted to YCbCr: %dx%d, Y=%d bytes, Cb=%d bytes, Cr=%d bytes",
		ycbcr.Bounds().Dx(), ycbcr.Bounds().Dy(), len(ycbcr.Y), len(ycbcr.Cb), len(ycbcr.Cr))
}

// TestYCbCrSubsampleFormats demonstrates different YCbCr subsample formats.
func TestYCbCrSubsampleFormats(t *testing.T) {
	tests := []struct {
		format      ImageFormat
		name        string
		expectRatio image.YCbCrSubsampleRatio
	}{
		{ImageFormatI420, "I420", image.YCbCrSubsampleRatio420},
		{ImageFormatI422, "I422", image.YCbCrSubsampleRatio422},
		{ImageFormatI440, "I440", image.YCbCrSubsampleRatio440},
	}

	const (
		width  = 64
		height = 64
	)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := ImageAlloc(nil, tt.format, width, height, 1)
			if img == nil {
				t.Fatalf("failed to allocate %s image", tt.name)
			}
			defer ImageFree(img)
			img.Deref()

			ycbcr := img.ImageYCbCr()
			if ycbcr == nil {
				t.Fatalf("ImageYCbCr returned nil for %s", tt.name)
			}

			if ycbcr.SubsampleRatio != tt.expectRatio {
				t.Errorf("%s: subsample ratio = %v, want %v",
					tt.name, ycbcr.SubsampleRatio, tt.expectRatio)
			}

			t.Logf("%s: Y=%d bytes, Cb=%d bytes, Cr=%d bytes, ratio=%v",
				tt.name, len(ycbcr.Y), len(ycbcr.Cb), len(ycbcr.Cr), ycbcr.SubsampleRatio)
		})
	}
}

// TestRGBAToDisplay demonstrates preparing RGBA for display.
func TestRGBAToDisplay(t *testing.T) {
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
		t.Fatal("failed to decode frame")
	}
	img.Deref()

	rgba := img.ImageRGBA()

	// Simulate accessing pixels for display
	var sumR, sumG, sumB uint64
	for y := 0; y < rgba.Bounds().Dy(); y++ {
		for x := 0; x < rgba.Bounds().Dx(); x++ {
			r, g, b, _ := rgba.At(x, y).RGBA()
			sumR += uint64(r >> 8)
			sumG += uint64(g >> 8)
			sumB += uint64(b >> 8)
		}
	}

	pixelCount := uint64(width * height)
	avgR := sumR / pixelCount
	avgG := sumG / pixelCount
	avgB := sumB / pixelCount

	t.Logf("average color: R=%d, G=%d, B=%d", avgR, avgG, avgB)
}

// TestYCbCrCompatibility demonstrates compatibility with Go's image library.
func TestYCbCrCompatibility(t *testing.T) {
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
		t.Fatal("failed to decode frame")
	}
	img.Deref()

	ycbcr := img.ImageYCbCr()

	// Verify it implements image.Image interface
	var goImage image.Image = ycbcr
	_ = goImage

	// Verify ColorModel
	if ycbcr.ColorModel() != color.YCbCrModel {
		t.Error("color model is not YCbCrModel")
	}

	// Sample some pixels
	for y := 0; y < height; y += height / 4 {
		for x := 0; x < width; x += width / 4 {
			c := ycbcr.At(x, y)
			yc, cb, cr := color.YCbCrModel.Convert(c).(color.YCbCr).Y,
				color.YCbCrModel.Convert(c).(color.YCbCr).Cb,
				color.YCbCrModel.Convert(c).(color.YCbCr).Cr
			t.Logf("pixel (%d,%d): Y=%d, Cb=%d, Cr=%d", x, y, yc, cb, cr)
		}
	}
}

// TestConversionPipeline demonstrates a full conversion pipeline.
func TestConversionPipeline(t *testing.T) {
	const (
		width      = 320
		height     = 240
		frameCount = 5
	)

	packets := encodeTestFrames(t, width, height, frameCount)

	ctx := NewCodecCtx()
	defer CodecDestroy(ctx)

	iface := DecoderIfaceVP8()
	CodecDecInitVer(ctx, iface, nil, 0, DecoderABIVersion)

	var rgbaFrames []*image.RGBA

	for _, pkt := range packets {
		CodecDecode(ctx, string(pkt), uint32(len(pkt)), nil, 0)

		var iter CodecIter
		for img := CodecGetFrame(ctx, &iter); img != nil; img = CodecGetFrame(ctx, &iter) {
			img.Deref()

			// Convert to RGBA
			rgba := img.ImageRGBA()
			rgbaFrames = append(rgbaFrames, rgba)
		}
	}

	if len(rgbaFrames) != frameCount {
		t.Errorf("converted %d frames, want %d", len(rgbaFrames), frameCount)
	}

	for i, frame := range rgbaFrames {
		t.Logf("frame %d: %dx%d RGBA", i, frame.Bounds().Dx(), frame.Bounds().Dy())
	}
}
