#!/usr/bin/env bash
# Regenerate the screens in assets/*.svg.
#
# The frames come out of the ui package itself (PREVIEW=1 go test), so what
# ends up in the README is the real render() output -- the same bytes the
# terminal is handed -- rather than a drawing of it. Rerun this after any
# change to the interface.
#
# SVG rather than PNG: it stays sharp at whatever width the README renders
# at, the text is selectable and searchable, and a screen is kilobytes
# instead of hundreds of kilobytes.
#
# Needs: go, and ansisvg -- go install github.com/wader/ansisvg@latest

set -euo pipefail
cd "$(dirname "$0")/.."

ansisvg=""
for candidate in ansisvg "$(go env GOPATH)/bin/ansisvg" "$(go env GOPATH)/bin/ansisvg.exe"; do
  if command -v "$candidate" >/dev/null 2>&1; then
    ansisvg="$candidate"
    break
  fi
done
if [ -z "$ansisvg" ]; then
  echo "ansisvg not found -- go install github.com/wader/ansisvg@latest" >&2
  exit 1
fi

PREVIEW=1 go test ./internal/ui -run TestPreview -count=1 >/dev/null

# Pixel units, not ch/em: an <img> on a README gives the SVG no font context
# to resolve font-relative sizes against, and the frame would come out the
# wrong shape. 9x20 is the character box for a 15px monospace face.
#
# --grid gives every character its own x instead of letting the line flow at
# the font's natural advance. Without it the text drifts away from the
# backgrounds painted under it whenever the reader's monospace font is not
# exactly 9px wide -- Consolas is 8.25px, which by the end of the status bar
# is most of a tenth of the width.
for ans in assets/frames/*.ans; do
  name=$(basename "$ans" .ans)
  "$ansisvg" \
    --fontsize 15 \
    --charboxsize 9x20 \
    --marginsize 18x18 \
    --grid \
    --fontname "JetBrains Mono,DejaVu Sans Mono,Menlo,Consolas,monospace" \
    < "$ans" > "assets/$name.svg"
  echo "· assets/$name.svg ($(wc -c < "assets/$name.svg") bytes)"
done
