# O'Town Makerspace — Marketing Campaign Suggestions

This folder contains the visual-identity refresh and a set of social-media branding materials sized for the platforms below. Source files are SVG; rasterised PNGs (at exact pixel dimensions) live in `_render/`.

```
marketing campaign suggestions/
├── README.md                         <- you are here
├── 00-visual-identity/               <- logo improvements + brand spec
├── 01-instagram/
├── 02-facebook/
├── 03-google-business/
├── 04-google-ads/
└── _render/                          <- generated PNGs at platform-correct sizes
```

## Brand voice — friendly, safe, enthusiastic

| Pillar | What it means in copy & imagery |
|---|---|
| **Friendly** | Plain language, second person ("you", "your"), no jargon. Smiling people, hands working together. |
| **Safe** | Show PPE (goggles, gloves), supervised onboarding, "Newcomers welcome", clear rules ("Ages 14+", "Inductions before solo use"). |
| **Enthusiastic** | Active verbs ("Make", "Build", "Try", "Drop in"). Bright orange call-outs. Real member projects, not stock. |

## Brand colours (harmonised)

| Token | Hex | Use |
|---|---|---|
| Navy (primary) | `#0D1A63` | Backgrounds, headings, large surfaces |
| Navy 2 | `#1A2CA3` | Secondary surfaces, gradients |
| Logo Blue | `#3B4BA0` | Existing logo elements (kept for continuity) |
| Orange (accent) | `#F68048` | Buttons, highlights, accents |
| Orange Hot | `#FF5722` | Logo orange (kept), bold call-outs |
| Orange Deep | `#E64A19` | Pressed/hover states, decorative shadows |
| Cream | `#EDF2F8` | Light backgrounds, large body text on dark |
| White | `#FFFFFF` | Foreground type on photo overlays |

## Type system

Two cuts of the same family — paired intentionally so site + logo feel like one design system:

- **Inter Display, 900** — logo wordmark, large headlines, hero text (≥24 px)
- **Inter, 400–600** — body, UI, paragraphs (already loaded by the site theme)

See `00-visual-identity/BRAND_GUIDELINES.md` § *Type system* for the full rationale, the loading-status note (Inter Display isn't currently shipped by the site), and the stroke-padding × letter-spacing rule.

---

## Social-media format reference

The table below lists every asset format used across the four target platforms. **All sizes are in pixels.** Source SVGs scale infinitely — the exported PNGs in `_render/` are sized to the recommended values (the larger of each platform's "minimum" and "recommended" specs, so they look sharp on retina displays).

### Instagram

| Asset | Aspect | Size (px) | Where it appears | Notes |
|---|---|---|---|---|
| Profile picture | 1:1 | 320 × 320 | Avatar, comments, story ring | Displayed circular at 110×110; keep mark inside a centre-safe disc |
| Feed post — square | 1:1 | 1080 × 1080 | Grid + feed | Default for product/announcement posts |
| Feed post — portrait | 4:5 | 1080 × 1350 | Grid + feed | Takes more screen real estate; best for promo posts |
| Feed post — landscape | 1.91:1 | 1080 × 566 | Feed only | Rare; useful for banners |
| Story | 9:16 | 1080 × 1920 | Stories, ad placements | Top/bottom 250 px reserved for UI (safe zone) |
| Reel cover | 9:16 | 1080 × 1920 | Reels grid (cropped to 1:1) | Centre 1080 × 1080 must read on its own (the grid crop) |
| Carousel slide | 1:1 or 4:5 | 1080 × 1080 / 1080 × 1350 | Multi-image post | Keep aspect consistent across slides |

### Facebook

| Asset | Aspect | Size (px) | Where it appears | Notes |
|---|---|---|---|---|
| Profile picture | 1:1 | 360 × 360 | Page avatar, comments | Displayed circular at 170×170 desktop, 128×128 mobile |
| Cover photo | ~2.7:1 | 1640 × 624 | Top of page | Mobile shows centre 640 × 360; keep key content centred |
| Feed post — square | 1:1 | 1080 × 1080 | Timeline, feed | |
| Feed post — landscape | 1.91:1 | 1200 × 630 | Timeline, feed | Same dimensions as link preview |
| Link preview / OG image | 1.91:1 | 1200 × 630 | When sharing a URL | Already exists at `static/images/og-default.jpg` |
| Event cover | 1.91:1 | 1920 × 1005 | Events page top | Cropped to 16:9 on mobile |
| Story | 9:16 | 1080 × 1920 | Stories | Same as Instagram |

### Google Business Profile

| Asset | Aspect | Size (px) | Where it appears | Notes |
|---|---|---|---|---|
| Logo | 1:1 | 720 × 720 | Knowledge panel, search results | Min 250 × 250; PNG with transparent background |
| Cover photo | 16:9 | 1920 × 1080 | Top of profile | First impression in Maps |
| Profile photo | 1:1 | 720 × 720 | Reviews, posts | Often shown alongside logo |
| Post image | 4:3 | 1200 × 900 | Updates, offers, events | Min 400 × 300 |
| Additional photos | varies | ≥720 × 720 | Photo gallery on profile | Interior, equipment, team, "at work" shots |

### Google Ads — Responsive Display

These are the assets uploaded to a single Responsive Display Ad; Google composes them automatically across the network.

| Asset | Aspect | Size (px) | Required? | Notes |
|---|---|---|---|---|
| Marketing image (landscape) | 1.91:1 | 1200 × 628 | ✓ required | The "hero" image; up to 15 per ad |
| Marketing image (square) | 1:1 | 1200 × 1200 | ✓ required | Pairs with the landscape image |
| Marketing image (portrait, optional) | 4:5 | 960 × 1200 | optional | Improves mobile vertical placements |
| Logo (square) | 1:1 | 1200 × 1200 | ✓ required | Min 128 × 128, transparent or solid |
| Logo (landscape) | 4:1 | 1200 × 300 | recommended | Used in larger placements |

### Google Ads — Display Banner sizes (legacy, for uploaded image ads)

| Placement | Size (px) | Notes |
|---|---|---|
| Medium rectangle | 300 × 250 | Most common; in-content |
| Large rectangle | 336 × 280 | Higher visibility variant |
| Leaderboard | 728 × 90 | Top-of-page desktop |
| Large leaderboard | 970 × 90 | Newer wider banner |
| Half-page / Wide skyscraper | 300 × 600 | High-engagement sidebar |
| Skyscraper | 160 × 600 | Sidebar |
| Mobile leaderboard | 320 × 50 | Mobile top-of-page |
| Large mobile banner | 320 × 100 | Mobile high-engagement |
| Square | 250 × 250 | Compact placements |
| Small square | 200 × 200 | Compact placements |

**File-size limit for Google Ads images: 150 KB per asset.** Use PNG-8 or aggressive JPEG (q≈80) for photo banners.

---

## Production notes

- All source files in this folder are **SVG**. Edit text and colours directly; re-export PNGs with the `_render/render.sh` script (run it from the repo root).
- The current site logo is `static/images/branding/otown_logo_2.svg`. Improved variants (see `00-visual-identity/`) are intended as additions, not replacements — adopt them gradually.
- Photography for posts: prefer the existing `assets/images/socials/` and `assets/images/workspace/` collections — they already capture the friendly, hands-on, real-people energy we want.
