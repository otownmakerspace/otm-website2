// marketing-materials/facebook/cover.typ — Facebook Page cover (851×315).
//
// Facebook crops the cover differently on desktop (851×315) and mobile
// (~640×360, centred), so the logo + headline are kept to the left/centre
// safe zone; the CTA sits on the right and may be trimmed on mobile feeds —
// acceptable for a page-branding banner.

#import "_layout.typ": *
#show: setup-page(851pt, 315pt)

#let hero = yaml("/marketing-materials/data/" + lang + "/hero/meta.yml")
#let parts = hero.tagline.split(", ")
#let humanize(s) = upper(s.slice(0, 1)) + s.slice(1)

#orb(top + right, 150pt, dx: 90pt, dy: -90pt)
#orb(bottom + left, 120pt, dx: -70pt, dy: 80pt, opacity: 10%)

#pad(x: 48pt, y: 40pt)[
  #image(logo-src, height: 40pt)
  #v(18pt)
  #grid(
    columns: (1fr, auto),
    align: horizon,
    column-gutter: 30pt,
    [
      #text(font: primary-font, size: 13pt, weight: 700, tracking: 0.12em, fill: primary)[#upper(hero.eyebrow)]
      #v(10pt, weak: true)
      #text(font: display-font, size: 38pt, weight: 700, tracking: -1.5pt)[#(humanize(parts.at(0)) + ".")]
      #v(12pt, weak: true)
      #text(font: display-font, size: 38pt, weight: 700, tracking: -1.5pt, fill: primary)[#(humanize(parts.at(1)) + ".")]
    ],
    cta-pill(hero.cta),
  )
]
