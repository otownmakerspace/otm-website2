// 300×250 — Medium Rectangle
#import "_layout.typ": *
#show: setup-page(300pt, 250pt)

#let hero = yaml("/marketing-materials/data/" + lang + "/hero/meta.yml")
#let parts = hero.tagline.split(", ")
#let humanize(s) = upper(s.slice(0, 1)) + s.slice(1)

#orb(top + right, 80pt, dx: 60pt, dy: -70pt)

#pad(18pt)[
  #image(logo-src, height: 26pt)
  #v(14pt, weak: true)
  #text(font: display-font, size: 26pt, weight: 700, tracking: -0.5pt)[#(humanize(parts.at(0)) + ".")]\
  #text(font: display-font, size: 26pt, weight: 700, tracking: -0.5pt, fill: primary)[#(humanize(parts.at(1)) + ".")]
  #v(1fr)
  #cta-pill(hero.cta)
]
