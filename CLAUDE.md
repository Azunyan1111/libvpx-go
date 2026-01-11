# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## プロジェクト概要

libvpx-goは、libvpx（VP8/VP9コーデック）のGoバインディングライブラリです。c-for-goツールを使用して自動生成されたCGOラッパーを提供しています。

## ビルドコマンド

```bash
# バインディングの再生成（c-for-goが必要）
make

# 生成ファイルのクリーンアップ
make clean

# ビルドテスト
make test
# または
cd vpx && go build
```

## 必要な依存関係

```bash
# macOS
brew install libvpx pkg-config

# Linux (Debian/Ubuntu)
apt-get install libvpx-dev

# デモアプリ用追加依存（cmd/webm-player）
brew install libogg libvorbis opus portaudio  # macOS
apt-get install libogg-dev libvorbis-dev libopus-dev portaudio19-dev  # Linux
```

## アーキテクチャ

### コード構造

- `vpx/` - メインパッケージ（自動生成されたバインディング + 手書きの拡張）
  - 自動生成ファイル: `vpx.go`, `types.go`, `const.go`, `doc.go`, `cgo_helpers.*`
  - 手書きファイル: `iface.go`（VP8/VP9エンコーダ・デコーダインターフェース）, `image.go`（YUV->RGBA変換）, `error.go`
- `cmd/webm-player/` - WebMプレイヤーのデモアプリケーション
- `vpx.yml` - c-for-go用のバインディング生成設定

### 主要API

```go
// デコーダインターフェースの取得
vpx.DecoderIfaceVP8()
vpx.DecoderIfaceVP9()
vpx.DecoderFor(fourcc int) // fourccに基づいて自動選択

// エンコーダインターフェースの取得
vpx.EncoderIfaceVP8()
vpx.EncoderIfaceVP9()
vpx.EncoderFor(fourcc int)

// デコード処理
vpx.CodecDecInitVer(ctx, iface, cfg, flags, ver)
vpx.CodecDecode(ctx, data, dataSz, userPriv, deadline)
vpx.CodecGetFrame(ctx, iter)

// エンコード処理
vpx.CodecEncInitVer(ctx, iface, cfg, flags, ver)
vpx.CodecEncode(ctx, img, pts, duration, flags, deadline)
vpx.CodecGetCxData(ctx, iter)

// 画像変換（Image構造体のメソッド）
img.ImageRGBA()   // YUV420 -> RGBA変換
img.ImageYCbCr()  // Go標準のimage.YCbCr形式へ変換
```

### バインディング再生成

vpx.ymlを編集後、以下のコマンドで再生成:

```bash
c-for-go -ccdefs vpx.yml
```

vpx.ymlの設定内容:
- `GENERATOR`: パッケージ名、pkg-configオプション、インクルードヘッダー
- `PARSER`: ヘッダーファイルのパス、定義済み定数（VP8_FOURCC等）
- `TRANSLATOR`: 定数ルール、メモリヒント、ポインタヒント、命名変換ルール
