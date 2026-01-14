package vpx

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"os"
	"testing"
	"unsafe"
)

// writeIVF writes encoded VP8 packets to an IVF file.
// IVF is a simple container format for VP8/VP9.
func writeIVF(filename string, packets []EncodedPacket, width, height, frameRate int) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	// IVF Header (32 bytes)
	header := make([]byte, 32)
	copy(header[0:4], "DKIF")                                   // Signature
	binary.LittleEndian.PutUint16(header[4:6], 0)               // Version
	binary.LittleEndian.PutUint16(header[6:8], 32)              // Header size
	copy(header[8:12], "VP80")                                  // FourCC (VP8)
	binary.LittleEndian.PutUint16(header[12:14], uint16(width)) // Width
	binary.LittleEndian.PutUint16(header[14:16], uint16(height))// Height
	binary.LittleEndian.PutUint32(header[16:20], uint32(frameRate)) // Frame rate numerator
	binary.LittleEndian.PutUint32(header[20:24], 1)             // Frame rate denominator
	binary.LittleEndian.PutUint64(header[24:32], uint64(len(packets))) // Number of frames

	if _, err := f.Write(header); err != nil {
		return err
	}

	// Write each frame
	for i, pkt := range packets {
		// Frame header (12 bytes)
		frameHeader := make([]byte, 12)
		binary.LittleEndian.PutUint32(frameHeader[0:4], uint32(len(pkt.Data))) // Frame size
		binary.LittleEndian.PutUint64(frameHeader[4:12], uint64(i))            // Timestamp

		if _, err := f.Write(frameHeader); err != nil {
			return err
		}
		if _, err := f.Write(pkt.Data); err != nil {
			return err
		}
	}

	return nil
}

const (
	testWidth     = 320
	testHeight    = 240
	testFrameRate = 30
	testFrames    = 30
	testBitrate   = 200
)

// EncodedPacket represents an encoded video packet.
type EncodedPacket struct {
	Data       []byte
	Pts        CodecPts
	Duration   uint
	IsKeyframe bool
}

// generateYUVFrame generates a YUV420 frame with a gradient pattern.
// frameNum is used to create variation between frames.
func generateYUVFrame(width, height, frameNum int) (y, u, v []byte) {
	ySize := width * height
	uvWidth := width / 2
	uvHeight := height / 2
	uvSize := uvWidth * uvHeight

	y = make([]byte, ySize)
	u = make([]byte, uvSize)
	v = make([]byte, uvSize)

	for row := 0; row < height; row++ {
		for col := 0; col < width; col++ {
			offset := (frameNum * 8) % 256
			luma := (row + col + offset) % 256
			y[row*width+col] = byte(luma)
		}
	}

	for row := 0; row < uvHeight; row++ {
		for col := 0; col < uvWidth; col++ {
			offset := (frameNum * 4) % 256
			u[row*uvWidth+col] = byte((128 + row + offset) % 256)
			v[row*uvWidth+col] = byte((128 + col + offset) % 256)
		}
	}

	return y, u, v
}

