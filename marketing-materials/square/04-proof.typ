// marketing-materials/square/04-proof.typ — square proof slide (1080×1080).

#import "_layout.typ": *
#show: slide-setup

#let intro = yaml("/marketing-materials/data/" + lang + "/features/intro.yml")
#let panels = (
  yaml("/marketing-materials/data/" + lang + "/features/panels/01-printing.yml"),
  yaml("/marketing-materials/data/" + lang + "/features/panels/02-laser.yml"),
  yaml("/marketing-materials/data/" + lang + "/features/panels/03-woodcnc.yml"),
  yaml("/marketing-materials/data/" + lang + "/features/panels/04-electronics.yml"),
)

#pad(left: SIZES.pad-x, right: SIZES.pad-x, top: SIZES.pad-top, bottom: SIZES.pad-bottom)[
  #header("04")

  #v(28pt)

  #block(width: 900pt)[
    #text(font: primary-font, size: SIZES.eyebrow, weight: 700, tracking: 0.12em, fill: primary)[#upper(intro.tag)]
    #v(18pt, weak: true)
    #text(font: display-font, size: SIZES.h-l, weight: 700, tracking: -2.5pt)[#intro.headline]
  ]

  #v(32pt)

  #grid(
    columns: (1fr, 1fr),
    column-gutter: 20pt,
    row-gutter: 20pt,
    ..panels.map(p => feature(p.icon, p.title, p.body)),
  )

  #v(1fr)
  #footer-with-arrow()
]
