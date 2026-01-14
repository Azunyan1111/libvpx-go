package vpx

import (
	"testing"
)

func TestDecoderIfaceVP8(t *testing.T) {
	iface := DecoderIfaceVP8()
	if iface == nil {
		t.Error("DecoderIfaceVP8() returned nil")
	}
}

func TestDecoderIfaceVP9(t *testing.T) {
	iface := DecoderIfaceVP9()
	if iface == nil {
		t.Error("DecoderIfaceVP9() returned nil")
	}
}

func TestEncoderIfaceVP8(t *testing.T) {
	iface := EncoderIfaceVP8()
	if iface == nil {
		t.Error("EncoderIfaceVP8() returned nil")
	}
}

func TestEncoderIfaceVP9(t *testing.T) {
	iface := EncoderIfaceVP9()
	if iface == nil {
		t.Error("EncoderIfaceVP9() returned nil")
	}
}

func TestDecoderFor(t *testing.T) {
	tests := []struct {
		name    string
		fourcc  int
		wantNil bool
	}{
		{
			name:    "VP8 fourcc returns VP8 decoder",
			fourcc:  Vp8Fourcc,
			wantNil: false,
		},
		{
			name:    "VP9 fourcc returns VP9 decoder",
			fourcc:  Vp9Fourcc,
			wantNil: false,
		},
		{
			name:    "Unknown fourcc returns nil",
			fourcc:  0,
			wantNil: true,
		},
		{
			name:    "Invalid fourcc returns nil",
			fourcc:  12345,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DecoderFor(tt.fourcc)
			if tt.wantNil {
				if got != nil {
					t.Errorf("DecoderFor(%d) = %v, want nil", tt.fourcc, got)
				}
			} else {
				if got == nil {
					t.Errorf("DecoderFor(%d) = nil, want non-nil", tt.fourcc)
				}
			}
		})
	}
}

func TestEncoderFor(t *testing.T) {
	tests := []struct {
		name    string
		fourcc  int
		wantNil bool
	}{
		{
			name:    "VP8 fourcc returns VP8 encoder",
			fourcc:  Vp8Fourcc,
			wantNil: false,
		},
		{
			name:    "VP9 fourcc returns VP9 encoder",
			fourcc:  Vp9Fourcc,
			wantNil: false,
		},
		{
			name:    "Unknown fourcc returns nil",
			fourcc:  0,
			wantNil: true,
		},
		{
			name:    "Invalid fourcc returns nil",
			fourcc:  12345,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EncoderFor(tt.fourcc)
			if tt.wantNil {
				if got != nil {
					t.Errorf("EncoderFor(%d) = %v, want nil", tt.fourcc, got)
				}
			} else {
				if got == nil {
					t.Errorf("EncoderFor(%d) = nil, want non-nil", tt.fourcc)
				}
			}
		})
	}
}

func TestDecoderForReturnsCorrectInterface(t *testing.T) {
	vp8Dec := DecoderFor(Vp8Fourcc)
	vp8Expected := DecoderIfaceVP8()
	if vp8Dec != vp8Expected {
		t.Error("DecoderFor(Vp8Fourcc) did not return the same interface as DecoderIfaceVP8()")
	}

	vp9Dec := DecoderFor(Vp9Fourcc)
	vp9Expected := DecoderIfaceVP9()
	if vp9Dec != vp9Expected {
		t.Error("DecoderFor(Vp9Fourcc) did not return the same interface as DecoderIfaceVP9()")
	}
}

func TestEncoderForReturnsCorrectInterface(t *testing.T) {
	vp8Enc := EncoderFor(Vp8Fourcc)
	vp8Expected := EncoderIfaceVP8()
	if vp8Enc != vp8Expected {
		t.Error("EncoderFor(Vp8Fourcc) did not return the same interface as EncoderIfaceVP8()")
	}

	vp9Enc := EncoderFor(Vp9Fourcc)
	vp9Expected := EncoderIfaceVP9()
	if vp9Enc != vp9Expected {
		t.Error("EncoderFor(Vp9Fourcc) did not return the same interface as EncoderIfaceVP9()")
	}
}
