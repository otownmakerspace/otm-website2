// marketing-materials/square/03-solution.typ — square solution slide (1080×1080).

#import "_layout.typ": *
#show: slide-setup

#let how = yaml("/marketing-materials/data/" + lang + "/how/meta.yml")

#pad(left: SIZES.pad-x, right: SIZES.pad-x, top: SIZES.pad-top, bottom: SIZES.pad-bottom)[
  #header("03")

  #v(28pt)

  #block(width: 920pt)[
    #text(font: primary-font, size: SIZES.eyebrow, weight: 700, tracking: 0.12em, fill: primary)[#upper(how.tag)]
    #v(16pt, weak: true)
    #text(font: display-font, size: SIZES.h-l, weight: 700, tracking: -2.5pt)[#how.headline]
    #v(12pt, weak: true)
    #block(width: 840pt)[
      #text(font: primary-font, size: 18pt, weight: 500, fill: muted)[#how.subtitle]
    ]
  ]

  #v(30pt)

  #stack(spacing: 16pt, ..how.steps.map(s => h-card(s.icon, s.title, s.body)))

  #v(1fr)
  #footer()
]
