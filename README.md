# libvpx-go

[![Go Reference](https://pkg.go.dev/badge/github.com/Azunyan1111/libvpx-go/vpx.svg)](https://pkg.go.dev/github.com/Azunyan1111/libvpx-go/vpx)

Go bindings for [libvpx-1.15.2](http://www.webmproject.org/code/), the WebM Project VP8/VP9 codec implementation.

This is a fork of [xlab/libvpx-go](https://github.com/xlab/libvpx-go) with the following improvements:

- **Static linking** - libvpx 1.15.2 static library is bundled, no system installation required
- **Multi-platform support** - macOS arm64 and Linux amd64 static libraries included
- **Go Modules support** - Modern `go.mod` included
- **Zero dependencies** - Just clone and build

## Supported Platforms

| Platform | Architecture | Status |
|----------|--------------|--------|
| macOS    | arm64        | Supported |
| Linux    | amd64        | Supported |

## Installation

```bash
go get github.com/Azunyan1111/libvpx-go/vpx
```

No need to install libvpx separately. The static library is included in this repository.

## Usage

```go
import "github.com/Azunyan1111/libvpx-go/vpx"

// Initialize VP9 decoder
ctx := vpx.NewCodecCtx()
iface := vpx.DecoderIfaceVP9()
err := vpx.Error(vpx.CodecDecInitVer(ctx, iface, nil, 0, vpx.DecoderABIVersion))

// Decode frame
vpx.CodecDecode(ctx, data, dataSize, nil, 0)

// Get decoded frame
var iter vpx.CodecIter
img := vpx.CodecGetFrame(ctx, &iter)
if img != nil {
    img.Deref()
    rgba := img.ImageRGBA() // Convert to Go image.RGBA
}
```

## Bundled Library

This repository includes:

- `lib/darwin_arm64/libvpx.a` - libvpx 1.15.2 static library for macOS arm64
- `lib/linux_amd64/libvpx.a` - libvpx 1.15.2 static library for Linux amd64
- `include/vpx/` - libvpx header files
- `lib/LICENSE.libvpx` - libvpx license file (BSD 3-Clause License)

## Demo Application

A simple WebM player with VP8/VP9 video and Vorbis/Opus audio support is included in [cmd/webm-player](cmd/webm-player).

### Additional dependencies for demo app

```bash
# macOS
brew install libogg libvorbis opus portaudio

# Linux (Debian/Ubuntu)
apt-get install libogg-dev libvorbis-dev libopus-dev portaudio19-dev
```

### Run the demo

```bash
go run ./cmd/webm-player your_video.webm
```

## Development

### Rebuilding the bindings

If you need to regenerate the bindings, install [c-for-go](https://git.io/c-for-go) first:

```bash
make clean
make
```

### Testing

```bash
# Run tests on host
make test

# Run tests on Linux (Docker required)
make test-linux
```

### Building static library for Linux

```bash
make build-libvpx-linux
```

## License

- Go bindings: MIT License
- libvpx: BSD 3-Clause License (see `lib/LICENSE.libvpx`)

## Credits

- Original library by [xlab](https://github.com/xlab/libvpx-go)
- libvpx by [The WebM Project](http://www.webmproject.org/)
