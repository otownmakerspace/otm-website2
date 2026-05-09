# SEO Playbook — Three Tiers, In Order of ROI

A reusable playbook for getting a new static site (Hugo / Astro / similar) ranking
in search. Three tiers, each tier compounds on the previous. **Do them in order.**

| Tier | Time   | Frequency  | What it does                                 |
|------|--------|------------|----------------------------------------------|
| 1    | ~20 min | Once       | Get on the map — search engines know you exist |
| 2    | 2-4 h  | Once       | Technical foundations — every page benefits forever |
| 3    | Ongoing | Weekly/monthly | Content + authority — the actual long-term lever |

The tier system matters because the failure mode is doing Tier 1+2 forever
(they're solvable, satisfying) and never doing Tier 3 (slow, ambiguous). For
B2B SaaS or marketing sites, Tier 3 produces 70-80% of organic growth.

---

# Tier 1 — Discoverability (one-time, ~20 min)

The discoverability layer. Until search engines know your site exists and can
crawl it, no other SEO work matters. Most of the lifetime "SEO benefit" of a
new site comes from getting Tier 1 right on day one.

## Pre-flight: site readiness

- [ ] **Site is live on production domain** (not staging, not preview)
- [ ] **HTTPS works** — `curl -I https://<domain>` returns 200/301/302
- [ ] **Canonical URL meta tag** on every page (`<link rel="canonical">`)
- [ ] **`robots.txt` returns 200 and includes a `Sitemap:` directive** — see fix below
- [ ] **`sitemap.xml` returns 200 and lists real pages** — `curl https://<domain>/sitemap.xml | head`
- [ ] **`hreflang` alternates** on bilingual sites

### Hugo: fix robots.txt to reference the sitemap

Hugo's default `robots.txt` is `User-agent: *` with no rules and **no sitemap reference**. Override at `themes/<theme>/layouts/robots.txt` (or `layouts/robots.txt` in the project):

```
User-agent: *
Allow: /

Sitemap: {{ "sitemap.xml" | absURL }}
```

Confirm `enableRobotsTXT = true` in `hugo.toml`. Push, verify with `curl https://<domain>/robots.txt`.

### Hugo: enable Git-based lastmod

Add to `hugo.toml`:

```toml
enableGitInfo = true
```

Sitemap then includes accurate `<lastmod>` per URL — Google uses this to schedule re-crawls. Without it, every URL shows the build date and looks stale.

## Step 1 — Google Search Console (~10 min)

The source of truth for how Google sees your site. Without it you're flying blind.

1. **Open** [search.google.com/search-console](https://search.google.com/search-console). Sign in with the account that owns the site.
2. Click **Add property** → **Domain** (not "URL prefix"). Domain covers all subdomains.
3. Enter the apex domain: `<your-domain>.tld` (no `https://`, no path).
4. Google shows a TXT record:
   ```
   google-site-verification=<long-random-string>
   ```
5. **Add the TXT record** at your DNS provider:
   - **Record type**: `TXT`
   - **Host/Name**: `@` (or blank — both mean apex)
   - **Value**: full `google-site-verification=...` string
   - **TTL**: default (3600s is fine)
6. Save. Back in Search Console, click **Verify**. Retry after 5 min if DNS hasn't propagated.

Once verified:

7. Sidebar → **Sitemaps** → enter `sitemap.xml` → **Submit**.

**You will not see data immediately.** Google takes 2-7 days to crawl, 2-3 weeks for query data to populate. Don't panic in week one.

## Step 2 — Bing Webmaster Tools (~3 min)

Covers Bing + DuckDuckGo + Yahoo + Ecosia. Bing imports from GSC, no separate verification.

1. **Open** [bing.com/webmasters](https://www.bing.com/webmasters). Sign in.
2. Click **Import** → authorize → select your domain.

Done.

## Step 3 — PageSpeed Insights baseline (~5 min)

Lighthouse score baseline. Catches obvious issues before they compound.

1. **Open** [pagespeed.web.dev](https://pagespeed.web.dev/).
2. Run on three URL types:
   - Homepage: `https://<domain>/`
   - Article/post: `https://<domain>/posts/<slug>/`
   - Bilingual: `https://<domain>/<lang>/`
3. Note any Failing audits.

### Common Lighthouse SEO complaints + how to triage

| Audit                                                       | Severity | Action |
|-------------------------------------------------------------|----------|--------|
| Document does not have a meta description                   | High     | Add `<meta name="description">` per page |
| Image elements do not have `[alt]` attributes               | Medium   | Real `alt` text — `alt=""` only for purely decorative |
| Tap targets are not sized appropriately                     | Medium   | Mobile button/link spacing — usually one CSS fix |
| Background/foreground colors lack contrast                  | Medium   | WCAG AA contrast — fix in design tokens |
| Image elements do not have explicit `width`/`height`        | Low      | Causes layout shift; safe to defer for SVG |
| `<title>` element                                           | High     | Unique per page, keyword-relevant |
| Links do not have descriptive text                          | Low      | Replace "click here" / "read more" |

If SEO < 90, fix the high-severity items before Tier 2. If 95+, move on.

## Step 4 — Social card preview (~3 min)

1. **Open** [opengraph.xyz](https://www.opengraph.xyz/) (or [metatags.io](https://metatags.io/)).
2. Paste homepage URL → confirm:
   - **og:image** loads (not 404, 1200×630)
   - **og:title**, **og:description** correct
   - All five platforms (Twitter, Facebook, LinkedIn, Slack, Discord) render
3. Repeat for one article URL — confirm `og:type=article`, `article:published_time`, `og:image`.

## Tier 1 is done when…

- GSC shows your sitemap status: **Success**
- Bing Webmaster shows the property
- PageSpeed SEO score ≥ 90 across multiple URLs
- opengraph.xyz renders the cards correctly

Within 7-10 days, GSC's **Coverage / Pages indexed** should be non-zero. If still zero, debug: blocked by robots.txt? Sitemap unreachable? Pages returning 4xx?

---

# Tier 2 — Technical Foundations (one-time, ~2-4 h)

Foundational technical SEO. Each item is a one-time setup; every page on the
site (current and future) benefits forever. Order below is rough ROI order.

## Step 1 — Structured data (JSON-LD)

Schema.org JSON-LD tells search engines *what* your pages are about, not just
the words on them. Powers rich results (knowledge panels, article cards with
images and dates, sitelinks, FAQ accordions).

### Organization schema (every page, in baseof head)

Identifies the brand. Powers Google's knowledge panel for brand searches.

```html
<script type="application/ld+json">
{
  "@context": "https://schema.org",
  "@type": "Organization",
  "name": "{{ site.Title }}",
  "url": "{{ site.BaseURL }}",
  "logo": "{{ "assets/logo-blue.svg" | absURL }}",
  "description": "{{ site.Params.description }}",
  "sameAs": [
    "https://www.linkedin.com/company/your-company",
    "https://twitter.com/your_handle"
  ]
}
</script>
```

Drop into a partial included from `<head>`. Wrap with `{{ if .IsHome }}` if you want it only on the homepage (Google accepts either).

### WebSite schema with SearchAction (homepage)

Enables the sitelinks search box in Google results.

```html
<script type="application/ld+json">
{
  "@context": "https://schema.org",
  "@type": "WebSite",
  "url": "{{ site.BaseURL }}",
  "name": "{{ site.Title }}",
  "potentialAction": {
    "@type": "SearchAction",
    "target": "{{ site.BaseURL }}search?q={search_term_string}",
    "query-input": "required name=search_term_string"
  }
}
</script>
```

Skip the SearchAction block if you don't have site search.

### Article / BlogPosting schema (posts and articles)

Powers rich article cards in Google Discover, Top Stories, and SERP.

```html
{{ if and .Date (not .Date.IsZero) }}
<script type="application/ld+json">
{
  "@context": "https://schema.org",
  "@type": "{{ if eq .Section "posts" }}BlogPosting{{ else }}Article{{ end }}",
  "headline": "{{ .Title }}",
  "description": "{{ .Description | default site.Params.description }}",
  "image": "{{ (.Params.image | default site.Params.ogImage) | absURL }}",
  "datePublished": "{{ .Date.Format "2006-01-02" }}",
  "dateModified": "{{ .Lastmod.Format "2006-01-02" }}",
  "author": {
    "@type": "{{ if .Params.author }}Person{{ else }}Organization{{ end }}",
    "name": "{{ .Params.author | default site.Title }}"
  },
  "publisher": {
    "@type": "Organization",
    "name": "{{ site.Title }}",
    "logo": {
      "@type": "ImageObject",
      "url": "{{ "assets/logo-blue.svg" | absURL }}"
    }
  },
  "mainEntityOfPage": "{{ .Permalink }}"
}
</script>
{{ end }}
```

### BreadcrumbList (nested pages)

Powers breadcrumb display in SERPs. Hugo's `.Site.Sections` + `.Parent` chain produces this naturally.

### Validate with [validator.schema.org](https://validator.schema.org/) and [Rich Results Test](https://search.google.com/test/rich-results)

Both accept a URL or pasted HTML. Run after wiring; fix any errors before pushing live.

## Step 2 — Page titles + meta descriptions

These are the literal text shown in search results. They drive clicks. Generic titles waste traffic.

### Title hygiene

- **Unique per page**: never two pages with identical `<title>`
- **Front-loaded keyword**: most-relevant term in the first 60 chars
- **Brand suffix**: `<page-specific-keyword> | <Brand>` is the standard pattern
- **Length**: 50-60 chars displayed; longer titles get truncated
- **No clickbait**: Google demotes pages whose titles don't match content

| Bad                          | Better                                                    |
|------------------------------|-----------------------------------------------------------|
| `Why ESIO \| ESIO`           | `Why ESIO — Budget App for Small Businesses in Denmark \| ESIO` |
| `Welcome`                    | `Welcome to ESIO — Cashflow & Tax Forecasting for Founders` |
| `Article`                    | `Cashflow vs. Profit: Why Profitable Businesses Go Broke \| ESIO` |

### Meta description hygiene

- **Unique per page** — every page should override the site default
- **140-160 chars** — longer gets truncated by Google
- **Include the keyword** — bolded in SERPs when matching the user's query
- **Include a verb / action** — "Learn", "See how", "Discover", "Try"
- **Don't keyword-stuff** — Google rewrites descriptions when they're spammy

Example:

```yaml
title: "Cashflow vs. Profit: Why Profitable Businesses Go Broke"
description: "A profitable business can still go bankrupt. Learn the difference between cashflow and profit, and how to spot the gap before it kills your runway."
```

## Step 3 — Real alt text on images

Two reasons: SEO (alt text is a ranking signal for image search and a context cue for body text), accessibility (screen readers).

- **Decorative images**: `alt=""` (empty, but the attribute must be present)
- **Informative images**: short, factual description — *what* is shown
- **Functional images** (icons in buttons): describe the action, not the image
- **Charts / diagrams**: describe the data conclusion, not the visual

| Bad                                | Better                                              |
|------------------------------------|-----------------------------------------------------|
| `alt="image"`                      | `alt="Woman working on laptop with charts and reports"` |
| `alt="hero-02.svg"`                | `alt="Founder reviewing budget forecast on laptop"` |
| `alt="logo"`                       | `alt="ESIO logo"`                                   |
| `alt="screenshot"`                 | `alt="ESIO dashboard showing 18-month cashflow projection"` |

For Hugo: edit the `data/<lang>/sections/*.yml` files to add real alt text on illustrations. The English `alt` should also be translated in the Danish data file.

## Step 4 — Per-page OG images

Site-default OG image is fine for utility pages. For posts and articles, custom cards drive **2-3× more clicks** from social.

Three approaches:

1. **Manual** (best for low-volume sites): design a template in Figma/Inkscape with the article title as a swappable text layer. Export per article. Path: `static/assets/share/<slug>.jpg`.
2. **Build-time generation**: Node script using `@resvg/resvg-js` + an SVG template that swaps in front-matter values. Wire as a `prebuild` npm script.
3. **Runtime generation**: Vercel OG / Cloudflare Workers. Render on first request, cache forever. Best for sites publishing weekly+.

Front-matter usage (any approach):

```yaml
---
title: "v1.1: Faster forecasts and PDF exports"
date: 2026-04-25
image: /assets/share/v1-1-release.jpg
---
```

The site's OG partial reads `image` and falls back to the site default.

## Step 5 — URL structure

URLs are a ranking signal and a UX signal. Set conventions early; renaming breaks links.

- **Lowercase, hyphenated**: `/posts/cashflow-vs-profit/` not `/posts/CashflowVsProfit/`
- **Short**: 3-5 words max in slug; trim filler words (`the`, `a`, `is`, `your`)
- **Stable**: don't change slugs after publish. If you must, set up a 301 redirect.
- **Descriptive**: `/posts/welcome/` is fine; `/posts/post-1/` is wasted real estate
- **No tracking params indexed**: GSC excludes `?utm_*` correctly, but watch for accidental indexable variants

For Hugo: slug is derived from the filename by default. Set `slug:` in front-matter to override without renaming the file.

## Step 6 — Heading hierarchy

Search engines (and screen readers) use `<h1>`-`<h6>` to understand page structure.

- **Exactly one `<h1>` per page** (the page title)
- **`<h2>` for section headings**, `<h3>` for subsections, etc.
- **Don't skip levels** — `<h1>` → `<h3>` is a Lighthouse warning
- **Don't use heading tags for visual size** — that's what CSS is for

Markdown's `#`, `##`, `###` map naturally. The trap is custom layouts that wrap titles in `<div>` for styling — fix by using semantic tags + Tailwind classes.

## Step 7 — Image optimization

- **Format priority**: SVG (icons/logos) → WebP (photos) → JPG (fallback) → AVIF (cutting edge, smaller). Avoid PNG except for transparency-required UI.
- **Dimensions**: serve at the displayed size, not the source size. A 1200×800 hero photo doesn't need to be 4000×2667.
- **Compression**: JPG at quality 80-85, WebP at 75-80. Lighthouse complains above 100 KB for non-photographic images.
- **Lazy-loading**: `loading="lazy"` on below-the-fold images. The hero / above-fold should be `loading="eager"`.
- **Hugo Pipes**: `{{ $img := resources.Get "..." | resources.Resize "1200x" | resources.WebP }}` for build-time conversion. Requires images in `assets/`, not `static/`.

## Step 8 — Internal linking conventions (start now, refine in Tier 3)

Even basic internal linking, applied consistently, lifts ranking.

- **Descriptive anchor text**: link the keyword, not "click here"
- **Link from new content to existing**: each new article should link to 2-3 older relevant articles
- **Don't over-optimize**: same keyword as anchor 20 times to the same page looks manipulative
- **Footer links don't count for much**: nav and body links carry more weight

## Tier 2 is done when…

- All schema validates without errors at validator.schema.org
- Every page has a unique `<title>` and `<meta description>`
- All informative images have descriptive alt text
- Per-page OG images set up for posts (or template ready for first article)
- URL structure conventions documented in your contributor guide
- Lighthouse SEO score: 100 across all major page types

---

# Tier 3 — Content + Authority (ongoing, the actual lever)

This is where 70-80% of long-term organic growth comes from. The work is slow,
ambiguous, and hard to attribute. Stay disciplined: an hour a week for 12 months
beats 40 hours of sprint work in week 1.

## Step 1 — Keyword research (free tools only)

Find what people actually search for. Don't write articles for keywords no one searches.

| Tool                                    | Cost      | Use for |
|-----------------------------------------|-----------|---------|
| [Google Keyword Planner](https://ads.google.com/aw/keywordplanner) | Free (requires Ads account) | Search volume estimates, keyword variants |
| [Google Trends](https://trends.google.com/trends/) | Free | Seasonal patterns, country comparison |
| [AnswerThePublic](https://answerthepublic.com/) | Free (3/day) | Question-based long-tail keywords |
| [AlsoAsked](https://alsoasked.com/) | Free (3/day) | "People also ask" tree |
| [Ahrefs Free Keyword Generator](https://ahrefs.com/keyword-generator) | Free | Top 100 ideas + difficulty estimate |
| [Ubersuggest](https://neilpatel.com/ubersuggest/) | Free (3/day) | Volume + difficulty (constant upsell, but data is OK) |

### How to use them

1. Brainstorm 5-10 **seed terms** — words your customer would type into Google when they have the problem you solve. For a budgeting app: `cashflow forecasting`, `small business budget app`, `tax deadline calendar`.
2. Run each seed through Keyword Planner and AnswerThePublic.
3. Filter results by **volume** (some searches per month) AND **intent** (informational/transactional/navigational). Skip zero-volume.
4. Group into **topic clusters** — 1 broad pillar keyword + 5-10 specific spoke keywords.
5. Spreadsheet the result. Each row = one future article.

### Pick the long tail, not the head

| Keyword                | Volume | Difficulty | Strategy           |
|------------------------|--------|------------|--------------------|
| `budgeting`            | 50,000 | Very High  | Skip — you can't outrank Investopedia |
| `business budget app`  | 5,000  | High       | Skip — outranking QuickBooks is unrealistic |
| `cashflow forecast danish small business` | 50 | Low | **Write this** — niche, low competition, exact-match intent |

Long-tail beats head-term for new sites. Five articles ranking #1 for low-volume queries beat one article ranking #50 for a head-term.

## Step 2 — Content cadence

Pick a sustainable cadence and hold it. **One thoughtful article every two weeks > five rushed articles in week 1, then nothing**.

| Site stage          | Realistic cadence | Why            |
|---------------------|-------------------|----------------|
| New (0-6 mo)        | 1 article/week   | Build a base; Google needs ~30 articles to take a domain seriously |
| Growth (6-18 mo)    | 1 article every 2 weeks | Quality over quantity; depth wins over breadth |
| Mature (18 mo+)     | 1 article every 3-4 weeks + content refreshes | Maintain + improve existing assets |

Each article should target ONE keyword cluster. Word count varies by topic (don't pad — Google has rejected the "longer is better" myth), but expect 1,000-2,500 words for a substantive piece.

## Step 3 — Topic clusters / pillar pages

Don't write 30 unrelated articles. Group them into clusters around a pillar.

```
Pillar: "Cashflow Forecasting for Small Businesses" (2,500-word comprehensive guide)
  ├─ Spoke 1: "Cashflow vs Profit"
  ├─ Spoke 2: "Forecasting Your First Year Without Data"
  ├─ Spoke 3: "Cashflow Patterns by Industry"
  ├─ Spoke 4: "Building a 13-Week Cash Flow Model"
  └─ Spoke 5: "Why Most Small Businesses Miss Their Cash Forecast"
```

Each spoke article links to the pillar. Pillar links back to spokes. Internal-link density tells Google "this site is an authority on cashflow forecasting" — much stronger signal than 30 disconnected articles.

For each topic cluster you build well, expect to rank well for the *pillar* keyword within 6-12 months even without external backlinks.

## Step 4 — Internal linking (continuous discipline)

Every new article: link to 2-3 older relevant articles. Every older article: edit to link to the new one where the topic fits naturally.

- **Anchor text**: use the target page's keyword, not "click here"
- **Don't force links**: only link where it genuinely helps the reader
- **Audit quarterly**: orphaned pages (no incoming internal links) get less ranking love. Fix them.

Hugo helper: write a small `unlinked.html` partial that lists pages with no incoming links. Or use Screaming Frog (free up to 500 URLs) to crawl + report orphans.

## Step 5 — Backlinks (slowest, highest impact)

Backlinks from other domains are the single largest ranking factor for competitive queries. Building them is sales work, not engineering work.

### What works

- **Quality over quantity**: one link from a relevant high-authority site (TechCrunch, Forbes, an industry trade publication) > 100 links from random blog farms
- **Industry directories**: ProductHunt, BetaList, Indie Hackers, country-specific tech listings
- **Guest posts**: write for established blogs in your industry, link back from author bio
- **Partnerships**: bookkeepers / accountants / consultants who'll list you as a recommended tool
- **Original research / data**: publish unique data and other sites cite you (HARO, "we surveyed 500 founders…")
- **Tool / template giveaways**: free spreadsheets, calculators, templates get linked naturally

### What doesn't work (or actively hurts)

- **Buying links**: Google's algorithms detect this; manual penalties exist
- **Link exchanges**: explicit reciprocal links are devalued
- **Comment spam**: zero ranking benefit since 2010
- **Press release spam**: distributed PR has no SEO value
- **Private blog networks (PBNs)**: algorithm-detected, manual-actionable

### Realistic expectations

- New domain in a competitive niche: 6-12 months before backlinks start mattering
- 5-10 quality links in year 1 is good for a B2B SaaS in a small market
- 50+ links from non-spam sources signals real traction

## Step 6 — Local SEO (if applicable)

If you serve a specific geography (e.g. ESIO targets Denmark):

- **Set up a Google Business Profile** if you have a physical address or service area
- **Mention the country/region** naturally in titles, descriptions, and content (`for Danish small businesses`)
- **Add a `LocalBusiness` schema** in addition to `Organization`
- **Get listed in country-specific directories** (Trustpilot DK, local Chamber of Commerce, industry-specific Danish directories)

Local intent queries (`accountant in Aalborg`, `momsfradrag for SMB`) reward locally-optimized content even with thin overall domain authority.

## Step 7 — Content refreshes (an underrated lever)

Your existing articles age. Google rewards freshness for many query types.

- **Quarterly review**: skim each published article. If facts have changed, update them. If new information exists, add a section.
- **Update `dateModified`**: Hugo's `.Lastmod` from git timestamps does this automatically when `enableGitInfo = true`. The schema you wired in Tier 2 will then signal the refresh.
- **Don't fake refreshes**: changing a comma to update the date is detectable and demoted
- **Consolidate weak articles**: two thin articles on similar topics → one strong article + 301 redirect

A 12-month-old article refreshed and expanded with current data often outranks a brand-new article on the same topic.

## Step 8 — Brand mentions (unlinked references count)

Google's algorithm gives weight to **unlinked brand mentions** — when sites mention your brand name without hyperlinking. This is hard to manufacture but worth noticing.

- **Set up Google Alerts** for your brand name and product name
- **Track mentions** monthly; engage where appropriate (thank, correct, etc.)
- **Mentions in industry forums** (Reddit, Hacker News, niche Slack communities) compound brand recognition

## Tier 3 cadence summary

| Activity                  | Frequency  | Time     |
|---------------------------|------------|----------|
| Write a new article       | Weekly to monthly | 2-4 h |
| Internal-link new content | Per publish | 30 min |
| Keyword research session  | Monthly    | 1 h      |
| Backlink outreach         | Weekly     | 1 h      |
| GSC review (queries, errors) | Monthly | 30 min   |
| Content refresh           | Quarterly  | 4 h      |
| Brand mention tracking    | Monthly    | 15 min   |

If this feels overwhelming, start with one: pick a content cadence and hold it. Everything else is multiplier.

---

# What's NOT covered

Each of these matters for specific cases but isn't universal:

- **AMP**: Google deprecated AMP-priority indexing in 2021. Skip unless you have legacy AMP pages.
- **Performance tuning beyond Lighthouse**: relevant if Core Web Vitals are red. Static sites usually pass without tuning.
- **JavaScript-heavy SPA SEO**: Hugo/Astro/Eleventy avoid this entirely. If you switch to a JS-heavy framework, SSR/SSG becomes important.
- **AI-overview / SGE optimization**: too new, advice changes monthly. Don't optimize for it; just write substantive content.
- **Multi-domain / cross-domain canonicals**: relevant for migrations. Skip otherwise.
- **Non-Google search engines** (Yandex, Baidu, Naver): only relevant if targeting Russia, China, Korea respectively.

---

# Notes for porting to another site

When applying to a new project, things that **change**:

- Domain name (obvious)
- Theme path for `robots.txt` template — `themes/<your-theme>/layouts/`
- Sitemap URL pattern — Hugo defaults to `/sitemap.xml`; Astro/Next vary
- DNS provider UI conventions (`@` vs blank vs apex-domain text)
- Multi-language structure for hreflang and per-language sitemaps
- Schema specifics if site type is different (LocalBusiness, SoftwareApplication, etc.)
- Industry-specific keyword research starting points

Things that **stay the same**:

- The three tiers, in order
- The Tier 1 four-step verification flow (GSC, Bing, PageSpeed, OG preview)
- The "TXT-record over HTML-file" recommendation
- The Lighthouse triage table
- Title / meta-description hygiene rules
- The keyword long-tail-over-head-term strategy
- The content cadence brackets (weekly → biweekly → monthly as the site matures)
- The ratio of Tier 2 effort to Tier 3 effort (small one-time vs. ongoing)

---

# The trap to avoid

People naturally gravitate to Tier 1 + Tier 2 because they're solvable —
discrete tasks with clear "done" states. They become procrastination zones.
Tier 3 feels endless and ambiguous, so people defer it.

For a B2B marketing site in a small market (say, Denmark):

- **Tier 1 done**: brings in maybe 5% of your eventual organic traffic
- **Tier 1 + 2 done**: maybe 15-20%
- **Tier 1 + 2 + 3 (12+ months of consistent content)**: 80-100%

Don't celebrate finishing Tier 2. Tier 2 is the warm-up.
