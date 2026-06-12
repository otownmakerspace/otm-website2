// 320×50 — Mobile Banner
#import "_layout.typ": *
#show: setup-page(320pt, 50pt)

#let hero = yaml("/marketing-materials/data/" + lang + "/hero/meta.yml")

#pad(x: 12pt, y: 8pt)[
  #grid(
    columns: (auto, 1fr, auto),
    align: horizon,
    column-gutter: 12pt,
    image(logo-src, height: 24pt),
    text(font: display-font, size: 15pt, weight: 700, tracking: -0.3pt)[
      #text(fill: primary)[#hero.headline.emphasis] #hero.headline.suffix
    ],
    cta-pill(hero.cta, size: 11pt, pad-x: 10pt, pad-y: 6pt),
  )
]