// encodeFrames encodes YUV frames to VP8.
func encodeFrames(t *testing.T, frames int, width, height uint32) []EncodedPacket {
	t.Helper()

	encCtx := NewCodecCtx()
	defer CodecDestroy(encCtx)

	iface := EncoderIfaceVP8()
	if iface == nil {
		t.Fatal("EncoderIfaceVP8 returned nil")
	}

	cfg := &CodecEncCfg{}
	err := Error(CodecEncConfigDefault(iface, cfg, 0))
	if err != nil {
		t.Fatalf("CodecEncConfigDefault failed: %v", err)
	}
	cfg.Deref()

	cfg.GW = width
	cfg.GH = height
	cfg.GTimebase = Rational{Num: 1, Den: testFrameRate}
	cfg.RcTargetBitrate = testBitrate
	cfg.GPass = RcOnePass
	cfg.RcEndUsage = Vbr
	cfg.KfMode = KfAuto
	cfg.KfMaxDist = 30
	cfg.GThreads = 1

	err = Error(CodecEncInitVer(encCtx, iface, cfg, 0, EncoderABIVersion))
	if err != nil {
		t.Fatalf("CodecEncInitVer failed: %v", err)
	}

	var packets []EncodedPacket

	for i := 0; i < frames; i++ {
		y, u, v := generateYUVFrame(int(width), int(height), i)

		img := ImageAlloc(nil, ImageFormatI420, width, height, 1)
		if img == nil {
			t.Fatal("ImageAlloc returned nil")
		}
		img.Deref()

		yDst := (*(*[1 << 30]byte)(unsafe.Pointer(img.Planes[PlaneY])))[:len(y):len(y)]
		copy(yDst, y)
		uDst := (*(*[1 << 30]byte)(unsafe.Pointer(img.Planes[PlaneU])))[:len(u):len(u)]
		copy(uDst, u)
		vDst := (*(*[1 << 30]byte)(unsafe.Pointer(img.Planes[PlaneV])))[:len(v):len(v)]
		copy(vDst, v)

		pts := CodecPts(i)
		err = Error(CodecEncode(encCtx, img, pts, 1, 0, DlGoodQuality))
		if err != nil {
			ImageFree(img)
			t.Fatalf("CodecEncode failed at frame %d: %v", i, err)
		}

		var iter CodecIter
		for pkt := CodecGetCxData(encCtx, &iter); pkt != nil; pkt = CodecGetCxData(encCtx, &iter) {
			pkt.Deref()
			if pkt.Kind == CodecCxFramePkt {
				data := pkt.GetFrameData()
				if len(data) > 0 {
					dataCopy := make([]byte, len(data))
					copy(dataCopy, data)
					packets = append(packets, EncodedPacket{
						Data:       dataCopy,
						Pts:        pkt.GetFramePts(),
						Duration:   pkt.GetFrameDuration(),
						IsKeyframe: pkt.IsKeyframe(),
					})
				}
			}
		}

		ImageFree(img)
	}

	err = Error(CodecEncode(encCtx, nil, 0, 0, 0, DlGoodQuality))
	if err != nil {
		t.Fatalf("CodecEncode flush failed: %v", err)
	}

	var iter CodecIter
	for pkt := CodecGetCxData(encCtx, &iter); pkt != nil; pkt = CodecGetCxData(encCtx, &iter) {
		pkt.Deref()
		if pkt.Kind == CodecCxFramePkt {
			data := pkt.GetFrameData()
			if len(data) > 0 {
				dataCopy := make([]byte, len(data))
				copy(dataCopy, data)
				packets = append(packets, EncodedPacket{
					Data:       dataCopy,
					Pts:        pkt.GetFramePts(),
					Duration:   pkt.GetFrameDuration(),
					IsKeyframe: pkt.IsKeyframe(),
				})
			}
		}
	}

	return packets
}

