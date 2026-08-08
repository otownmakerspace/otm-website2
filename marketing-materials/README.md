# Marketing materials — Typst pipeline

Ad assets for O'Town Makerspace (a universal 1080×1080 square social carousel
and Google Display banners), rendered by [Typst](https://typst.app) from
declarative templates. The whole subsystem is self-contained in this folder —
templates, fonts, brand colors, slide content, the brand logos, and the
scripts all live together.

Modeled on `esio-website/marketing-typst/`, re-skinned with O'Town
Makerspace's brand, copy, and logos. One square asset satisfies every social
feed (Facebook, Instagram, LinkedIn); the `google-ads/` banners stay separate
because their IAB dimensions are mandated by the ad network.

## What lives where

```
marketing-materials/
├── marketing.toml          palette name + font config (pinned)
├── palette/brand/          brand colors per theme — light.json + dark.json
├── tokens.typ              loader — reads marketing.toml + palette JSON into
│                           named constants (primary, surface, on-surface, …)
├── fonts/                  static TTFs Typst loads via --font-path
│                           (Space Grotesk display, Roboto body, Material Symbols)
├── data/                   slide content (YAML), {en, da}
│   ├── en/{hero,pain,how,features,subscribe}/…
│   └── da/…
├── square/                 1080×1080 universal social — 5 slides + _layout.typ
├── google-ads/             5 display-banner sizes + _layout.typ
├── assets/logos/           rasterized logos (PNG, generated — gitignored):
│   ├── logo-light.png      navy+orange lockup (templates, light bg)
│   ├── logo-dark.png       all-white lockup   (templates, dark bg)
│   └── individual/         every brand mark rendered standalone (33 PNGs)
├── export-logos.sh         flattens logo SVGs → assets/logos/ (composed + individual)
├── render.sh               runs export-logos.sh, then compiles every
│                           (format × lang × theme × slide) to SVG + PNG
├── out/                    rendered ads, <lang>/<format>/<theme>/<slide>.{svg,png}
│                           (gitignored — .png is the upload-ready raster)
└── .gitignore              ignores out/ and the generated logo PNGs
```

## Running it

```sh
# one step — exports the logos, then renders all 40 assets
TYPST=~/.local/bin/typst ./render.sh
```

Outputs 40 assets (2 formats × 2 languages × 2 themes × 5 slides), each as
**both** `<slide>.svg` (scalable master) and `<slide>.png` (rasterized,
upload-ready, logo baked in) under `out/<lang>/<format>/<theme>/`. PNGs come
out at exact spec pixels — square at 1080×1080, banners at their IAB sizes —
ready to drag-upload to the ad manager. Set `PPI=144` for 2× (retina) social
PNGs (keep banners at the default 72 for spec-exact pixels). To export just the logos (e.g. after editing a brand SVG):

```sh
./export-logos.sh        # → assets/logos/logo-light.png, logo-dark.png
```

Requires Typst on PATH (or `TYPST=path/to/typst`), plus a rasterizer for the
logos (`inkscape`, `rsvg-convert`, or `magick`).

## The logo step — why it exists

The brand lockups (`frontend/branding/logos/horizontal-lockup-*.svg`)
are *composite* SVGs: they pull the gear mark and wordmark in via nested
`<image href="/images/…">` references. Typst's `image()` can't resolve those
nested hrefs, so it would render an empty logo. `export-logos.sh` flattens
each lockup to a single PNG (hrefs resolved against the Hugo static root),
and the templates embed the PNG. Output is PNG, not JPEG, so the mark keeps
its transparency on both light and dark ad backgrounds.

- `logo-light.png` ← `horizontal-lockup-notagline.svg` (navy + orange)
- `logo-dark.png`  ← `horizontal-lockup-white-transparent.svg`, **with the gear
  swapped to the white variant** (`gear-mark-white.svg`). The shipped
  white-transparent lockup keeps the full-colour gear, whose navy parts vanish
  on a navy panel — the swap makes the whole mark contrast on dark backgrounds.

`export-logos.sh` also renders **every** brand logo SVG standalone to
`assets/logos/individual/<name>.png` (all gear marks, wordmarks, and lockup
variants), so each mark is available as its own raster, not only the two the
templates embed.

## Decisions / how this differs from the esio original

- **Brand colors live in `palette/brand/{light,dark}.json`.** esio sourced
  colors from its M3 design-token JSON; otm ships a fixed brand, so a small
  role set (primary, surface, on-surface, …) is defined directly here.
- **Logos are rasterized PNGs, not flat SVGs.** esio's `logo-blue.svg` /
  `logo-white.svg` were already flat; otm's are composite lockups, hence the
  `export-logos.sh` flattening step.
- **Fonts:** Space Grotesk (headings) + Roboto (body) match the live site
  (`frontend/config/_default/hugo.toml`). Material Symbols (icons) is reused
  from the esio kit. Space Grotesk ships only 500/700 static weights here, so
  headings use 700 (esio used 900 of a different family).
- **Decoration is brand-color orbs**, not character illustrations — otm has
  no character art.

> Note: inkscape mis-renders Typst's 8-digit hex alpha (e.g. `#f680482e`) as
> opaque black, so the translucent orbs look wrong if you preview an output
> SVG through inkscape. They are correct in the SVG itself and in Typst's
> native PNG export — it's an inkscape quirk, not a render bug.
