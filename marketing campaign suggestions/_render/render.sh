#!/usr/bin/env bash
# Renders every marketing-campaign SVG into platform-correct PNGs.
# Run from repo root:  bash "marketing campaign suggestions/_render/render.sh"
set -e

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/_render"
mkdir -p "$OUT"

# ---- helpers ----
render() {
  # render <src.svg> <width> <height> <out.png>
  local src="$1" w="$2" h="$3" out="$4"
  echo "  → $(basename "$out")  (${w}x${h})"
  inkscape "$src" --export-type=png --export-filename="$out" \
                  --export-width="$w" --export-height="$h" 2>/dev/null
}

# ---- 00 visual identity ----
echo "== Visual identity =="
render "$ROOT/00-visual-identity/01-cube-mark.svg"           1024 1024 "$OUT/00_cube-mark_1024.png"
render "$ROOT/00-visual-identity/02-horizontal-lockup.svg"   1600  400 "$OUT/00_horizontal-lockup_1600x400.png"
render "$ROOT/00-visual-identity/03-vertical-lockup.svg"      800 1000 "$OUT/00_vertical-lockup_800x1000.png"
render "$ROOT/00-visual-identity/04-avatar.svg"              1024 1024 "$OUT/00_avatar_1024.png"
render "$ROOT/00-visual-identity/05-community-badge.svg"     1024 1024 "$OUT/00_community-badge_1024.png"
render "$ROOT/00-visual-identity/06-monochrome.svg"           800  800 "$OUT/00_monochrome_800.png"

# ---- 01 instagram ----
echo "== Instagram =="
render "$ROOT/00-visual-identity/04-avatar.svg"               320  320 "$OUT/01_ig_profile_320.png"
render "$ROOT/01-instagram/01-square-welcome.svg"            1080 1080 "$OUT/01_ig_square-welcome_1080.png"
render "$ROOT/01-instagram/02-square-tools.svg"              1080 1080 "$OUT/01_ig_square-tools_1080.png"
render "$ROOT/01-instagram/03-portrait-event.svg"            1080 1350 "$OUT/01_ig_portrait-event_1080x1350.png"
render "$ROOT/01-instagram/04-story-droptintry.svg"          1080 1920 "$OUT/01_ig_story_1080x1920.png"

# ---- 02 facebook ----
echo "== Facebook =="
render "$ROOT/00-visual-identity/04-avatar.svg"               360  360 "$OUT/02_fb_profile_360.png"
render "$ROOT/02-facebook/01-cover-1640x624.svg"             1640  624 "$OUT/02_fb_cover_1640x624.png"
render "$ROOT/02-facebook/02-link-card-1200x630.svg"         1200  630 "$OUT/02_fb_link-card_1200x630.png"
render "$ROOT/02-facebook/03-event-cover-1920x1005.svg"      1920 1005 "$OUT/02_fb_event-cover_1920x1005.png"
render "$ROOT/01-instagram/01-square-welcome.svg"            1080 1080 "$OUT/02_fb_square-welcome_1080.png"
render "$ROOT/01-instagram/04-story-droptintry.svg"          1080 1920 "$OUT/02_fb_story_1080x1920.png"

# ---- 03 google business ----
echo "== Google Business =="
render "$ROOT/00-visual-identity/04-avatar.svg"               720  720 "$OUT/03_gb_logo_720.png"
render "$ROOT/03-google-business/01-cover-1920x1080.svg"     1920 1080 "$OUT/03_gb_cover_1920x1080.png"
render "$ROOT/03-google-business/02-post-update-1200x900.svg" 1200  900 "$OUT/03_gb_post_1200x900.png"

# ---- 04 google ads ----
echo "== Google Ads =="
render "$ROOT/00-visual-identity/04-avatar.svg"              1200 1200 "$OUT/04_gads_logo-square_1200.png"
render "$ROOT/00-visual-identity/02-horizontal-lockup.svg"   1200  300 "$OUT/04_gads_logo-landscape_1200x300.png"
render "$ROOT/04-google-ads/01-display-landscape-1200x628.svg" 1200  628 "$OUT/04_gads_landscape_1200x628.png"
render "$ROOT/04-google-ads/02-display-square-1200x1200.svg"   1200 1200 "$OUT/04_gads_square_1200x1200.png"
render "$ROOT/04-google-ads/03-banner-728x90.svg"               728   90 "$OUT/04_gads_banner_728x90.png"
render "$ROOT/04-google-ads/04-banner-300x250.svg"              300  250 "$OUT/04_gads_banner_300x250.png"
render "$ROOT/04-google-ads/05-banner-300x600.svg"              300  600 "$OUT/04_gads_banner_300x600.png"

# ---- cleanup test files from earlier development ----
rm -f "$OUT"/test_*.png

echo
echo "Done. PNGs written to: $OUT"
ls -1 "$OUT" | grep -v '\.sh$' | sort