// decodePackets decodes VP8 packets and returns decoded frame count.
func decodePackets(t *testing.T, packets []EncodedPacket) []*Image {
	t.Helper()

	decCtx := NewCodecCtx()
	defer CodecDestroy(decCtx)

	iface := DecoderIfaceVP8()
	if iface == nil {
		t.Fatal("DecoderIfaceVP8 returned nil")
	}

	err := Error(CodecDecInitVer(decCtx, iface, nil, 0, DecoderABIVersion))
	if err != nil {
		t.Fatalf("CodecDecInitVer failed: %v", err)
	}

	var decodedFrames []*Image

	for i, pkt := range packets {
		err = Error(CodecDecode(decCtx, string(pkt.Data), uint32(len(pkt.Data)), nil, 0))
		if err != nil {
			t.Fatalf("CodecDecode failed at packet %d: %v", i, err)
		}

		var iter CodecIter
		for img := CodecGetFrame(decCtx, &iter); img != nil; img = CodecGetFrame(decCtx, &iter) {
			img.Deref()

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

			ySize := int(img.Stride[PlaneY]) * int(img.DH)
			uvH := int(img.DH) / 2
			uSize := int(img.Stride[PlaneU]) * uvH
			vSize := int(img.Stride[PlaneV]) * uvH

			ySrc := (*(*[1 << 30]byte)(unsafe.Pointer(img.Planes[PlaneY])))[:ySize:ySize]
			uSrc := (*(*[1 << 30]byte)(unsafe.Pointer(img.Planes[PlaneU])))[:uSize:uSize]
			vSrc := (*(*[1 << 30]byte)(unsafe.Pointer(img.Planes[PlaneV])))[:vSize:vSize]

			imgCopy.ImgData = make([]byte, ySize+uSize+vSize)
			copy(imgCopy.ImgData[0:ySize], ySrc)
			copy(imgCopy.ImgData[ySize:ySize+uSize], uSrc)
			copy(imgCopy.ImgData[ySize+uSize:], vSrc)

			imgCopy.Planes[PlaneY] = &imgCopy.ImgData[0]
			imgCopy.Planes[PlaneU] = &imgCopy.ImgData[ySize]
			imgCopy.Planes[PlaneV] = &imgCopy.ImgData[ySize+uSize]

			decodedFrames = append(decodedFrames, imgCopy)
		}
	}

	return decodedFrames
}

// reencodeFrames re-encodes decoded frames to VP8.
func reencodeFrames(t *testing.T, frames []*Image) []EncodedPacket {
	t.Helper()

	if len(frames) == 0 {
		return nil
	}

	encCtx := NewCodecCtx()
	defer CodecDestroy(encCtx)

	iface := EncoderIfaceVP8()
	if iface == nil {
		t.Fatal("EncoderIfaceVP8 returned nil")
	}

	cfg := &CodecEncCfg{}
	err := Error(CodecEncConfigDefault(iface, cfg, 0))
	if err != nil {
		t.Fatalf("CodecEncConfigDefault failed: %v", err)
	}
	cfg.Deref()

	width := frames[0].DW
	height := frames[0].DH

	cfg.GW = width
	cfg.GH = height
	cfg.GTimebase = Rational{Num: 1, Den: testFrameRate}
	cfg.RcTargetBitrate = testBitrate
	cfg.GPass = RcOnePass
	cfg.RcEndUsage = Vbr
	cfg.KfMode = KfAuto
	cfg.KfMaxDist = 30
	cfg.GThreads = 1

	err = Error(CodecEncInitVer(encCtx, iface, cfg, 0, EncoderABIVersion))
	if err != nil {
		t.Fatalf("CodecEncInitVer failed: %v", err)
	}

	var packets []EncodedPacket

	for i, frame := range frames {
		img := ImageAlloc(nil, ImageFormatI420, width, height, 1)
		if img == nil {
			t.Fatal("ImageAlloc returned nil")
		}
		img.Deref()

		// Copy Y plane row by row (handle different strides)
		srcYStride := int(frame.Stride[PlaneY])
		dstYStride := int(img.Stride[PlaneY])
		srcUStride := int(frame.Stride[PlaneU])
		dstUStride := int(img.Stride[PlaneU])
		srcVStride := int(frame.Stride[PlaneV])
		dstVStride := int(img.Stride[PlaneV])

		h := int(height)
		uvH := h / 2
		w := int(width)
		uvW := w / 2

		// Source data offsets in frame.ImgData
		srcYOffset := 0
		srcUOffset := srcYStride * h
		srcVOffset := srcUOffset + srcUStride * uvH

		dstY := (*(*[1 << 30]byte)(unsafe.Pointer(img.Planes[PlaneY])))[:dstYStride*h:dstYStride*h]
		dstU := (*(*[1 << 30]byte)(unsafe.Pointer(img.Planes[PlaneU])))[:dstUStride*uvH:dstUStride*uvH]
		dstV := (*(*[1 << 30]byte)(unsafe.Pointer(img.Planes[PlaneV])))[:dstVStride*uvH:dstVStride*uvH]

		// Copy Y plane row by row
		for row := 0; row < h; row++ {
			srcStart := srcYOffset + row*srcYStride
			dstStart := row * dstYStride
			copy(dstY[dstStart:dstStart+w], frame.ImgData[srcStart:srcStart+w])
		}

		// Copy U plane row by row
		for row := 0; row < uvH; row++ {
			srcStart := srcUOffset + row*srcUStride
			dstStart := row * dstUStride
			copy(dstU[dstStart:dstStart+uvW], frame.ImgData[srcStart:srcStart+uvW])
		}

		// Copy V plane row by row
		for row := 0; row < uvH; row++ {
			srcStart := srcVOffset + row*srcVStride
			dstStart := row * dstVStride
			copy(dstV[dstStart:dstStart+uvW], frame.ImgData[srcStart:srcStart+uvW])
		}

		pts := CodecPts(i)
		err = Error(CodecEncode(encCtx, img, pts, 1, 0, DlGoodQuality))
		if err != nil {
			ImageFree(img)
			t.Fatalf("CodecEncode failed at frame %d: %v", i, err)
		}

		var iter CodecIter
		for pkt := CodecGetCxData(encCtx, &iter); pkt != nil; pkt = CodecGetCxData(encCtx, &iter) {
			pkt.Deref()
			if pkt.Kind == CodecCxFramePkt {
				data := pkt.GetFrameData()
				if len(data) > 0 {
					dataCopy := make([]byte, len(data))
					copy(dataCopy, data)
					packets = append(packets, EncodedPacket{
						Data:       dataCopy,
						Pts:        pkt.GetFramePts(),
						Duration:   pkt.GetFrameDuration(),
						IsKeyframe: pkt.IsKeyframe(),
					})
				}
			}
		}

		ImageFree(img)
	}

	err = Error(CodecEncode(encCtx, nil, 0, 0, 0, DlGoodQuality))
	if err != nil {
		t.Fatalf("CodecEncode flush failed: %v", err)
	}

	var iter CodecIter
	for pkt := CodecGetCxData(encCtx, &iter); pkt != nil; pkt = CodecGetCxData(encCtx, &iter) {
		pkt.Deref()
		if pkt.Kind == CodecCxFramePkt {
			data := pkt.GetFrameData()
			if len(data) > 0 {
				dataCopy := make([]byte, len(data))
				copy(dataCopy, data)
				packets = append(packets, EncodedPacket{
					Data:       dataCopy,
					Pts:        pkt.GetFramePts(),
					Duration:   pkt.GetFrameDuration(),
					IsKeyframe: pkt.IsKeyframe(),
				})
			}
		}
	}

	return packets
}

