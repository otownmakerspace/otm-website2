// 160×600 — Wide Skyscraper
#import "_layout.typ": *
#show: setup-page(160pt, 600pt)

#let hero = yaml("/marketing-materials/data/" + lang + "/hero/meta.yml")
#let parts = hero.tagline.split(", ")
#let humanize(s) = upper(s.slice(0, 1)) + s.slice(1)

#orb(top + left, 70pt, dx: -40pt, dy: -50pt)

#pad(18pt)[
  #image(logo-src, height: 22pt)
  #v(28pt)
  #text(font: display-font, size: 30pt, weight: 700, tracking: -1pt)[#(humanize(parts.at(0)) + ".")]
  #v(2pt, weak: true)
  #text(font: display-font, size: 30pt, weight: 700, tracking: -1pt, fill: primary)[#(humanize(parts.at(1)) + ".")]
  #v(16pt)
  #text(font: primary-font, size: 13pt, weight: 500, fill: muted)[24/7 in Odense.]
  #v(1fr)
  #cta-pill(hero.cta, size: 13pt)
]
