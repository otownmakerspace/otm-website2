// marketing-materials/facebook/_layout.typ
// Facebook Page cover banner — shared helpers. The cover sets its own page
// size (851×315, the desktop cover-photo spec). Mirrors the structure of
// google-ads/_layout.typ; the logo is the rasterized PNG (export-logos.sh).

#import "../tokens.typ": *

#let setup-page(width, height) = {
  body => {
    set page(width: width, height: height, margin: 0pt, fill: surface)
    set text(font: primary-font, fill: on-surface)
    set par(leading: 0.4em)
    body
  }
}

#let logo-src = if theme == "light" {
  "/marketing-materials/assets/logos/logo-light.png"
} else {
  "/marketing-materials/assets/logos/logo-dark.png"
}

#let muted = on-surface.transparentize(35%)

#let cta-pill(label, size: 18pt, pad-x: 22pt, pad-y: 12pt) = box(
  fill: primary, inset: (x: pad-x, y: pad-y), radius: 999pt,
)[
  #text(font: primary-font, size: size, weight: 700, fill: on-primary)[#label →]
]

#let orb(pos, radius, dx: 0pt, dy: 0pt, opacity: 18%) = place(
  pos, dx: dx, dy: dy,
  circle(radius: radius, fill: primary.transparentize(100% - opacity)),
)
