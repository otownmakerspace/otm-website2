// marketing-materials/square/05-cta.typ — square call-to-action slide (1080×1080).

#import "_layout.typ": *
#show: slide-setup

#bg-decoration()

#let cta = yaml("/marketing-materials/data/" + lang + "/subscribe/meta.yml")

#pad(left: SIZES.pad-x, right: SIZES.pad-x, top: SIZES.pad-top, bottom: SIZES.pad-bottom)[
  #header("05")

  #v(64pt)

  #block(width: 920pt)[
    #text(font: primary-font, size: SIZES.eyebrow, weight: 700, tracking: 0.12em, fill: primary)[#upper(cta.tag)]
    #v(16pt, weak: true)
    #text(font: display-font, size: SIZES.h-xxl, weight: 700, tracking: -3pt)[#cta.headline.primary]\
    #text(font: display-font, size: SIZES.h-xxl, weight: 700, tracking: -3pt, fill: primary)[#cta.headline.emphasis]
    #v(24pt, weak: true)
    #block(width: 780pt)[
      #text(font: primary-font, size: SIZES.lede, weight: 500, fill: muted)[#cta.body]
    ]
    #v(38pt, weak: true)
    #cta-pill(cta.cta.label)
  ]

  #v(1fr)
  #footer()
]
