#!/usr/bin/env bash
# Build Faultline.
#   ./build.sh                 desktop app for this OS (Linux: webkit2_41)
#   ./build.sh windows         desktop .exe (cross-compile; needs mingw-w64-gcc)
#   ./build.sh windows -nsis   .exe plus NSIS installer
#   ./build.sh tui             TUI-only binary for this OS
#   ./build.sh tui windows     TUI-only faultline.exe
#   ./build.sh extension       zip the unpacked browser extension
#   ./build.sh release v0.1.0  Linux + Windows + extension, publish with gh
#   ./build.sh release v0.1.0 -nsis
#   NOTES='…' ./build.sh release v0.1.0
set -euo pipefail

cd "$(dirname "$0")"
mkdir -p build/bin

target="${1:-desktop}"
if [[ $# -gt 0 ]]; then
  shift
fi

windows_cc() {
  export CGO_ENABLED=1
  export CC="${CC:-x86_64-w64-mingw32-gcc}"
  export CXX="${CXX:-x86_64-w64-mingw32-g++}"
  if ! command -v "$CC" >/dev/null; then
    echo "missing $CC — install mingw-w64-gcc (Arch: pacman -S mingw-w64-gcc)" >&2
    exit 1
  fi
}

build_desktop() {
  local tags="${TAGS:-}"
  if [[ "$(uname -s)" == "Linux" && -z "$tags" ]]; then
    tags="webkit2_41"
  fi
  echo "Building desktop app${tags:+ (tags: $tags)}…"
  if [[ -n "$tags" ]]; then
    wails build -tags "$tags" "$@"
  else
    wails build "$@"
  fi
  echo "→ build/bin/faultline"
}

build_windows() {
  windows_cc
  echo "Building Windows desktop app…"
  wails build -platform windows/amd64 "$@"
  echo "→ build/bin/faultline.exe"
}

build_tui() {
  if [[ "${1:-}" == "windows" ]]; then
    shift || true
    echo "Building Windows TUI binary…"
    GOOS=windows GOARCH=amd64 go build -o build/bin/faultline.exe ./cmd/faultline "$@"
    echo "→ build/bin/faultline.exe"
  else
    echo "Building TUI-only binary…"
    go build -o build/bin/faultline ./cmd/faultline "$@"
    echo "→ build/bin/faultline"
  fi
}

pack_extension() {
  local dest="${1:-build/bin/faultline-extension.zip}"
  local ver="${2:-}"
  local tmp
  tmp="$(mktemp -d)"
  mkdir -p "$tmp/faultline-extension"
  cp -a extension/. "$tmp/faultline-extension/"
  if [[ -n "$ver" ]]; then
    local numeric="${ver#v}"
    python3 - "$tmp/faultline-extension/manifest.json" "$numeric" <<'PY'
import json, sys
path, version = sys.argv[1], sys.argv[2]
with open(path, encoding="utf-8") as f:
    manifest = json.load(f)
manifest["version"] = version
with open(path, "w", encoding="utf-8") as f:
    json.dump(manifest, f, indent=2)
    f.write("\n")
PY
  fi
  mkdir -p "$(dirname "$dest")"
  rm -f "$dest"
  python3 - "$tmp" "$dest" <<'PY'
import os, sys, zipfile
root, dest = sys.argv[1], sys.argv[2]
os.makedirs(os.path.dirname(dest) or ".", exist_ok=True)
with zipfile.ZipFile(dest, "w", zipfile.ZIP_DEFLATED) as zf:
    for dirpath, dirnames, filenames in os.walk(root):
        dirnames[:] = [d for d in dirnames if d not in {".DS_Store", "__MACOSX"}]
        for name in filenames:
            if name == ".DS_Store":
                continue
            full = os.path.join(dirpath, name)
            zf.write(full, os.path.relpath(full, root))
PY
  rm -rf "$tmp"
  echo "→ $dest"
}

publish_release() {
  local ver="${1:-}"
  if [[ -z "$ver" ]]; then
    echo "usage: $0 release v0.1.0 [-nsis]" >&2
    exit 1
  fi
  shift || true
  if ! command -v gh >/dev/null; then
    echo "gh is required to publish (https://cli.github.com/)" >&2
    exit 1
  fi
  if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    echo "not a git repo" >&2
    exit 1
  fi

  local stage="build/release"
  rm -rf "$stage"
  mkdir -p "$stage"

  build_desktop
  if [[ ! -f build/bin/faultline ]]; then
    echo "expected build/bin/faultline" >&2
    exit 1
  fi
  cp build/bin/faultline "$stage/faultline-linux-amd64"

  build_windows "$@"
  if [[ ! -f build/bin/faultline.exe ]]; then
    echo "expected build/bin/faultline.exe" >&2
    exit 1
  fi
  cp build/bin/faultline.exe "$stage/faultline-windows-amd64.exe"

  pack_extension "$stage/faultline-extension.zip" "$ver"

  local f
  for f in build/bin/*setup*.exe build/bin/*installer*.exe; do
    if [[ -f "$f" ]]; then
      cp "$f" "$stage/"
    fi
  done

  local -a files=()
  while IFS= read -r f; do
    files+=("$f")
  done < <(find "$stage" -type f | sort)
  if [[ ${#files[@]} -eq 0 ]]; then
    echo "no release assets in $stage" >&2
    exit 1
  fi

  local -a notes=(--generate-notes)
  if [[ -n "${NOTES:-}" ]]; then
    notes=(--notes "$NOTES")
  fi

  echo "Publishing ${ver} with:"
  printf '  %s\n' "${files[@]}"

  git push origin HEAD

  if gh release view "$ver" >/dev/null 2>&1; then
    gh release upload "$ver" "${files[@]}" --clobber
    echo "Updated existing release $ver"
  else
    gh release create "$ver" --title "$ver" "${notes[@]}" --latest "${files[@]}"
    echo "Created $ver"
  fi
  gh release view "$ver" --web
}

case "$target" in
  desktop | linux)
    build_desktop "$@"
    ;;
  windows)
    build_windows "$@"
    ;;
  tui)
    build_tui "$@"
    ;;
  extension)
    pack_extension "$@"
    ;;
  release)
    publish_release "$@"
    ;;
  *)
    echo "usage: $0 [desktop|windows|tui|extension|release] …" >&2
    echo "       $0 release v0.1.0 [-nsis]" >&2
    exit 1
    ;;
esac
