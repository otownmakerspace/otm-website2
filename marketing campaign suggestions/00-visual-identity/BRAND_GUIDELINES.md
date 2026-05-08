# O'Town Makerspace — Visual Identity Improvements

## Why a refresh

The current logo set (variants 1–7 in `static/images/branding/`) covers a lot of ground but has three issues holding back the rebrand:

1. **No clear primary mark.** Six wordmarks and one cube — but everything in the wild uses a wordmark. The cube (variant 5) is the strongest, most distinctive asset and should lead.
2. **Colour drift.** The logos use `#3B4BA0` blue and `#FF5722` orange; the website tokens are deeper (`#0D1A63` navy, `#F68048` warm orange). Marketing materials drifting between the two looks like two different brands.
3. **Cold typography.** Arial Black + tight tracking reads industrial. The brief — "friendly, safe, enthusiastic" — wants warmer, more open type.

## What changed

| File | Role | Replaces / supplements |
|---|---|---|
| `01-cube-mark.svg` | Primary icon | Refines `otown_logo_5.svg` |
| `02-horizontal-lockup.svg` | Header / business-card lockup | Replaces `otown_logo_2.svg` over time |
| `03-vertical-lockup.svg` | Tall-format lockup (posters, narrow ads) | New |
| `04-avatar.svg` | Optimised social profile picture | New — needed for round-cropping |
| `05-community-badge.svg` | Stickers, certificates, swag | New — reinforces "safe, friendly community" |
| `06-monochrome.svg` | Single-ink applications | New — needed for embroidery, stamps, etching |

## Specific design decisions

- **Cube M is thicker.** The original M had thin strokes that vanished below ~64 px. The refined version uses 4 quadrilateral strokes with consistent width, readable down to ~32 px.
- **Top-face square is rounded.** Sharp corners read industrial; `rx=6` reads friendly-modern (same logic Apple, Google, GitHub apply).
- **Soft white shine on top face.** Subtle gradient → "this is a real, lit object", not a flat pictogram. Adds energy.
- **Wordmark uses Inter Display.** Same weight (800) as Arial Black but wider apertures and gentler curves. Same impact, more warmth.
- **Tagline line-height is generous.** "MAKE · TINKER · SHARE" is 4-px-tracked and small — supportive, not shouty.

## Type system — Inter & Inter Display

The brand uses **two cuts of the same typeface family**, paired intentionally so the website and the logo feel like one design system.

| Surface | Font | Why |
|---|---|---|
| Web pages — body, UI, paragraphs | **Inter** (regular cut) | Tuned for small sizes; wider apertures and looser default spacing make it readable at 14–18 px |
| Logo wordmark, large headlines, hero text | **Inter Display** (display cut) | Tuned for ≥24 px; tighter spacing and slightly condensed widths punch harder at brand-mark scale |

Inter and Inter Display are designed by the same designer (Rasmus Andersson) and share an identical skeleton — the cuts diverge only in optical-size tuning. Used together they read as one coherent voice; mixing in a third unrelated typeface (e.g. Arial Black, Roboto Mono) is what makes a brand look untidy.

### Where this is currently set up vs missing

- **Inter (regular cut):** ✅ already loaded on the site via `themes/hugo-up-business-main/assets/css/fonts.css` — self-hosted woff2 files, weights 100–900, both styles.
- **Inter Display (display cut):** ⚠️ *not currently loaded.* The wordmark SVG declares `font-family: 'Inter Display', 'Inter', sans-serif`, so today the browser falls back to plain Inter when rendering the SVG inline. To make the wordmark render with the proper display cut, either:
  1. **Add Inter Display to the font loader** — download the woff2 files (rsms.me/inter or @fontsource/inter), add matching `@font-face` declarations to `fonts.css`. Use this if you want the logo to remain *live text* in the DOM.
  2. **Convert text → paths in the final SVG** — open the wordmark in Inkscape, select the text, run `Path → Object to Path`. The glyphs become outlined paths and render identically on any device, no font needed. **Recommended for production logos** — this is what most real-world brand SVGs ship as.

### Stroke-padding × letter-spacing rule

Anywhere the brand uses the rounded-corner stroke trick on text (logo, social-media headlines), this relationship holds:

> **Visual letter-spacing = `letter-spacing` − `stroke-width`**

Each glyph grows by `stroke-width / 2` on every side, so two adjacent letters lose `stroke-width` pixels of gap. To get the same visual spacing as plain text, add the stroke-width back to `letter-spacing`. Example:

| Element | Font-size | Stroke | Letter-spacing | Effective visual spacing |
|---|---|---|---|---|
| Gear-mark TM | 230 | 14 | −4 | tight (no neighbour to crowd) |
| Wordmark O'TOWN | 200 | 12 | +8 | tight modern (≈ −4 visual) |
| Wordmark MAKERSPACE | 130 | 8 | +4 | tight modern (≈ −4 visual) |

If you ever change the stroke-width on rounded-corner text, `letter-spacing` must track the change or letters will collide.

## When to use each

- **Header / website / signage:** `02-horizontal-lockup.svg`
- **Profile pictures / avatars / app icons:** `04-avatar.svg`
- **Posters / event flyers / merch:** `03-vertical-lockup.svg`
- **Stickers / inductions / member kits:** `05-community-badge.svg`
- **Single-ink (embroidery, etching, watermarks):** `06-monochrome.svg`
- **Whenever you need just the icon (favicon, app dock, story stickers):** `01-cube-mark.svg`

## Don'ts

- Don't stretch the cube — it's isometric, distortion breaks the illusion
- Don't put the wordmark on a busy photo without a navy overlay (≥40% opacity)
- Don't recolour the cube outside the harmonised palette
- Don't pair the wordmark with another typeface for the same line of copy
- Don't drop the tagline below 16 px on screen — it becomes illegible