// TestVP8Transcode tests the full VP8 transcode pipeline:
// Generate YUV frames -> Encode to VP8 -> Decode -> Re-encode to VP8
func TestVP8Transcode(t *testing.T) {
	t.Log("Step 1: Encoding generated YUV frames to VP8")
	encodedPackets := encodeFrames(t, testFrames, testWidth, testHeight)
	if len(encodedPackets) == 0 {
		t.Fatal("No packets encoded")
	}
	t.Logf("Encoded %d packets", len(encodedPackets))

	keyframeCount := 0
	for _, pkt := range encodedPackets {
		if pkt.IsKeyframe {
			keyframeCount++
		}
	}
	t.Logf("Keyframes: %d", keyframeCount)
	if keyframeCount == 0 {
		t.Error("Expected at least one keyframe")
	}

	t.Log("Step 2: Decoding VP8 packets")
	decodedFrames := decodePackets(t, encodedPackets)
	if len(decodedFrames) == 0 {
		t.Fatal("No frames decoded")
	}
	t.Logf("Decoded %d frames", len(decodedFrames))

	if len(decodedFrames) != testFrames {
		t.Errorf("Expected %d frames, got %d", testFrames, len(decodedFrames))
	}

	t.Log("Step 3: Re-encoding decoded frames to VP8")
	reEncodedPackets := reencodeFrames(t, decodedFrames)
	if len(reEncodedPackets) == 0 {
		t.Fatal("No packets re-encoded")
	}
	t.Logf("Re-encoded %d packets", len(reEncodedPackets))

	t.Log("Step 4: Verifying re-encoded data by decoding again")
	finalFrames := decodePackets(t, reEncodedPackets)
	if len(finalFrames) == 0 {
		t.Fatal("No frames decoded from re-encoded data")
	}
	t.Logf("Final decoded %d frames", len(finalFrames))

	if len(finalFrames) != testFrames {
		t.Errorf("Expected %d final frames, got %d", testFrames, len(finalFrames))
	}

	t.Log("Step 5: Saving IVF files")
	originalFile := "original.ivf"
	reEncodedFile := "reencoded.ivf"

	if err := writeIVF(originalFile, encodedPackets, testWidth, testHeight, testFrameRate); err != nil {
		t.Fatalf("Failed to write original IVF: %v", err)
	}
	t.Logf("Saved original video to %s", originalFile)

	if err := writeIVF(reEncodedFile, reEncodedPackets, testWidth, testHeight, testFrameRate); err != nil {
		t.Fatalf("Failed to write re-encoded IVF: %v", err)
	}
	t.Logf("Saved re-encoded video to %s", reEncodedFile)

	t.Log("VP8 transcode test completed successfully")
}

