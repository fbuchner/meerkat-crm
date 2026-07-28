# Fonts

Self-hosted brand/UI fonts, shared across the web app (`frontend/public/fonts/`, WOFF2) and any future
mobile clients (which should use the TTFs here directly rather than duplicating a second copy).

- **EB Garamond** — branding only (the wordmark in the web app's top bar). Weights: Regular (400),
  Medium (500), SemiBold (600). No italic.
- **Source Sans 3** — everything else (all functional UI). Weights: Regular (400), Medium (500),
  SemiBold (600), Bold (700), plus italic for all four.

Both are SIL Open Font License 1.1 (`OFL.txt` in each directory) — free to embed and redistribute,
including in a mobile app bundle.

## Provenance

Source: the variable-font files Google Fonts actually serves, from the
[google/fonts](https://github.com/google/fonts) repo (`ofl/ebgaramond/`, `ofl/sourcesans3/`) — these are
the canonical, production-hinted OFL sources, not a hand-built subset.

Both ship upstream only as variable fonts (a single file covering a weight range via a `wght` axis).
The static per-weight files here were extracted with [fonttools](https://github.com/fonttools/fonttools)'
instancer, which pins the axis to a fixed value without touching glyph coverage (unlike subsetting, this
doesn't drop any characters — full latin/accented-character coverage for all five UI locales is intact).

```bash
pip install fonttools brotli

# --update-name-table is required -- without it, every instanced file keeps the
# variable font's default-instance name internally (e.g. every weight would
# self-report as "Regular"), which breaks font registration on iOS/Android
# (they key off the font's own name table, not the filename).
fonttools varLib.instancer -q --update-name-table \
  -o EBGaramond-Medium.ttf EBGaramond-VF.ttf wght=500

# WOFF2 for the web build (frontend/public/fonts/), from the instanced TTF:
python3 -c "
from fontTools.ttLib import TTFont
f = TTFont('EBGaramond-Medium.ttf')
f.flavor = 'woff2'
f.save('EBGaramond-Medium.woff2')
"
```

To add a weight later: re-run the instancer at the new `wght` value against the original variable font
(re-download from `google/fonts` if it's not kept locally), generate its WOFF2, drop both into the
matching directory here / `frontend/public/fonts/`, and add the `@font-face` block in
`frontend/public/fonts.css`.
