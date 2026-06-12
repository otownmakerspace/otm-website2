// 728×90 — Leaderboard
#import "_layout.typ": *
#show: setup-page(728pt, 90pt)

#let hero = yaml("/marketing-materials/data/" + lang + "/hero/meta.yml")

#orb(top + right, 60pt, dx: 40pt, dy: -40pt)

#pad(x: 22pt, y: 16pt)[
  #grid(
    columns: (auto, 1fr, auto),
    align: horizon,
    column-gutter: 22pt,
    image(logo-src, height: 34pt),
    text(font: display-font, size: 24pt, weight: 700, tracking: -0.5pt)[
      #text(fill: primary)[#hero.headline.emphasis] #hero.headline.suffix
    ],
    cta-pill(hero.cta, size: 15pt),
  )
]