// TestExtractFrames extracts frames from original and re-encoded videos for comparison.
func TestExtractFrames(t *testing.T) {
	// Create output directory
	os.MkdirAll("frames", 0755)

	t.Log("Step 1: Encoding generated YUV frames to VP8")
	encodedPackets := encodeFrames(t, testFrames, testWidth, testHeight)

	t.Log("Step 2: Decoding original VP8 packets")
	originalFrames := decodePackets(t, encodedPackets)

	t.Log("Step 3: Re-encoding decoded frames to VP8")
	reEncodedPackets := reencodeFrames(t, originalFrames)

	t.Log("Step 4: Decoding re-encoded VP8 packets")
	reEncodedFrames := decodePackets(t, reEncodedPackets)

	t.Log("Step 5: Saving frames as PNG")

	// Save first, middle, and last frames from both
	framesToSave := []int{0, 14, 29}

	for _, frameIdx := range framesToSave {
		// Save original frame
		if frameIdx < len(originalFrames) {
			img := originalFrames[frameIdx]
			rgba := imageToRGBA(img)
			filename := fmt.Sprintf("frames/original_frame_%02d.png", frameIdx)
			saveRGBA(t, filename, rgba)
			t.Logf("Saved %s", filename)
		}

		// Save re-encoded frame
		if frameIdx < len(reEncodedFrames) {
			img := reEncodedFrames[frameIdx]
			rgba := imageToRGBA(img)
			filename := fmt.Sprintf("frames/reencoded_frame_%02d.png", frameIdx)
			saveRGBA(t, filename, rgba)
			t.Logf("Saved %s", filename)
		}
	}

	// Also save the generated YUV frame for reference
	for _, frameIdx := range framesToSave {
		y, u, v := generateYUVFrame(testWidth, testHeight, frameIdx)
		rgba := yuvToRGBA(y, u, v, testWidth, testHeight)
		filename := fmt.Sprintf("frames/generated_frame_%02d.png", frameIdx)
		saveRGBA(t, filename, rgba)
		t.Logf("Saved %s", filename)
	}

	t.Log("Frame extraction completed")
}

