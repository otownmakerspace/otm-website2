# O'Town Makerspace — Brand Guidelines

A practical reference for the gear-mark + wordmark logo system. Every value
in this document is checkable against the canonical SVG files in this folder
(and their twins in `static/images/branding/logos/`). When the docs and the
SVGs disagree, **the SVGs win** — fix the docs.

---

## 1. The system at a glance

The brand is a two-piece system:

| Piece | File | Role |
|---|---|---|
| **Gear-mark** | `gear-mark.svg` | Primary symbol. A 12-tooth gear ring with the letters **T** and **M** stacked inside. Use anywhere you need a recognizable square/circular badge — avatars, app icons, favicons, single-mark stamps. |
| **Wordmark** | `wordmark-consolidated.svg` | Primary type lockup. **O'TOWN** stacked above **MAKERSPACE**, both all-caps Inter Display 900. Use whenever space allows the full name to read at a comfortable size. |

The two pieces share a single design DNA:

- The same **typeface and weight** — Inter Display 900 — for the gear's TM letters and the wordmark.
- The same **colours** — navy `#0D1A63`, warm orange `#F68048`, cream `#EDF2F8`.
- The same **rounding trick** — every fill is paired with a matching-colour stroke at `stroke-linejoin="round"` and `stroke-linecap="round"`, with stroke-width ≈ 6% of font-size. That's what gives every edge its soft, slightly chunky finish without forcing us to redraw paths.
- A consistent **letter-spacing rule** — negative spacing on tight inner-circle layouts (gear: `-4`), positive spacing on the wide wordmark (`+8` / `+4`) to compensate for the stroke padding.

If you need to add a new variant or extension, copy these conventions before
inventing your own.

---

## 2. Colour palette

