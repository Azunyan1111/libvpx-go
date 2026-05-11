package vpx

import "testing"

func TestVP8EncoderControls(t *testing.T) {
	ctx := newInitializedEncoder(t, EncoderIfaceVP8())
	defer CodecDestroy(ctx)

	if err := Error(CodecControlInt(ctx, VP8ESetCPUUsed, 6)); err != nil {
		t.Fatalf("VP8ESetCPUUsed failed: %v", err)
	}
	if err := Error(CodecControlUint(ctx, VP8ESetStaticThreshold, 0)); err != nil {
		t.Fatalf("VP8ESetStaticThreshold failed: %v", err)
	}
	if err := Error(CodecControlUint(ctx, VP8ESetMaxIntraBitratePct, 300)); err != nil {
		t.Fatalf("VP8ESetMaxIntraBitratePct failed: %v", err)
	}
}

func TestVP9EncoderControls(t *testing.T) {
	ctx := newInitializedEncoder(t, EncoderIfaceVP9())
	defer CodecDestroy(ctx)

	if err := Error(CodecControlInt(ctx, VP9ESetTileColumns, 2)); err != nil {
		t.Fatalf("VP9ESetTileColumns failed: %v", err)
	}
	if err := Error(CodecControlUint(ctx, VP9ESetFrameParallelDecoding, 1)); err != nil {
		t.Fatalf("VP9ESetFrameParallelDecoding failed: %v", err)
	}
	if err := Error(CodecControlUint(ctx, VP9ESetRowMT, 1)); err != nil {
		t.Fatalf("VP9ESetRowMT failed: %v", err)
	}
}

func TestCodecControlInvalidID(t *testing.T) {
	ctx := newInitializedEncoder(t, EncoderIfaceVP8())
	defer CodecDestroy(ctx)

	if err := CodecControlInt(ctx, -1, 0); err == CodecOk {
		t.Fatal("expected invalid control ID to fail")
	}
}

func newInitializedEncoder(t *testing.T, iface *CodecIface) *CodecCtx {
	t.Helper()

	if iface == nil {
		t.Fatal("failed to get encoder interface")
	}

	ctx := NewCodecCtx()
	if ctx == nil {
		t.Fatal("failed to create codec context")
	}

	cfg := &CodecEncCfg{}
	if err := Error(CodecEncConfigDefault(iface, cfg, 0)); err != nil {
		CodecDestroy(ctx)
		t.Fatalf("failed to get default encoder config: %v", err)
	}
	cfg.Deref()

	cfg.GW = 320
	cfg.GH = 240
	cfg.GTimebase = Rational{Num: 1, Den: 30}
	cfg.RcTargetBitrate = 200
	cfg.GPass = RcOnePass
	cfg.GLagInFrames = 0

	if err := Error(CodecEncInitVer(ctx, iface, cfg, 0, EncoderABIVersion)); err != nil {
		CodecDestroy(ctx)
		t.Fatalf("failed to initialize encoder: %v", err)
	}

	return ctx
}
