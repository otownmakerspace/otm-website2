// marketing-materials/tokens.typ
// ─────────────────────────────────────────────────────────────────────────────
// Shared palette + font config for O'Town Makerspace ad templates.
//
// Reads:
//   - marketing.toml                        — pinned palette name + fonts
//   - palette/<palette>/<theme>.json        — brand colors per light/dark
//
// otm has no M3 design-token system like esio-website; the brand ships a
// fixed set of colors, so they live in palette/brand/{light,dark}.json with
// a small role set. Mirrors esio's tokens.typ (which read M3 JSON sidecars),
// adapted to otm's brand source.
//
// Theme (light/dark) + language (en/da) selected at render time via Typst's
// --input flags:  typst compile slide.typ out.svg --input theme=dark --input lang=da
// ─────────────────────────────────────────────────────────────────────────────

#let marketing = toml("/marketing-materials/marketing.toml")
#let primary-font = marketing.fonts.primary
#let display-font = marketing.fonts.at("display", default: marketing.fonts.primary)

#let theme        = sys.inputs.at("theme", default: "light")
#let lang         = sys.inputs.at("lang",  default: "en")
#let palette-name = sys.inputs.at("palette", default: marketing.palette)

// Brand colors for the active theme.
#let palette = json("/marketing-materials/palette/" + palette-name + "/" + theme + ".json")

// Keys are kebab-case; access via .at() to avoid hyphen parsing ambiguity.
#let c(k) = rgb(palette.at(k))

#let primary                 = c("primary")
#let on-primary              = c("on-primary")
#let secondary               = c("secondary")
#let on-secondary            = c("on-secondary")
#let surface                 = c("surface")
#let on-surface              = c("on-surface")
#let surface-dim             = c("surface-dim")
#let surface-container-high  = c("surface-container-high")
#let outline-variant         = c("outline-variant")

// Card surfaces: in light mode the dim tint reads better than the container.
#let card-bg = if theme == "light" { surface-dim } else { surface-container-high }
#let card-border = outline-variant
