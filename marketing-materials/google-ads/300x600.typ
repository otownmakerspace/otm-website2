// 300×600 — Half Page
#import "_layout.typ": *
#show: setup-page(300pt, 600pt)

#let hero = yaml("/marketing-materials/data/" + lang + "/hero/meta.yml")
#let parts = hero.tagline.split(", ")
#let humanize(s) = upper(s.slice(0, 1)) + s.slice(1)

#orb(top + right, 90pt, dx: 70pt, dy: -70pt)
#orb(bottom + left, 80pt, dx: -50pt, dy: 60pt, opacity: 12%)

#pad(24pt)[
  #image(logo-src, height: 30pt)
  #v(36pt)
  #text(font: primary-font, size: 13pt, weight: 700, tracking: 0.12em, fill: primary)[#upper(hero.eyebrow)]
  #v(14pt, weak: true)
  #text(font: display-font, size: 44pt, weight: 700, tracking: -1.5pt)[#(humanize(parts.at(0)) + ".")]
  #v(2pt, weak: true)
  #text(font: display-font, size: 44pt, weight: 700, tracking: -1.5pt, fill: primary)[#(humanize(parts.at(1)) + ".")]
  #v(18pt)
  #text(font: primary-font, size: 15pt, weight: 500, fill: muted)[#hero.paragraph]
  #v(1fr)
  #cta-pill(hero.cta, size: 16pt, pad-x: 18pt, pad-y: 10pt)
]
