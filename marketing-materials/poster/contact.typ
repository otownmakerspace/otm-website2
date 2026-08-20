// marketing-materials/poster/contact.typ — A4 contact-information poster.
//
// Print piece for notice boards / the space's front door: every way to reach
// O'Town Makerspace, one color-coded card per channel (see _layout.typ for
// the channel palette). Copy comes from data/<lang>/contact/meta.yml; the
// contact facts there mirror hugo.toml (the single source of truth).
//
// Mirrors the structure of the site's contact page (email / location / open
// hours cards, then socials), so paper and web stay recognizably the same.

#import "_layout.typ": *
#show: poster-setup

#let d = yaml("/marketing-materials/data/" + lang + "/contact/meta.yml")

// Footer band is placed full-bleed at the page bottom; the padded content
// column above must stay clear of it (band ≈ 30mm tall with its QR). If
// content grows past one page, Typst spills to page 2 and render.sh's SVG
// export errors — that's the fit check (same convention as the square slides).
#pad(x: 16mm, top: 10mm)[
  // Header: brand lockup, then headline row with the gear mark as the
  // second logo — "use the logos", plural.
  #image(logo-src, height: 12mm)
  #v(6mm)
  #grid(
    columns: (1fr, auto), column-gutter: 10mm, align: (left + horizon, right + horizon),
    [
      #text(font: primary-font, size: 12pt, weight: 700, tracking: 0.14em, fill: primary)[#upper(d.eyebrow)]
      #v(9pt, weak: true)
      #text(font: display-font, size: 36pt, weight: 700, tracking: -1pt)[#d.title]
    ],
    image(gear-src, height: 21mm),
  )
  #v(6mm)

  // Color-coded contact channels.
  #contact-card(channel.email, "mail", d.email_label, d.email)
  #v(4mm)
  #contact-card(channel.location, "location_on", d.location_label, d.address)
  #v(4mm)
  #contact-card(channel.hours, "schedule", d.hours_label,
    [#d.hours_public #linebreak() #d.hours_members],
    note: d.hours_note)
  #v(6mm)

  // Socials — QR codes instead of printed links; platform colors as the code.
  // Platform names are brand names, identical in both languages, so they are
  // literal here rather than in the data files.
  #text(font: display-font, size: 15pt, weight: 700, fill: on-surface)[#d.follow_label]
  #v(4mm)
  #stack(
    dir: ltr, spacing: 5mm,
    qr-tile(social.facebook, "thumb_up", "Facebook", d.facebook_url),
    qr-tile(social.instagram, "photo_camera", "Instagram", d.instagram_url),
    qr-tile(social.discord, "forum", "Discord", d.discord_url),
  )
]

#place(bottom, footer-band(d.site, d.site_url, d.tagline, d.scan_hint))
