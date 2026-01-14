package vpx

import (
	"testing"
)

// TestVP8FullPipeline demonstrates a complete VP8 encode-decode-reencode pipeline.
func TestVP8FullPipeline(t *testing.T) {
	const (
		width      = 320
		height     = 240
		frameCount = 10
	)

	t.Log("Step 1: Encoding generated YUV frames to VP8")
	encodedPackets := encodeTestFrames(t, width, height, frameCount)
	if len(encodedPackets) == 0 {
		t.Fatal("no packets encoded")
	}
	t.Logf("encoded %d packets", len(encodedPackets))

	t.Log("Step 2: Decoding VP8 packets")
	decodedFrames := decodeVP8Packets(t, encodedPackets)
	if len(decodedFrames) == 0 {
		t.Fatal("no frames decoded")
	}
	t.Logf("decoded %d frames", len(decodedFrames))

	if len(decodedFrames) != frameCount {
		t.Errorf("expected %d frames, got %d", frameCount, len(decodedFrames))
	}

	t.Log("Step 3: Re-encoding decoded frames to VP8")
	reEncodedPackets := reencodeVP8Frames(t, decodedFrames)
	if len(reEncodedPackets) == 0 {
		t.Fatal("no packets re-encoded")
	}
	t.Logf("re-encoded %d packets", len(reEncodedPackets))

	t.Log("Step 4: Verifying re-encoded data by decoding again")
	finalFrames := decodeVP8Packets(t, reEncodedPackets)
	if len(finalFrames) == 0 {
		t.Fatal("no frames decoded from re-encoded data")
	}
	t.Logf("final decoded %d frames", len(finalFrames))

	if len(finalFrames) != frameCount {
		t.Errorf("expected %d final frames, got %d", frameCount, len(finalFrames))
	}

	t.Log("VP8 full pipeline test completed successfully")
}

// TestVP9FullPipeline demonstrates a complete VP9 encode-decode-reencode pipeline.
func TestVP9FullPipeline(t *testing.T) {
	const (
		width      = 320
		height     = 240
		frameCount = 10
	)

	t.Log("Step 1: Encoding generated YUV frames to VP9")
	encodedPackets := encodeVP9TestFrames(t, width, height, frameCount)
	if len(encodedPackets) == 0 {
		t.Fatal("no packets encoded")
	}
	t.Logf("encoded %d packets", len(encodedPackets))

	t.Log("Step 2: Decoding VP9 packets")
	decodedFrames := decodeVP9Packets(t, encodedPackets)
	if len(decodedFrames) == 0 {
		t.Fatal("no frames decoded")
	}
	t.Logf("decoded %d frames", len(decodedFrames))

	if len(decodedFrames) != frameCount {
		t.Errorf("expected %d frames, got %d", frameCount, len(decodedFrames))
	}

	t.Log("VP9 full pipeline test completed successfully")
}

// TestCodecVersionInfo demonstrates getting codec version information.
func TestCodecVersionInfo(t *testing.T) {
	version := CodecVersion()
	versionStr := CodecVersionStr()
	buildConfig := CodecBuildConfig()

	t.Logf("libvpx version: %d (%s)", version, versionStr)
	t.Logf("build config: %s", buildConfig)

	if version == 0 {
		t.Error("CodecVersion returned 0")
	}
	if versionStr == "" {
		t.Error("CodecVersionStr returned empty string")
	}
}

// decodeVP8Packets decodes VP8 packets and returns decoded frames.
func decodeVP8Packets(t *testing.T, packets [][]byte) []*Image {
	t.Helper()

	ctx := NewCodecCtx()
	defer CodecDestroy(ctx)

	iface := DecoderIfaceVP8()
	if err := Error(CodecDecInitVer(ctx, iface, nil, 0, DecoderABIVersion)); err != nil {
		t.Fatalf("failed to initialize decoder: %v", err)
	}

	var frames []*Image
	for i, pkt := range packets {
		if err := Error(CodecDecode(ctx, string(pkt), uint32(len(pkt)), nil, 0)); err != nil {
			t.Fatalf("failed to decode packet %d: %v", i, err)
		}

		var iter CodecIter
		for img := CodecGetFrame(ctx, &iter); img != nil; img = CodecGetFrame(ctx, &iter) {
			img.Deref()
			frames = append(frames, copyDecodedFrame(img))
		}
	}

	return frames
}

// decodeVP9Packets decodes VP9 packets and returns decoded frames.
func decodeVP9Packets(t *testing.T, packets [][]byte) []*Image {
	t.Helper()

	ctx := NewCodecCtx()
	defer CodecDestroy(ctx)

	iface := DecoderIfaceVP9()
	if err := Error(CodecDecInitVer(ctx, iface, nil, 0, DecoderABIVersion)); err != nil {
		t.Fatalf("failed to initialize decoder: %v", err)
	}

	var frames []*Image
	for i, pkt := range packets {
		if err := Error(CodecDecode(ctx, string(pkt), uint32(len(pkt)), nil, 0)); err != nil {
			t.Fatalf("failed to decode packet %d: %v", i, err)
		}

		var iter CodecIter
		for img := CodecGetFrame(ctx, &iter); img != nil; img = CodecGetFrame(ctx, &iter) {
			img.Deref()
			frames = append(frames, copyDecodedFrame(img))
		}
	}

	return frames
}

// reencodeVP8Frames re-encodes decoded frames to VP8.
func reencodeVP8Frames(t *testing.T, frames []*Image) [][]byte {
	t.Helper()

	if len(frames) == 0 {
		return nil
	}

	ctx := NewCodecCtx()
	defer CodecDestroy(ctx)

	iface := EncoderIfaceVP8()
	cfg := &CodecEncCfg{}
	CodecEncConfigDefault(iface, cfg, 0)
	cfg.Deref()

	width := frames[0].DW
	height := frames[0].DH

	cfg.GW = width
	cfg.GH = height
	cfg.GTimebase = Rational{Num: 1, Den: 30}
	cfg.RcTargetBitrate = 200
	cfg.GPass = RcOnePass

	if err := Error(CodecEncInitVer(ctx, iface, cfg, 0, EncoderABIVersion)); err != nil {
		t.Fatalf("failed to initialize encoder: %v", err)
	}

	var packets [][]byte

	for i, frame := range frames {
		img := ImageAlloc(nil, ImageFormatI420, width, height, 1)
		img.Deref()

		copyImageData(img, frame)

		if err := Error(CodecEncode(ctx, img, CodecPts(i), 1, 0, DlGoodQuality)); err != nil {
			ImageFree(img)
			t.Fatalf("failed to encode frame %d: %v", i, err)
		}

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

		ImageFree(img)
	}

	// Flush
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
