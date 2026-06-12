// marketing-materials/square/01-hook.typ — square hook slide (1080×1080).

#import "_layout.typ": *
#show: slide-setup

#bg-decoration()

#let hero = yaml("/marketing-materials/data/" + lang + "/hero/meta.yml")
#let parts = hero.tagline.split(", ")
#let humanize(s) = upper(s.slice(0, 1)) + s.slice(1)
#let title-a = humanize(parts.at(0)) + "."
#let title-b = humanize(parts.at(1)) + "."

#pad(left: SIZES.pad-x, right: SIZES.pad-x, top: SIZES.pad-top, bottom: SIZES.pad-bottom)[
  #header("01")

  #v(28pt)

  #block(width: 100%)[
    #text(font: primary-font, size: SIZES.eyebrow, weight: 700, tracking: 0.12em, fill: primary)[#upper(hero.eyebrow)]
    #v(14pt, weak: true)
    #text(font: display-font, size: SIZES.h-xxl, weight: 700, tracking: -3pt)[#title-a]\
    #text(font: display-font, size: SIZES.h-xxl, weight: 700, tracking: -3pt, fill: primary)[#title-b]
    #v(20pt, weak: true)
    #block(width: 820pt)[
      #text(font: primary-font, size: SIZES.lede, weight: 500, fill: muted)[#hero.paragraph]
    ]
    #v(24pt, weak: true)
    #big-arrow()
  ]

  #v(1fr)
  #footer()
]
