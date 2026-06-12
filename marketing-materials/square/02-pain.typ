// marketing-materials/square/02-pain.typ — square problem slide (1080×1080).

#import "_layout.typ": *
#show: slide-setup

#let pain = yaml("/marketing-materials/data/" + lang + "/pain/meta.yml")

#pad(left: SIZES.pad-x, right: SIZES.pad-x, top: SIZES.pad-top, bottom: SIZES.pad-bottom)[
  #header("02")

  #v(56pt)

  #block(width: 100%)[
    #text(font: primary-font, size: SIZES.eyebrow, weight: 700, tracking: 0.12em, fill: primary)[#upper(pain.tag)]
    #v(18pt, weak: true)
    #text(font: display-font, size: SIZES.h-xl, weight: 700, tracking: -2.5pt)[#pain.headline.primary]\
    #text(font: display-font, size: SIZES.h-xl, weight: 700, tracking: -2.5pt, fill: primary)[#pain.headline.emphasis]
    #v(28pt, weak: true)
    #block(width: 840pt)[
      #text(font: primary-font, size: SIZES.lede, weight: 500, fill: muted)[#pain.subtitle]
    ]
  ]

  #v(1fr)
  #footer()
]
