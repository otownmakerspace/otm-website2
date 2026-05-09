---
title: "Byg en skræddersyet MIDI-controller fra bunden"
date: 2026-02-10
author: "Iustinian Olaru"
description: "En trin-for-trin gennemgang af design og bygning af en fuldt skræddersyet MIDI-controller ved hjælp af makerspacets værktøj — fra PCB-design til det færdige kabinet."
reading_time: 8
tags: ["Elektronik", "Tutorial", "Medlemsprojekt"]
---

Det startede, som de fleste projekter i makerspacet gør — med en halvfærdig idé og en kop kaffe. Jeg ville have en MIDI-controller skræddersyet til mit live-performancesetup: noget kompakt, med præcis de knapper og knapper, jeg har brug for, og intet, jeg ikke har.

## Hvorfor bygge din egen?

Standardmæssige MIDI-controllere er gode, men de er designet til alle. Hvis du har en specifik arbejdsgang — for eksempel at styre Ableton Live med den ene hånd, mens du justerer effekter med den anden — gør et tilpasset layout hele forskellen. Desuden betyder det at bygge den selv, at du forstår hver eneste del af den, og du kan reparere eller modificere den når som helst.

## Planlægning af layoutet

Jeg startede med at skitsere panellayoutet på papir. Jeg havde brug for:

- 8 roterende potentiometre til mixerkanaler
- 4 knapper til transportkontroller (play, stop, record, loop)
- 2 fadere til mastervolumen og crossfade
- Et lille OLED-display, der viser det aktuelle preset

Da layoutet føltes rigtigt, gik jeg videre til KiCad for at designe skemaet og printkortet.

## Elektronikken

Hjernen i controlleren er en **Arduino Pro Micro**, som har native USB MIDI-understøttelse — ingen ekstra drivere nødvendige. Potentiometrene og knapperne er forbundet gennem en multiplexer (CD74HC4067) for at spare GPIO-pins.

Lodningen blev udført ved elektronikbænken i makerspacet. At have en ordentlig station med hjælpende hænder og godt lys gjorde det finmaskede arbejde meget lettere end mit køkkenbordsetup derhjemme.

## Kabinettet

Til kabinettet laserskar jeg en frontplade fra 3mm sort akryl og designede en simpel kasse af 6mm krydsfiner på laserskæreren. Rillerne og tapperne blev designet i Inkscape, så alting passer sammen som press-fit, med lidt trælim for permanens.

Jeg 3D-printede også specialdesignede knopper i orange PLA for at matche makerspacets branding — en lille detalje, men den får mig til at smile, hver gang jeg bruger den.

## Programmering

Firmwaren er ligetil Arduino-kode med MIDIUSB-biblioteket. Hver knap sender en CC-besked på en specifik kanal, og knapperne skifter til/fra. OLED'en viser, hvilket preset der er indlæst, som kan skiftes med et langt tryk på en af knapperne.

Koden er open source og tilgængelig på min GitHub, hvis nogen vil bygge deres egen version.

## Hvad jeg lærte

- **Planlæg to gange, lod én gang.** Jeg måtte lave en sektion af printkortet om, fordi jeg byttede om på to pin-tildelinger. En ordentlig gennemgang af skemaet ville have fanget det.
- **Brug det rigtige værktøj til opgaven.** Laserskæreren fik kabinettets panel til at se professionelt ud på få minutter. At gøre det i hånden ville have taget timer og set værre ud.
- **Bed om hjælp.** To medlemmer hjalp mig med at fejlfinde et jordingsproblem, der forårsagede støj på de analoge aflæsninger. Det er den bedste del af et fælles værksted.

## Afsluttende tanker

Hele projektet tog omkring tre weekender fordelt over en måned. De samlede komponentomkostninger var omkring 35€, hvilket er en brøkdel af, hvad en sammenlignelig kommerciel controller ville koste. Endnu vigtigere gør den præcis, hvad jeg har brug for — hverken mere eller mindre.

Hvis du overvejer et lignende projekt, kom forbi makerspacet og tag et kig på elektronikbænken. Alt, hvad du har brug for, er her.