| Token | Hex | Use |
|---|---|---|
| **Navy** (primary) | `#0D1A63` | Gear teeth + body, M letter, wordmark `O'TOWN` line. Background for dark/photo overlays. |
| Navy 2 | `#1A2CA3` | Secondary surface, gradients, body copy on cream. |
| Logo Blue | `#3B4BA0` | Legacy logo blue — kept available for continuity but not used in the consolidated system. |
| **Orange** (accent) | `#F68048` | Gear T letter, wordmark `MAKERSPACE` line, CTAs, hot accents. |
| Orange Hot | `#FF5722` | Bright variant — bold call-outs only. Older logo orange; preserved. |
| Orange Deep | `#E64A19` | Pressed/hover states, decorative shadows. |
| **Cream** | `#EDF2F8` | Logo backgrounds (the gear's interior face), light surfaces, large body text on dark. |
| White | `#FFFFFF` | Knockouts, foreground type on photo overlays. |
| Black | `#000000` | Single-ink print only — not a brand colour for digital. |

Do not invent new tints. If you need a softer navy, use Navy 2; if you need
a softer orange, use cream as a contrast surface, not a tinted orange.

---

## 3. Typography

**Inter Display 900** — the heaviest cut of Inter — is the brand display
voice. It's used for:

- The TM letters inside the gear-mark (font-size 270 / 230, letter-spacing `-4`)
- Both lines of the wordmark (`O'TOWN` 200pt / `MAKERSPACE` 130pt, letter-spacing `+8` / `+4`)
- Headline copy across all marketing material

**Inter 400–600** — the same family at body weights — is used for paragraph
copy, UI, and small labels.

The two cuts pair intentionally: site + logo feel like one system because
they share the family.

### The rounded-corner stroke trick

Every brand glyph uses this pattern:

```svg
<text fill="#0D1A63" stroke="#0D1A63" stroke-width="14"
      stroke-linejoin="round" stroke-linecap="round">T</text>
```

`stroke-width` is set to roughly **6% of font-size** — that gives a corner
radius of stroke-width / 2, which reads as visibly soft but never blobby.
The stroke also adds chunky weight to every edge, contributing to the
heavy, friendly feel.

| Glyph | font-size | stroke-width | Ratio |
|---|---|---|---|
| Gear T | 270 | 14 | 5.2% |
| Gear M | 230 | 14 | 6.1% |
| Wordmark `O'TOWN` | 200 | 12 | 6.0% |
| Wordmark `MAKERSPACE` | 130 | 8 | 6.2% |

When extending the system, keep the ratio in this band (5–7%).

### Letter-spacing compensation

Stroke padding adds `stroke-width / 2` to every side of a glyph, which
visually closes the gap between adjacent letters. Compensate by widening
`letter-spacing` by approximately the stroke width, then bias slightly
tighter for a modern-display feel.

### Font loading status

- **Inter (regular cut):** ✅ loaded on the site via `themes/hugo-up-business-main/assets/css/fonts.css` (self-hosted woff2, weights 100–900, both styles).
- **Inter Display (display cut):** ⚠️ **not currently loaded.** Every SVG declares `font-family: 'Inter Display', 'Inter', sans-serif`, so today the browser falls back to plain Inter when rendering the wordmark or the gear-mark inline. The fallback is acceptable but soft — Inter Display's tighter, more condensed display cut is what gives the brand its punch.

To fix, either:

1. **Add Inter Display to the font loader** — download the woff2 files (rsms.me/inter or `@fontsource/inter`) and add matching `@font-face` declarations. Use this if you want the logo to remain live, editable text in the DOM.
2. **Convert text → paths in the final SVG** — open in Inkscape, select all text, run `Path → Object to Path`. Glyphs become outlined paths and render identically on any device, no font needed. **Recommended for production logos** — this is what most shipped brand SVGs do.

### Production note: convert to paths before shipping

For final production assets (PNG exports, vinyl-cut files, embroidery files,
laser-cut DXF), always run **Path → Object to Path** in Inkscape so the
type is baked into outlines. Keep the editable `<text>` source in version
control for future tweaks; ship the outlined version.

---

## 4. Variants

### 4.1 Gear-mark (square, 800 × 800 viewBox)

| File | Treatment | When to use |
|---|---|---|
| `gear-mark.svg` | Full colour: navy gear, cream interior, orange T, navy M. Cream background. | Default. Anywhere the surrounding canvas is light/cream. |
| `gear-mark-transparent.svg` | Full colour, **no background fill**. | Compositing onto badges, lockups, or canvases where the host bg should show through. |
| `gear-mark-white.svg` | White gear ring, navy interior, white T+M. **Navy bg baked in.** | Knockout on brand-navy surfaces. |
| `gear-mark-mono-navy.svg` | All navy, including T (loses the orange accent). | Spot-colour print, embossed signage, single-ink runs. |
| `gear-mark-mono-orange.svg` | All warm orange. | Single-PMS runs, tonal posters, orange-only contexts. |
| `gear-mark-black.svg` | Single-ink black. | Photocopier, fax, single-ink print, watermarks. |
| `gear-mark-outline.svg` | Hollow line-art, no fills. | Laser engraving, embroidery, embossing, etched signage. |
| `gear-mark-blueprint.svg` | White gear on a drafting-paper grid; the inner hub also reads as a window onto the same blueprint. | Hero graphics, "we make things" creative posts, technical-feel marketing. |
| `gear-mark-motion.svg` | Default colour gear plus three orange motion-streak arcs to the right of the gear, suggesting clockwise rotation. | Animated banners, "we're working", energetic CTAs. |

### 4.2 Wordmark (landscape, 1200 × 320 viewBox)

| File | Treatment | When to use |
|---|---|---|
| `wordmark-consolidated.svg` | Default: navy `O'TOWN` over orange `MAKERSPACE`. Transparent bg. | Default lockup for headers, footers, title-card type. |
| `wordmark-white.svg` | Both lines white, **navy bg baked in**. | Hard-edge knockout block on brand-navy. |
| `wordmark-white-transparent.svg` | Both lines white, no bg. | Photo overlays, dark-mode chrome, lockups where you don't want a navy box. |
| `wordmark-mono-navy.svg` | Both lines navy. | Single-colour navy runs. |
| `wordmark-mono-orange.svg` | Both lines warm orange. | Single-PMS orange runs, accent placements. |
| `wordmark-black.svg` | Single-ink black. | Photocopier, fax, single-ink print. |
| `wordmark-outline.svg` | Hollow stroke, no fill. | Laser, embroidery, embossing. |
| `wordmark-blueprint.svg` | White type on drafting-paper grid. | Pairs with `gear-mark-blueprint.svg` in technical-feel layouts. |

### 4.3 Pairing matrix — which gear with which wordmark

When using both pieces in one layout, match the colorway:

| Surface | Gear | Wordmark |
|---|---|---|
| Cream / light bg | `gear-mark.svg` | `wordmark-consolidated.svg` |
| Navy / dark photo | `gear-mark-white.svg` (or `-transparent` over a navy fill) | `wordmark-white-transparent.svg` |
| Single-PMS navy | `gear-mark-mono-navy.svg` | `wordmark-mono-navy.svg` |
| Single-PMS orange | `gear-mark-mono-orange.svg` | `wordmark-mono-orange.svg` |
| Single-ink black | `gear-mark-black.svg` | `wordmark-black.svg` |
| Engrave / embroider | `gear-mark-outline.svg` | `wordmark-outline.svg` |
| Blueprint / technical | `gear-mark-blueprint.svg` | `wordmark-blueprint.svg` |
| Animated / energetic | `gear-mark-motion.svg` | `wordmark-consolidated.svg` |

---

## 5. Construction reference

Anyone extending the gear-mark needs these numbers:

```
viewBox       0 0 800 800
content       transformed to (400, 400) so origin = visual centre
gear teeth    12, evenly spaced (rotate 30°)
              outer extent  r = 345 (tooth tips)
              tooth base    r = 290 (flush with body)
gear body     r = 290 fill
inner face    r = 220 fill (cream / colourway-specific)
inner ring    r = 220 stroke, stroke-width 12
T letter      x=0 y=-10, font-size 270, letter-spacing -4
M letter      x=0 y=185, font-size 230, letter-spacing -4
glyph stroke  stroke-width 14, linejoin=round, linecap=round
```

For the wordmark:

```
viewBox       0 0 1200 320
O'TOWN        x=600 y=155 (centre-anchor),
              font-size 200, stroke-width 12, letter-spacing 8
MAKERSPACE    x=600 y=290 (centre-anchor),
              font-size 130, stroke-width 8, letter-spacing 4
```

The 320 height (originally 300) leaves 20 px of breathing room around the
stroke-padded letters. Don't crop tighter.

---

## 6. Clear-space and minimum size

### Clear-space (exclusion zone)

Reserve a clear margin around the logo equal to **the gear's tooth height**
— 55 units in gear coordinates, or **10% of the symbol's drawn diameter**
in target output. No type, edges, or other graphics inside that zone.

### Minimum size

| Symbol | Minimum width |
|---|---|
| Gear-mark (any variant) | **48 px** on screen, **15 mm** in print. Below this the inner ring stroke and TM letters become illegible. |
| Wordmark | **160 px** on screen, **40 mm** in print. Below this `MAKERSPACE` becomes mush. |

If you need to stamp a smaller mark, use **gear-mark-mono-navy** or
**gear-mark-mono-orange** — single-colour variants survive smaller because
they don't depend on the cream/navy contrast for legibility.

---

## 7. Don'ts

- ✗ Don't recolour the gear — pick the right variant from the list.
- ✗ Don't separate the T from the M, or change their relative sizes.
- ✗ Don't rotate the gear (except via `gear-mark-motion.svg`).
- ✗ Don't apply drop shadows, bevels, or strokes on top of the existing strokes.
- ✗ Don't stretch either piece — width and height scale must stay equal (gear) or proportional (wordmark).
- ✗ Don't place the default cream-bg gear on a coloured surface — switch to `gear-mark-transparent.svg` or the appropriate knockout.
- ✗ Don't use `gear-mark-white.svg`'s baked-in navy bg over a photo — its hard-edge rectangle will fight the image. Use `gear-mark-transparent.svg` (with a white fill swap) or compose against a brand-navy panel.
- ✗ Don't substitute Inter for Inter Display in the wordmark — the heavier display cut is what carries the system.

---

## 8. Compositions

The single-piece logos are the building blocks. The numbered files in
this folder are **ready-to-use compositions** that combine gear and
wordmark into common layout shapes:

| File | Canvas | Purpose | Picks pieces from |
|---|---|---|---|
| `02-horizontal-lockup.svg` | 1600 × 400 (4:1) | Header, business card, email signature, Google Ads landscape logo | `gear-mark-transparent.svg` + `wordmark-consolidated.svg` |
| `03-vertical-lockup.svg` | 800 × 1000 (4:5) | Posters, narrow flyers, Instagram portrait carousels, merch back-print | `gear-mark-transparent.svg` + `wordmark-consolidated.svg` |
| `04-avatar.svg` | 800 × 800 (1:1) | Social profile pictures, app icons, favicons. Centre-safe so platform-applied circular masks don't clip the gear's teeth | `gear-mark.svg` |
| `05-community-badge.svg` | 800 × 800 (1:1) | Stickers, t-shirts, induction certificates, "member since" stamps. Reads as a community seal | `gear-mark-transparent.svg` (centred inside a custom navy/orange ring composition) |
| `06-monochrome.svg` | 800 × 1000 (4:5) | Single-ink production: laser engraving, embroidery, embossing, single-PMS print, etched signage | `gear-mark-outline.svg` + `wordmark-outline.svg` |

These compositions reference their pieces via `<image href="...">` rather
than inlining paths, so a future edit to `gear-mark.svg` or
`wordmark-consolidated.svg` propagates here automatically. Run
`_render/render.sh` to refresh the PNG exports in `_render/`.

If you need a horizontal lockup for a dark surface, duplicate
`02-horizontal-lockup.svg` and swap `wordmark-consolidated.svg` for
`wordmark-white-transparent.svg`. Same recipe for any other knockout
variant — pick the right per-piece file from the variants tables in § 4.

---

## 9. File index

All single-piece SVGs in this folder are **canonical copies of `static/images/branding/logos/`**.
If you edit a file here, mirror the change to `static/` so the deployed
site and the marketing source stay in sync.

The numbered composition files (`02-` through `06-`) are unique to this
folder — they don't exist in `static/`.

```
00-visual-identity/
├── BRAND_GUIDELINES.md             <- you are here
│
│   # compositions (this folder only)
├── 02-horizontal-lockup.svg        gear + wordmark side-by-side
├── 03-vertical-lockup.svg          gear above wordmark
├── 04-avatar.svg                   gear, square, profile-pic ready
├── 05-community-badge.svg          circular sticker emblem
├── 06-monochrome.svg               outline gear + outline wordmark
│
│   # gear-mark variants (mirrored from static/)
├── gear-mark.svg                   default, full colour, cream bg
├── gear-mark-transparent.svg       default, no bg
├── gear-mark-white.svg             knockout, navy bg
├── gear-mark-mono-navy.svg
├── gear-mark-mono-orange.svg
├── gear-mark-black.svg
├── gear-mark-outline.svg
├── gear-mark-blueprint.svg
├── gear-mark-motion.svg
│
│   # wordmark variants (mirrored from static/)
├── wordmark-consolidated.svg       default, two-colour
├── wordmark-white.svg              knockout, navy bg
├── wordmark-white-transparent.svg
├── wordmark-mono-navy.svg
├── wordmark-mono-orange.svg
├── wordmark-black.svg
├── wordmark-outline.svg
└── wordmark-blueprint.svg
```

---

## 10. Where the logos live in the site

- **Source of truth:** `static/images/branding/logos/*.svg`
- **Header / footer reference:** `layouts/_partials/shared/header.html` and `footer.html`
- **OG image (link previews):** `static/images/og-default.jpg` (raster, 1200×630) — regenerate if the wordmark changes.
- **Marketing campaign embeds:** the gear-mark is **inlined** (paths, not `<image>` references) in every SVG under `01-instagram/` … `04-google-ads/` so the files render standalone when uploaded to social platforms. If you change the canonical gear, run a search-and-replace on the inlined polygons across those files.
