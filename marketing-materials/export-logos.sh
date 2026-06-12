#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# marketing-materials/export-logos.sh
#
# Rasterizes the brand logo SVGs into PNGs for marketing use.
#
# Produces two groups, both under marketing-materials/assets/logos/ :
#
#   1. The two composed lockups the Typst templates embed:
#        logo-light.png   navy + orange lockup  (for light ad backgrounds)
#        logo-dark.png    all-WHITE lockup       (for dark  ad backgrounds)
#
#   2. individual/<name>.png — EVERY brand logo SVG rendered on its own
#        (gear marks, wordmarks, and lockup variants), so each mark is
#        available as a standalone raster.
#
# Why a rasterize step exists: the lockups are *composite* SVGs that pull the
# gear + wordmark in via nested <image href="/images/…"> references. Typst's
# image() loader can't resolve those, so we flatten to PNG here (hrefs
# resolved against the Hugo static root).
#
# Contrast fix for dark backgrounds: the white-transparent lockup ships with
# the FULL-COLOUR gear (gear-mark-transparent.svg), whose navy parts vanish on
# a navy panel. For logo-dark.png we swap it for the white gear
# (gear-mark-white.svg) so the whole mark contrasts.
#
# Output is PNG, not JPEG, to preserve transparency on light OR dark surfaces.
# Re-run after changing any brand logo SVG. assets/logos/ is gitignored.
#
# Requires one of: inkscape | rsvg-convert | magick (ImageMagick) on PATH.
# Override output width (px): WIDTH=2400 ./export-logos.sh
# ─────────────────────────────────────────────────────────────────────────────
set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
STATIC_DIR=$(cd "$SCRIPT_DIR/../frontend/static" && pwd)
LOGO_SRC_DIR="$STATIC_DIR/images/branding/logos"
OUT_DIR="$SCRIPT_DIR/assets/logos"
IND_DIR="$OUT_DIR/individual"
WIDTH="${WIDTH:-1600}"

mkdir -p "$OUT_DIR" "$IND_DIR"

if   command -v inkscape     >/dev/null 2>&1; then RENDERER=inkscape
elif command -v rsvg-convert >/dev/null 2>&1; then RENDERER=rsvg
elif command -v magick       >/dev/null 2>&1; then RENDERER=magick
else
  echo "error: need one of inkscape, rsvg-convert, or magick on PATH." >&2
  exit 1
fi

# render_svg <src.svg> <out.png> [sed-substitution]
# The optional 3rd arg is an extra `sed` expression applied before the href
# rewrite (used to swap the gear for the white variant on the dark lockup).
render_svg() {
  local src="$1" out="$2" pre="${3:-}"
  local tmp; tmp=$(mktemp --suffix=.svg)
  if [[ -n "$pre" ]]; then
    sed -e "$pre" -e "s#href=\"/images/#href=\"file://$STATIC_DIR/images/#g" "$src" > "$tmp"
  else
    sed "s#href=\"/images/#href=\"file://$STATIC_DIR/images/#g" "$src" > "$tmp"
  fi
  case "$RENDERER" in
    inkscape) inkscape "$tmp" --export-type=png --export-filename="$out" -w "$WIDTH" >/dev/null 2>&1 ;;
    rsvg)     rsvg-convert -w "$WIDTH" "$tmp" -o "$out" ;;
    magick)   magick -background none -density 300 "$tmp" -resize "${WIDTH}x" "$out" ;;
  esac
  rm -f "$tmp"
}

echo "Rasterizing brand logos (${WIDTH}px, $RENDERER):"

# 1) Composed lockups embedded by the Typst templates.
render_svg "$LOGO_SRC_DIR/horizontal-lockup-notagline.svg"        "$OUT_DIR/logo-light.png"
render_svg "$LOGO_SRC_DIR/horizontal-lockup-white-transparent.svg" "$OUT_DIR/logo-dark.png" \
           "s#gear-mark-transparent.svg#gear-mark-white.svg#"
echo "  templates → logo-light.png, logo-dark.png (dark uses white gear)"

# 2) Every brand logo SVG, rendered individually.
count=0
for svg in "$LOGO_SRC_DIR"/*.svg; do
  [[ -e "$svg" ]] || continue
  base=$(basename "$svg" .svg)
  render_svg "$svg" "$IND_DIR/$base.png"
  count=$((count + 1))
done
echo "  individual → $count logos in assets/logos/individual/"
echo "done."
