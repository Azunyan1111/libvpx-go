package vpx

import (
	"testing"
)

// TestCodecEncCfgNonDefaultSize tests encoding with non-default resolution.
// This test verifies that cfg changes after Deref() are properly synced to C struct.
// Bug: PassRef() returns cached C struct without syncing Go struct changes.
func TestCodecEncCfgNonDefaultSize(t *testing.T) {
	const (
		width   = 640 // Non-default (default is 320)
		height  = 480 // Non-default (default is 240)
		bitrate = 500
	)

	t.Run("VP8", func(t *testing.T) {
		testEncodeWithSize(t, width, height, bitrate, true)
	})

	t.Run("VP9", func(t *testing.T) {
		testEncodeWithSize(t, width, height, bitrate, false)
	})
}

// TestCodecEncCfgMultipleResolutions tests various resolutions to ensure
// cfg changes are properly synced for all sizes.
func TestCodecEncCfgMultipleResolutions(t *testing.T) {
	resolutions := []struct {
		name string
		w, h uint32
	}{
		{"VGA", 640, 480},
		{"HD720", 1280, 720},
		{"CIF", 352, 288},
		{"QCIF", 176, 144},
		{"Custom", 400, 300},
	}

	for _, res := range resolutions {
		t.Run(res.name+"_VP8", func(t *testing.T) {
			testEncodeWithSize(t, res.w, res.h, 500, true)
		})
		t.Run(res.name+"_VP9", func(t *testing.T) {
			testEncodeWithSize(t, res.w, res.h, 500, false)
		})
	}
}

// testEncodeWithSize performs encoding with specified size.
// This will fail if PassRef() doesn't sync Go struct changes to C struct.
func testEncodeWithSize(t *testing.T, width, height, bitrate uint32, isVP8 bool) {
	t.Helper()

	codecName := "VP9"
	if isVP8 {
		codecName = "VP8"
	}

	// Create encoder context
	ctx := NewCodecCtx()
	if ctx == nil {
		t.Fatal("failed to create codec context")
	}
	defer CodecDestroy(ctx)

	// Get encoder interface
	var iface *CodecIface
	if isVP8 {
		iface = EncoderIfaceVP8()
	} else {
		iface = EncoderIfaceVP9()
	}

	// Get default config - this sets ref37e25db9 internally
	cfg := &CodecEncCfg{}
	if err := Error(CodecEncConfigDefault(iface, cfg, 0)); err != nil {
		t.Fatalf("failed to get default config: %v", err)
	}

	// Deref populates Go struct from C struct
	cfg.Deref()

	// Verify default values
	t.Logf("%s default config: GW=%d, GH=%d", codecName, cfg.GW, cfg.GH)

	// Change to non-default size
	// BUG: These changes won't be synced to C struct in PassRef()
	cfg.GW = width
	cfg.GH = height
	cfg.GTimebase = Rational{Num: 1, Den: 30}
	cfg.RcTargetBitrate = bitrate
	cfg.GPass = RcOnePass
	if !isVP8 {
		cfg.GLagInFrames = 0
	}

	t.Logf("%s modified config: GW=%d, GH=%d", codecName, cfg.GW, cfg.GH)

	// Initialize encoder - this calls PassRef() internally
	// If PassRef() doesn't sync, encoder will be initialized with default 320x240
	if err := Error(CodecEncInitVer(ctx, iface, cfg, 0, EncoderABIVersion)); err != nil {
		t.Fatalf("%s encoder init failed: %v", codecName, err)
	}

	// Create image with specified size
	img := ImageAlloc(nil, ImageFormatI420, width, height, 1)
	if img == nil {
		t.Fatalf("failed to allocate %dx%d image", width, height)
	}
	defer ImageFree(img)
	img.Deref()

	// Fill test pattern
	fillTestPattern(img, 0)

	// Encode multiple frames - this will fail with "invalid param" if encoder was initialized
	// with wrong size (320x240 instead of width x height)
	// VP9 requires multiple frames due to frame buffering
	numFrames := 1
	if !isVP8 {
		numFrames = 5
	}

	var totalBytes int

	for i := 0; i < numFrames; i++ {
		fillTestPattern(img, i)
		if err := Error(CodecEncode(ctx, img, CodecPts(i), 1, 0, DlGoodQuality)); err != nil {
			t.Fatalf("%s encode failed with %dx%d image at frame %d: %v", codecName, width, height, i, err)
		}

		// Collect packets after each frame
		var iter CodecIter
		for pkt := CodecGetCxData(ctx, &iter); pkt != nil; pkt = CodecGetCxData(ctx, &iter) {
			pkt.Deref()
			if pkt.Kind == CodecCxFramePkt {
				data := pkt.GetFrameData()
				totalBytes += len(data)
			}
		}
	}

	// Flush encoder
	CodecEncode(ctx, nil, 0, 0, 0, DlGoodQuality)

	// Collect remaining packets after flush
	var iter CodecIter
	for pkt := CodecGetCxData(ctx, &iter); pkt != nil; pkt = CodecGetCxData(ctx, &iter) {
		pkt.Deref()
		if pkt.Kind == CodecCxFramePkt {
			data := pkt.GetFrameData()
			totalBytes += len(data)
		}
	}

	if totalBytes == 0 {
		t.Fatalf("%s: no encoded data for %dx%d", codecName, width, height)
	}

	t.Logf("%s %dx%d: encoded %d bytes total", codecName, width, height, totalBytes)
}
