package vpx

import (
	"image"
	"testing"
	"unsafe"
)

func TestImage_ImageRGBA(t *testing.T) {
	width := uint32(4)
	height := uint32(4)

	img := ImageAlloc(nil, ImageFormatI420, width, height, 1)
	if img == nil {
		t.Fatal("ImageAlloc returned nil")
	}
	defer ImageFree(img)
	img.Deref()

	ySize := int(img.Stride[PlaneY]) * int(height)
	uvHeight := int(height) / 2
	uSize := int(img.Stride[PlaneU]) * uvHeight
	vSize := int(img.Stride[PlaneV]) * uvHeight

	yData := make([]byte, ySize)
	uData := make([]byte, uSize)
	vData := make([]byte, vSize)

	for i := range yData {
		yData[i] = 128
	}
	for i := range uData {
		uData[i] = 128
	}
	for i := range vData {
		vData[i] = 128
	}

	copyToPlane(img.Planes[PlaneY], yData)
	copyToPlane(img.Planes[PlaneU], uData)
	copyToPlane(img.Planes[PlaneV], vData)

	rgba := img.ImageRGBA()
	if rgba == nil {
		t.Fatal("ImageRGBA returned nil")
	}

	expectedRect := image.Rect(0, 0, int(width), int(height))
	if rgba.Rect != expectedRect {
		t.Errorf("ImageRGBA rect = %v, want %v", rgba.Rect, expectedRect)
	}

	expectedPixLen := int(width) * int(height) * 4
	if len(rgba.Pix) != expectedPixLen {
		t.Errorf("ImageRGBA pix length = %d, want %d", len(rgba.Pix), expectedPixLen)
	}

	for i := 0; i < len(rgba.Pix); i += 4 {
		if rgba.Pix[i+3] != 255 {
			t.Errorf("Alpha channel at pixel %d = %d, want 255", i/4, rgba.Pix[i+3])
		}
	}
}

func TestImage_ImageYCbCr_I420(t *testing.T) {
	width := uint32(4)
	height := uint32(4)

	img := ImageAlloc(nil, ImageFormatI420, width, height, 1)
	if img == nil {
		t.Fatal("ImageAlloc returned nil")
	}
	defer ImageFree(img)
	img.Deref()

	ySize := int(img.Stride[PlaneY]) * int(height)
	uvHeight := int(height) / 2
	uSize := int(img.Stride[PlaneU]) * uvHeight
	vSize := int(img.Stride[PlaneV]) * uvHeight

	yData := make([]byte, ySize)
	uData := make([]byte, uSize)
	vData := make([]byte, vSize)

	for i := range yData {
		yData[i] = byte(i % 256)
	}
	for i := range uData {
		uData[i] = byte((i + 100) % 256)
	}
	for i := range vData {
		vData[i] = byte((i + 200) % 256)
	}

	copyToPlane(img.Planes[PlaneY], yData)
	copyToPlane(img.Planes[PlaneU], uData)
	copyToPlane(img.Planes[PlaneV], vData)

	ycbcr := img.ImageYCbCr()
	if ycbcr == nil {
		t.Fatal("ImageYCbCr returned nil")
	}

	expectedRect := image.Rect(0, 0, int(width), int(height))
	if ycbcr.Rect != expectedRect {
		t.Errorf("ImageYCbCr rect = %v, want %v", ycbcr.Rect, expectedRect)
	}

	if ycbcr.SubsampleRatio != image.YCbCrSubsampleRatio420 {
		t.Errorf("ImageYCbCr subsample ratio = %v, want YCbCrSubsampleRatio420", ycbcr.SubsampleRatio)
	}

	if ycbcr.YStride != int(img.Stride[PlaneY]) {
		t.Errorf("ImageYCbCr YStride = %d, want %d", ycbcr.YStride, img.Stride[PlaneY])
	}

	if ycbcr.CStride != int(img.Stride[PlaneU]) {
		t.Errorf("ImageYCbCr CStride = %d, want %d", ycbcr.CStride, img.Stride[PlaneU])
	}
}

func TestImage_ImageYCbCr_I422(t *testing.T) {
	width := uint32(4)
	height := uint32(4)

	img := ImageAlloc(nil, ImageFormatI422, width, height, 1)
	if img == nil {
		t.Fatal("ImageAlloc returned nil")
	}
	defer ImageFree(img)
	img.Deref()

	ycbcr := img.ImageYCbCr()
	if ycbcr == nil {
		t.Fatal("ImageYCbCr returned nil")
	}

	if ycbcr.SubsampleRatio != image.YCbCrSubsampleRatio422 {
		t.Errorf("ImageYCbCr subsample ratio = %v, want YCbCrSubsampleRatio422", ycbcr.SubsampleRatio)
	}
}

func TestImage_ImageYCbCr_I440(t *testing.T) {
	width := uint32(4)
	height := uint32(4)

	img := ImageAlloc(nil, ImageFormatI440, width, height, 1)
	if img == nil {
		t.Fatal("ImageAlloc returned nil")
	}
	defer ImageFree(img)
	img.Deref()

	ycbcr := img.ImageYCbCr()
	if ycbcr == nil {
		t.Fatal("ImageYCbCr returned nil")
	}

	if ycbcr.SubsampleRatio != image.YCbCrSubsampleRatio440 {
		t.Errorf("ImageYCbCr subsample ratio = %v, want YCbCrSubsampleRatio440", ycbcr.SubsampleRatio)
	}
}

func copyToPlane(plane *byte, data []byte) {
	if plane == nil || len(data) == 0 {
		return
	}
	dst := (*(*[1 << 30]byte)(unsafe.Pointer(plane)))[:len(data):len(data)]
	copy(dst, data)
}
