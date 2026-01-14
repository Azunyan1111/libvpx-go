package vpx

import (
	"testing"
	"unsafe"
)

// TestFrameExtraction demonstrates extracting individual frames from video.
func TestFrameExtraction(t *testing.T) {
	const (
		width      = 320
		height     = 240
		frameCount = 10
	)

	// Encode test video
	packets := encodeTestFrames(t, width, height, frameCount)

	// Initialize decoder
	ctx := NewCodecCtx()
	defer CodecDestroy(ctx)

	iface := DecoderIfaceVP8()
	CodecDecInitVer(ctx, iface, nil, 0, DecoderABIVersion)

	// Extract each frame
	var frames []*Image
	for _, pkt := range packets {
		CodecDecode(ctx, string(pkt), uint32(len(pkt)), nil, 0)

		var iter CodecIter
		for img := CodecGetFrame(ctx, &iter); img != nil; img = CodecGetFrame(ctx, &iter) {
			img.Deref()

			// Copy frame data (decoder owns the original)
			frameCopy := copyDecodedFrame(img)
			frames = append(frames, frameCopy)
		}
	}

	if len(frames) != frameCount {
		t.Errorf("extracted %d frames, want %d", len(frames), frameCount)
	}

	for i, frame := range frames {
		t.Logf("frame %d: %dx%d, format=%v", i, frame.DW, frame.DH, frame.Fmt)
	}
}

// TestFrameSeek demonstrates seeking to specific frame (re-decode from keyframe).
func TestFrameSeek(t *testing.T) {
	const (
		width      = 320
		height     = 240
		frameCount = 30
		targetFrame = 15
	)

	// Encode with keyframes every 10 frames
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
	cfg.KfMaxDist = 10 // Keyframe every 10 frames

	CodecEncInitVer(ctx, iface, cfg, 0, EncoderABIVersion)

	img := ImageAlloc(nil, ImageFormatI420, width, height, 1)
	defer ImageFree(img)
	img.Deref()

	type packetInfo struct {
		data       []byte
		pts        CodecPts
		isKeyframe bool
	}
	var packets []packetInfo

	for i := 0; i < frameCount; i++ {
		fillTestPattern(img, i)
		CodecEncode(ctx, img, CodecPts(i), 1, 0, DlGoodQuality)

		var iter CodecIter
		for pkt := CodecGetCxData(ctx, &iter); pkt != nil; pkt = CodecGetCxData(ctx, &iter) {
			pkt.Deref()
			if pkt.Kind == CodecCxFramePkt {
				data := pkt.GetFrameData()
				cpy := make([]byte, len(data))
				copy(cpy, data)
				packets = append(packets, packetInfo{
					data:       cpy,
					pts:        pkt.GetFramePts(),
					isKeyframe: pkt.IsKeyframe(),
				})
			}
		}
	}

	// Find keyframe before target
	var keyframeIdx int
	for i := targetFrame; i >= 0; i-- {
		if packets[i].isKeyframe {
			keyframeIdx = i
			break
		}
	}
	t.Logf("seeking to frame %d, starting from keyframe at %d", targetFrame, keyframeIdx)

	// Decode from keyframe to target
	decCtx := NewCodecCtx()
	defer CodecDestroy(decCtx)

	decIface := DecoderIfaceVP8()
	CodecDecInitVer(decCtx, decIface, nil, 0, DecoderABIVersion)

	var targetImg *Image
	for i := keyframeIdx; i <= targetFrame; i++ {
		CodecDecode(decCtx, string(packets[i].data), uint32(len(packets[i].data)), nil, 0)

		var iter CodecIter
		for decImg := CodecGetFrame(decCtx, &iter); decImg != nil; decImg = CodecGetFrame(decCtx, &iter) {
			decImg.Deref()
			if i == targetFrame {
				targetImg = copyDecodedFrame(decImg)
			}
		}
	}

	if targetImg == nil {
		t.Fatal("failed to seek to target frame")
	}

	t.Logf("successfully seeked to frame %d: %dx%d", targetFrame, targetImg.DW, targetImg.DH)
}

