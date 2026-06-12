#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# marketing-materials/render.sh
#
# Compiles every O'Town Makerspace marketing template into an SVG for every
# (format × lang × theme × slide) combination via Typst.
#
# Mirrors esio-website/marketing-typst/render.sh, adapted to otm:
#   - the brand logo is a rasterized PNG, so this script first runs
#     export-logos.sh to (re)generate assets/logos/*.png before compiling,
#   - --root is the otm repo root, so absolute paths like
#     /marketing-materials/… and the logo PNGs resolve.
#
# Output — language / platform / theme:
#   marketing-materials/out/<lang>/<format>/<theme>/<slide>.svg
# out/ and the logo PNGs are gitignored — re-run this script to regenerate.
#
# Requires: typst on PATH (or TYPST=path/to/typst), plus a rasterizer for
# export-logos.sh (inkscape | rsvg-convert | magick).
# ─────────────────────────────────────────────────────────────────────────────
set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd "$SCRIPT_DIR/.." && pwd)
OUTPUT_DIR="$SCRIPT_DIR/out"
FONT_PATH="$SCRIPT_DIR/fonts"
TYPST="${TYPST:-typst}"
# Pixels-per-inch for the PNG export. Templates size pages in pt where 1pt is
# meant as 1px. At 72ppi the PNG matches those exact pixel dimensions; higher
# ppi multiplies them for crisper (retina) output.
#
# Social + Facebook cover render at PPI (default 144 = 2×) so they look sharp
# on modern displays — the platforms downscale. Google Display banners are
# FORCED to 72 below: IAB slots reject off-spec pixel dimensions, so a
# "300×250" must export at exactly 300×250.
PPI="${PPI:-144}"

if ! command -v "$TYPST" >/dev/null 2>&1; then
  echo "error: typst not found on PATH. Install via 'apt install typst' or" >&2
  echo "       download from github.com/typst/typst/releases and set TYPST=path/to/typst." >&2
  exit 1
fi

# Brand logos are embedded as rasterized PNGs — (re)generate them first so a
# fresh checkout renders without an empty logo.
echo "Exporting brand logos…"
"$SCRIPT_DIR/export-logos.sh"
echo ""

# Format catalogue — slides are .typ basenames under marketing-materials/<format>/
declare -A SLIDES=(
  ["square"]="01-hook 02-pain 03-solution 04-proof 05-cta"
  ["google-ads"]="160x600 300x250 300x600 320x50 728x90"
  ["facebook"]="cover"
)

LANGS=("en" "da")
THEMES=("light" "dark")

total=0
for lang in "${LANGS[@]}"; do
  for format in square google-ads facebook; do
    for theme in "${THEMES[@]}"; do
      out_dir="$OUTPUT_DIR/$lang/$format/$theme"
      mkdir -p "$out_dir"
      for slide in ${SLIDES[$format]}; do
        src="$SCRIPT_DIR/$format/${slide}.typ"
        if [[ ! -f "$src" ]]; then
          echo "warn: missing $src — skipping" >&2
          continue
        fi
        # Shared compile flags.
        common=(--root "$REPO_ROOT" --font-path "$FONT_PATH" --input "theme=$theme" --input "lang=$lang")
        # SVG — scalable master.
        "$TYPST" compile "$src" "$out_dir/${slide}.svg" --format svg "${common[@]}"
        # PNG — rasterized, upload-ready marketing asset (logo baked in).
        # Banners stay at 72ppi for spec-exact IAB pixels; everything else
        # renders at PPI (default 2×) for crispness.
        if [[ "$format" == "google-ads" ]]; then ppi=72; else ppi="$PPI"; fi
        "$TYPST" compile "$src" "$out_dir/${slide}.png" --format png --ppi "$ppi" "${common[@]}"
        total=$((total + 1))
        printf "  → %s/%s/%s/%s.{svg,png}\n" "$lang" "$format" "$theme" "$slide"
      done
    done
  done
done

echo ""
echo "done. $total asset(s), each as .svg (scalable) + .png (rasterized, ${PPI}ppi)"
echo "      written to $OUTPUT_DIR"
