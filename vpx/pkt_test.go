package vpx

import (
	"testing"
)

func TestCodecCxPkt_GetFrameData_NilPacket(t *testing.T) {
	var pkt *CodecCxPkt
	data := pkt.GetFrameData()
	if data != nil {
		t.Errorf("GetFrameData() on nil packet = %v, want nil", data)
	}
}

func TestCodecCxPkt_GetFrameData_NilRef(t *testing.T) {
	pkt := &CodecCxPkt{
		refa671fc83: nil,
	}
	data := pkt.GetFrameData()
	if data != nil {
		t.Errorf("GetFrameData() on packet with nil ref = %v, want nil", data)
	}
}

func TestCodecCxPkt_GetFramePts_NilPacket(t *testing.T) {
	var pkt *CodecCxPkt
	pts := pkt.GetFramePts()
	if pts != 0 {
		t.Errorf("GetFramePts() on nil packet = %v, want 0", pts)
	}
}

func TestCodecCxPkt_GetFramePts_NilRef(t *testing.T) {
	pkt := &CodecCxPkt{
		refa671fc83: nil,
	}
	pts := pkt.GetFramePts()
	if pts != 0 {
		t.Errorf("GetFramePts() on packet with nil ref = %v, want 0", pts)
	}
}

func TestCodecCxPkt_GetFrameDuration_NilPacket(t *testing.T) {
	var pkt *CodecCxPkt
	duration := pkt.GetFrameDuration()
	if duration != 0 {
		t.Errorf("GetFrameDuration() on nil packet = %v, want 0", duration)
	}
}

func TestCodecCxPkt_GetFrameDuration_NilRef(t *testing.T) {
	pkt := &CodecCxPkt{
		refa671fc83: nil,
	}
	duration := pkt.GetFrameDuration()
	if duration != 0 {
		t.Errorf("GetFrameDuration() on packet with nil ref = %v, want 0", duration)
	}
}

func TestCodecCxPkt_GetFrameFlags_NilPacket(t *testing.T) {
	var pkt *CodecCxPkt
	flags := pkt.GetFrameFlags()
	if flags != 0 {
		t.Errorf("GetFrameFlags() on nil packet = %v, want 0", flags)
	}
}

func TestCodecCxPkt_GetFrameFlags_NilRef(t *testing.T) {
	pkt := &CodecCxPkt{
		refa671fc83: nil,
	}
	flags := pkt.GetFrameFlags()
	if flags != 0 {
		t.Errorf("GetFrameFlags() on packet with nil ref = %v, want 0", flags)
	}
}

func TestCodecCxPkt_IsKeyframe_NilPacket(t *testing.T) {
	var pkt *CodecCxPkt
	if pkt.IsKeyframe() {
		t.Error("IsKeyframe() on nil packet should return false")
	}
}

func TestCodecCxPkt_IsKeyframe_NilRef(t *testing.T) {
	pkt := &CodecCxPkt{
		refa671fc83: nil,
	}
	if pkt.IsKeyframe() {
		t.Error("IsKeyframe() on packet with nil ref should return false")
	}
}

func TestImage_SetImageData_NilImage(t *testing.T) {
	var img *Image
	img.SetImageData([]byte{1, 2, 3}, []byte{4, 5}, []byte{6, 7})
}

func TestImage_SetImageData(t *testing.T) {
	img := &Image{}
	y := []byte{1, 2, 3, 4}
	u := []byte{5, 6}
	v := []byte{7, 8}

	img.SetImageData(y, u, v)

	if img.Planes[PlaneY] != &y[0] {
		t.Error("SetImageData did not set Y plane correctly")
	}
	if img.Planes[PlaneU] != &u[0] {
		t.Error("SetImageData did not set U plane correctly")
	}
	if img.Planes[PlaneV] != &v[0] {
		t.Error("SetImageData did not set V plane correctly")
	}
}

func TestImage_SetImageData_EmptySlices(t *testing.T) {
	img := &Image{}
	img.SetImageData([]byte{}, []byte{}, []byte{})

	if img.Planes[PlaneY] != nil {
		t.Error("SetImageData with empty Y slice should not set pointer")
	}
	if img.Planes[PlaneU] != nil {
		t.Error("SetImageData with empty U slice should not set pointer")
	}
	if img.Planes[PlaneV] != nil {
		t.Error("SetImageData with empty V slice should not set pointer")
	}
}

func TestImage_GetYUVData_NilImage(t *testing.T) {
	var img *Image
	y, u, v := img.GetYUVData()
	if y != nil || u != nil || v != nil {
		t.Error("GetYUVData() on nil image should return nil slices")
	}
}