// imageToRGBA converts vpx.Image to image.RGBA
func imageToRGBA(img *Image) *image.RGBA {
	width := int(img.DW)
	height := int(img.DH)
	yStride := int(img.Stride[PlaneY])
	uStride := int(img.Stride[PlaneU])
	vStride := int(img.Stride[PlaneV])

	rgba := image.NewRGBA(image.Rect(0, 0, width, height))

	for row := 0; row < height; row++ {
		for col := 0; col < width; col++ {
			yIdx := row*yStride + col
			uvRow := row / 2
			uvCol := col / 2
			uIdx := uvRow*uStride + uvCol
			vIdx := uvRow*vStride + uvCol

			y := int(img.ImgData[yIdx])
			u := int(img.ImgData[yStride*height+uIdx])
			v := int(img.ImgData[yStride*height+uStride*(height/2)+vIdx])

			r, g, b := yuvToRGB(y, u, v)
			rgba.Pix[(row*width+col)*4+0] = r
			rgba.Pix[(row*width+col)*4+1] = g
			rgba.Pix[(row*width+col)*4+2] = b
			rgba.Pix[(row*width+col)*4+3] = 255
		}
	}

	return rgba
}

// yuvToRGBA converts raw YUV data to image.RGBA
func yuvToRGBA(y, u, v []byte, width, height int) *image.RGBA {
	rgba := image.NewRGBA(image.Rect(0, 0, width, height))
	uvWidth := width / 2

	for row := 0; row < height; row++ {
		for col := 0; col < width; col++ {
			yIdx := row*width + col
			uvRow := row / 2
			uvCol := col / 2
			uvIdx := uvRow*uvWidth + uvCol

			yVal := int(y[yIdx])
			uVal := int(u[uvIdx])
			vVal := int(v[uvIdx])

			r, g, b := yuvToRGB(yVal, uVal, vVal)
			rgba.Pix[(row*width+col)*4+0] = r
			rgba.Pix[(row*width+col)*4+1] = g
			rgba.Pix[(row*width+col)*4+2] = b
			rgba.Pix[(row*width+col)*4+3] = 255
		}
	}

	return rgba
}

// yuvToRGB converts YUV to RGB
func yuvToRGB(y, u, v int) (r, g, b uint8) {
	// BT.601 conversion
	y = y - 16
	if y < 0 {
		y = 0
	}
	u = u - 128
	v = v - 128

	r1 := (298*y + 409*v + 128) >> 8
	g1 := (298*y - 100*u - 208*v + 128) >> 8
	b1 := (298*y + 516*u + 128) >> 8

	if r1 < 0 {
		r1 = 0
	} else if r1 > 255 {
		r1 = 255
	}
	if g1 < 0 {
		g1 = 0
	} else if g1 > 255 {
		g1 = 255
	}
	if b1 < 0 {
		b1 = 0
	} else if b1 > 255 {
		b1 = 255
	}

	return uint8(r1), uint8(g1), uint8(b1)
}

// saveRGBA saves image.RGBA to a PNG file
func saveRGBA(t *testing.T, filename string, img *image.RGBA) {
	t.Helper()
	f, err := os.Create(filename)
	if err != nil {
		t.Fatalf("Failed to create %s: %v", filename, err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		t.Fatalf("Failed to encode PNG: %v", err)
	}
}

// TestVP8Encode tests VP8 encoding only.
func TestVP8Encode(t *testing.T) {
	packets := encodeFrames(t, 10, testWidth, testHeight)
	if len(packets) == 0 {
		t.Fatal("No packets encoded")
	}
	t.Logf("Encoded %d packets", len(packets))

	if !packets[0].IsKeyframe {
		t.Error("First frame should be a keyframe")
	}
}

// TestVP8Decode tests VP8 decoding only.
func TestVP8Decode(t *testing.T) {
	packets := encodeFrames(t, 10, testWidth, testHeight)
	if len(packets) == 0 {
		t.Fatal("No packets to decode")
	}

	frames := decodePackets(t, packets)
	if len(frames) != 10 {
		t.Errorf("Expected 10 frames, got %d", len(frames))
	}
}

// TestCodecVersion tests that the codec version is available.
func TestCodecVersion(t *testing.T) {
	version := CodecVersion()
	versionStr := CodecVersionStr()
	t.Logf("libvpx version: %d (%s)", version, versionStr)

	if version == 0 {
		t.Error("CodecVersion returned 0")
	}
	if versionStr == "" {
		t.Error("CodecVersionStr returned empty string")
	}
}
