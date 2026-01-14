package vpx

import (
	"testing"
)

func TestError(t *testing.T) {
	tests := []struct {
		name     string
		input    CodecErr
		wantErr  error
		wantNil  bool
	}{
		{
			name:    "CodecOk returns nil",
			input:   CodecOk,
			wantErr: nil,
			wantNil: true,
		},
		{
			name:    "CodecError returns ErrCodecUnknownError",
			input:   CodecError,
			wantErr: ErrCodecUnknownError,
			wantNil: false,
		},
		{
			name:    "CodecMemError returns ErrCodecMemError",
			input:   CodecMemError,
			wantErr: ErrCodecMemError,
			wantNil: false,
		},
		{
			name:    "CodecABIMismatch returns ErrCodecABIMismatch",
			input:   CodecABIMismatch,
			wantErr: ErrCodecABIMismatch,
			wantNil: false,
		},
		{
			name:    "CodecIncapable returns ErrCodecIncapable",
			input:   CodecIncapable,
			wantErr: ErrCodecIncapable,
			wantNil: false,
		},
		{
			name:    "CodecUnsupBitstream returns ErrCodecUnsupBitstream",
			input:   CodecUnsupBitstream,
			wantErr: ErrCodecUnsupBitstream,
			wantNil: false,
		},
		{
			name:    "CodecUnsupFeature returns ErrCodecUnsupFeature",
			input:   CodecUnsupFeature,
			wantErr: ErrCodecUnsupFeature,
			wantNil: false,
		},
		{
			name:    "CodecCorruptFrame returns ErrCodecCorruptFrame",
			input:   CodecCorruptFrame,
			wantErr: ErrCodecCorruptFrame,
			wantNil: false,
		},
		{
			name:    "CodecInvalidParam returns ErrCodecInvalidParam",
			input:   CodecInvalidParam,
			wantErr: ErrCodecInvalidParam,
			wantNil: false,
		},
		{
			name:    "Unknown error code returns ErrCodecUnknownError",
			input:   CodecErr(9999),
			wantErr: ErrCodecUnknownError,
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Error(tt.input)
			if tt.wantNil {
				if got != nil {
					t.Errorf("Error(%v) = %v, want nil", tt.input, got)
				}
			} else {
				if got != tt.wantErr {
					t.Errorf("Error(%v) = %v, want %v", tt.input, got, tt.wantErr)
				}
			}
		})
	}
}

func TestErrorMessages(t *testing.T) {
	tests := []struct {
		err     error
		wantMsg string
	}{
		{ErrCodecUnknownError, "vpx: unknown error"},
		{ErrCodecMemError, "vpx: memory error"},
		{ErrCodecABIMismatch, "vpx: ABI mismatch"},
		{ErrCodecIncapable, "vpx: incapable"},
		{ErrCodecUnsupBitstream, "vpx: unsupported bitstream"},
		{ErrCodecUnsupFeature, "vpx: unsupported feature"},
		{ErrCodecCorruptFrame, "vpx: corrupt frame"},
		{ErrCodecInvalidParam, "vpx: invalid param"},
	}

	for _, tt := range tests {
		t.Run(tt.wantMsg, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.wantMsg {
				t.Errorf("error message = %q, want %q", got, tt.wantMsg)
			}
		})
	}
}
