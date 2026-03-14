#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
WHISPER_DIR="$PROJECT_DIR/.whisper-cpp"
WEB_WHISPER_DIR="$PROJECT_DIR/web/whisper"

MODEL_NAME="${1:-base}"

echo "=== whisper.wasm セットアップ ==="

# Check for Emscripten
if ! command -v emcc &> /dev/null; then
  echo "Error: Emscripten (emcc) が見つかりません"
  echo "インストール: https://emscripten.org/docs/getting_started/downloads.html"
  echo ""
  echo "  git clone https://github.com/emscripten-core/emsdk.git"
  echo "  cd emsdk && ./emsdk install latest && ./emsdk activate latest"
  echo "  source ./emsdk_env.sh"
  exit 1
fi

# Clone whisper.cpp
if [ ! -d "$WHISPER_DIR" ]; then
  echo "whisper.cpp をクローン中..."
  git clone --depth 1 https://github.com/ggerganov/whisper.cpp.git "$WHISPER_DIR"
else
  echo "whisper.cpp は既に存在します"
fi

# Download model
MODEL_FILE="$WHISPER_DIR/models/ggml-${MODEL_NAME}.bin"
if [ ! -f "$MODEL_FILE" ]; then
  echo "モデル ggml-${MODEL_NAME} をダウンロード中..."
  cd "$WHISPER_DIR"
  bash models/download-ggml-model.sh "$MODEL_NAME"
else
  echo "モデルは既にダウンロード済みです"
fi

# Build WASM
echo "WASM をビルド中..."
cd "$WHISPER_DIR"
mkdir -p build-wasm
cd build-wasm
emcmake cmake .. \
  -DWHISPER_WASM=ON \
  -DWHISPER_WASM_SINGLE_FILE=ON \
  -DCMAKE_BUILD_TYPE=Release
make -j$(nproc 2>/dev/null || sysctl -n hw.ncpu) main

# Copy artifacts
echo "成果物をコピー中..."
mkdir -p "$WEB_WHISPER_DIR"
cp bin/main.js "$WEB_WHISPER_DIR/whisper.js"
cp bin/main.wasm "$WEB_WHISPER_DIR/whisper.wasm" 2>/dev/null || true

# Convert model to format suitable for web delivery
echo "モデルをコピー中..."
cp "$MODEL_FILE" "$WEB_WHISPER_DIR/ggml-${MODEL_NAME}.bin"

echo ""
echo "=== セットアップ完了 ==="
echo "  WASM: $WEB_WHISPER_DIR/whisper.js"
echo "  モデル: $WEB_WHISPER_DIR/ggml-${MODEL_NAME}.bin"