// TestFrameTimestamp demonstrates handling frame timestamps.
func TestFrameTimestamp(t *testing.T) {
	const (
		width     = 320
		height    = 240
		fps       = 30
		duration  = 2 // seconds
	)

	frameCount := fps * duration

	ctx := NewCodecCtx()
	defer CodecDestroy(ctx)

	iface := EncoderIfaceVP8()
	cfg := &CodecEncCfg{}
	CodecEncConfigDefault(iface, cfg, 0)
	cfg.Deref()
	cfg.GW = width
	cfg.GH = height
	cfg.GTimebase = Rational{Num: 1, Den: fps}
	cfg.RcTargetBitrate = 200
	cfg.GPass = RcOnePass

	CodecEncInitVer(ctx, iface, cfg, 0, EncoderABIVersion)

	img := ImageAlloc(nil, ImageFormatI420, width, height, 1)
	defer ImageFree(img)
	img.Deref()

	type frameInfo struct {
		pts      CodecPts
		duration uint
	}
	var frameInfos []frameInfo

	for i := 0; i < frameCount; i++ {
		fillTestPattern(img, i)

		// pts is frame number (in timebase units)
		pts := CodecPts(i)

		CodecEncode(ctx, img, pts, 1, 0, DlGoodQuality)

		var iter CodecIter
		for pkt := CodecGetCxData(ctx, &iter); pkt != nil; pkt = CodecGetCxData(ctx, &iter) {
			pkt.Deref()
			if pkt.Kind == CodecCxFramePkt {
				frameInfos = append(frameInfos, frameInfo{
					pts:      pkt.GetFramePts(),
					duration: pkt.GetFrameDuration(),
				})
			}
		}
	}

	// Verify timestamps
	for i, info := range frameInfos {
		expectedPts := CodecPts(i)
		if info.pts != expectedPts {
			t.Errorf("frame %d: pts=%d, want %d", i, info.pts, expectedPts)
		}

		// Calculate time in seconds
		timeSeconds := float64(info.pts) / float64(fps)
		t.Logf("frame %d: pts=%d (%.3fs), duration=%d", i, info.pts, timeSeconds, info.duration)
	}

	totalDurationPts := frameInfos[len(frameInfos)-1].pts
	totalSeconds := float64(totalDurationPts) / float64(fps)
	t.Logf("total duration: %.2f seconds (%d frames)", totalSeconds, len(frameInfos))
}

// TestFrameDroppable demonstrates identifying droppable frames.
func TestFrameDroppable(t *testing.T) {
	const (
		width      = 320
		height     = 240
		frameCount = 20
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
	cfg.GErrorResilient = ErrorResilientDefault

	CodecEncInitVer(ctx, iface, cfg, 0, EncoderABIVersion)

	img := ImageAlloc(nil, ImageFormatI420, width, height, 1)
	defer ImageFree(img)
	img.Deref()

	var keyframes, droppable int

	for i := 0; i < frameCount; i++ {
		fillTestPattern(img, i)
		CodecEncode(ctx, img, CodecPts(i), 1, 0, DlGoodQuality)

		var iter CodecIter
		for pkt := CodecGetCxData(ctx, &iter); pkt != nil; pkt = CodecGetCxData(ctx, &iter) {
			pkt.Deref()
			if pkt.Kind == CodecCxFramePkt {
				flags := pkt.GetFrameFlags()
				if flags&FrameIsKey != 0 {
					keyframes++
					t.Logf("frame %d: keyframe", i)
				}
				if flags&FrameIsDroppable != 0 {
					droppable++
					t.Logf("frame %d: droppable", i)
				}
			}
		}
	}

	t.Logf("total: %d frames, %d keyframes, %d droppable", frameCount, keyframes, droppable)
}

// copyImageData copies YUV data from source to destination image.
func copyImageData(dst, src *Image) {
	h := int(src.DH)
	w := int(src.DW)

	srcYStride := int(src.Stride[PlaneY])
	srcUStride := int(src.Stride[PlaneU])
	srcVStride := int(src.Stride[PlaneV])

	dstYStride := int(dst.Stride[PlaneY])
	dstUStride := int(dst.Stride[PlaneU])
	dstVStride := int(dst.Stride[PlaneV])

	uvH := h / 2
	uvW := w / 2

	srcY := (*(*[1 << 30]byte)(unsafe.Pointer(src.Planes[PlaneY])))[:srcYStride*h]
	srcU := (*(*[1 << 30]byte)(unsafe.Pointer(src.Planes[PlaneU])))[:srcUStride*uvH]
	srcV := (*(*[1 << 30]byte)(unsafe.Pointer(src.Planes[PlaneV])))[:srcVStride*uvH]

	dstY := (*(*[1 << 30]byte)(unsafe.Pointer(dst.Planes[PlaneY])))[:dstYStride*h]
	dstU := (*(*[1 << 30]byte)(unsafe.Pointer(dst.Planes[PlaneU])))[:dstUStride*uvH]
	dstV := (*(*[1 << 30]byte)(unsafe.Pointer(dst.Planes[PlaneV])))[:dstVStride*uvH]

	// Copy row by row to handle different strides
	for row := 0; row < h; row++ {
		copy(dstY[row*dstYStride:row*dstYStride+w], srcY[row*srcYStride:row*srcYStride+w])
	}
	for row := 0; row < uvH; row++ {
		copy(dstU[row*dstUStride:row*dstUStride+uvW], srcU[row*srcUStride:row*srcUStride+uvW])
		copy(dstV[row*dstVStride:row*dstVStride+uvW], srcV[row*srcVStride:row*srcVStride+uvW])
	}
}
