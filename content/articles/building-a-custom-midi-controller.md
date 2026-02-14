---
title: "Building a Custom MIDI Controller From Scratch"
date: 2026-02-10
author: "Iustinian Olaru"
description: "A step-by-step walkthrough of designing and building a fully custom MIDI controller using the makerspace's tools — from PCB design to the finished enclosure."
reading_time: 8
tags: ["Electronics", "Tutorial", "Member Project"]
---

It started the way most projects at the makerspace do — with a half-baked idea and a cup of coffee. I wanted a MIDI controller tailored to my live performance setup: something compact, with exactly the knobs and buttons I need, and nothing I don't.

## Why Build Your Own?

Off-the-shelf MIDI controllers are great, but they're designed for everyone. If you have a specific workflow — say, controlling Ableton Live with one hand while adjusting effects with the other — a custom layout makes all the difference. Plus, building it yourself means you understand every part of it, and you can fix or modify it anytime.

## Planning the Layout

I started by sketching the panel layout on paper. I needed:

- 8 rotary potentiometers for mixer channels
- 4 buttons for transport controls (play, stop, record, loop)
- 2 faders for master volume and crossfade
- A small OLED display showing the current preset

Once the layout felt right, I moved to KiCad to design the schematic and PCB.

## The Electronics

The brain of the controller is an **Arduino Pro Micro**, which natively supports USB MIDI — no extra drivers needed. The pots and buttons connect through a multiplexer (CD74HC4067) to save on GPIO pins.

The soldering was done at the electronics bench in the makerspace. Having a proper station with helping hands and good lighting made the fine-pitch work much easier than my kitchen table setup at home.

## The Enclosure

For the enclosure, I laser-cut a front panel from 3mm black acrylic and designed a simple box from 6mm plywood on the laser cutter. The slots and tabs were designed in Inkscape so everything press-fits together, with a bit of wood glue for permanence.

I also 3D-printed custom knob caps in orange PLA to match the makerspace branding — a small touch, but it makes me smile every time I use it.

## Programming

The firmware is straightforward Arduino code using the MIDIUSB library. Each knob sends a CC message on a specific channel, and the buttons toggle on/off. The OLED shows which preset is loaded, switchable with a long-press on one of the buttons.

The code is open source and available on my GitHub if anyone wants to build their own version.

## What I Learned

- **Plan twice, solder once.** I had to redo one section of the PCB because I mixed up two pin assignments. A proper schematic review would have caught it.
- **Use the right tool for the job.** The laser cutter made the enclosure panel look professional in minutes. Doing it by hand would have taken hours and looked worse.
- **Ask for help.** Two members helped me debug a grounding issue that was causing noise on the analog readings. That's the best part of a shared workshop.

## Final Thoughts

The whole project took about three weekends spread over a month. The total component cost was around 35€, which is a fraction of what a comparable commercial controller would cost. More importantly, it does exactly what I need — nothing more, nothing less.

If you're thinking about a similar project, come by the makerspace and take a look at the electronics bench. Everything you need is here.
