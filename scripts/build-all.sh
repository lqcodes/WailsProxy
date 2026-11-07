#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# WailsProxy 跨平台构建与内核依赖打包脚本
# 支持目标平台:
#   mac-arm64      : macOS Apple Silicon (M1/M2/M3/M4 苹果芯)
#   mac-amd64      : macOS Intel (x86_64 英特尔芯)
#   mac-universal  : macOS Universal (苹果芯 + 英特尔芯 双架构通用包)
#   windows-amd64  : Windows 64位 (x64 .exe)
#   windows-arm64  : Windows ARM64 (.exe)
#   linux-amd64    : Linux 64位 (x86_64)
#   linux-arm64    : Linux ARM64 (aarch64)
# ==============================================================================

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

export PATH="$HOME/.local/bin:$HOME/code/go/bin:/usr/local/go/bin:$PATH"

TARGET="${1:-mac-arm64}"

bundle_macos_deps() {
  local app_path="$ROOT_DIR/build/bin/WailsProxy.app"
  if [ -d "$app_path" ] && [ -d "$ROOT_DIR/bin" ]; then
    echo ">>> 正在将开源内核与 Geo 规则库打包进 $app_path ..."
    mkdir -p "$app_path/Contents/MacOS/bin" "$app_path/Contents/Resources/bin" "$ROOT_DIR/build/bin/bin"
    for f in xray sing-box geoip.dat geosite.dat; do
      if [ -f "$ROOT_DIR/bin/$f" ]; then
        cp -f "$ROOT_DIR/bin/$f" "$app_path/Contents/MacOS/bin/"
        cp -f "$ROOT_DIR/bin/$f" "$app_path/Contents/Resources/bin/"
        cp -f "$ROOT_DIR/bin/$f" "$ROOT_DIR/build/bin/bin/"
      fi
    done
    cp -f "$app_path/Contents/MacOS/WailsProxy" "$ROOT_DIR/build/bin/WailsProxy"
  fi
}

case "$TARGET" in
  mac-arm64)
    echo "=== 构建 macOS Apple Silicon (darwin/arm64 苹果芯) ==="
    wails build -platform darwin/arm64 -clean
    bundle_macos_deps
    ;;
  mac-amd64)
    echo "=== 构建 macOS Intel (darwin/amd64 英特尔芯) ==="
    wails build -platform darwin/amd64 -clean
    bundle_macos_deps
    ;;
  mac-universal)
    echo "=== 构建 macOS Universal (苹果芯 + 英特尔芯 双架构通用包) ==="
    wails build -platform darwin/universal -clean
    bundle_macos_deps
    ;;
  windows-amd64)
    echo "=== 构建 Windows x64 (windows/amd64) ==="
    wails build -platform windows/amd64 -clean
    ;;
  windows-arm64)
    echo "=== 构建 Windows ARM64 (windows/arm64) ==="
    wails build -platform windows/arm64 -clean
    ;;
  linux-amd64)
    echo "=== 构建 Linux x86_64 (linux/amd64) ==="
    wails build -platform linux/amd64 -clean
    ;;
  linux-arm64)
    echo "=== 构建 Linux ARM64 (linux/arm64) ==="
    wails build -platform linux/arm64 -clean
    ;;
  *)
    echo "用法: $0 {mac-arm64|mac-amd64|mac-universal|windows-amd64|windows-arm64|linux-amd64|linux-arm64}"
    exit 1
    ;;
esac

echo "=== 构建完成: $TARGET ==="
