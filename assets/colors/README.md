# Colors

Canonical, platform-agnostic color tokens — shared across the web app (`frontend/src/colors.css`) and
any future mobile clients, same reasoning as `assets/fonts/`.

`tokens.json` is the source of truth: 8 primitive tokens (`bone`, `parchment`, `paper`, `hypha`,
`lichen`, `mycelium`, `bark`, `soil`), each with a `light` and `dark` value, plus computed interaction
states (`hover`/`active`/`focusRing`/`disabled`) for the ones that need them, plus a `semantic` map from
role names (`background.default`, `brand.primary`, `text.secondary`, ...) to a token+state pair.

## Provenance

Base OKLCH values were supplied directly (not derived) — see the `light`/`dark` values in `tokens.json`.
Hex/RGB were computed from OKLCH using the exact CSS Color Module 4 conversion (OKLCH → OKLab → linear
sRGB → gamma-companded sRGB, via the standard Björn Ottosson matrices — same math browsers use natively
for `oklch()`). All 16 base tokens and every derived state land inside sRGB gamut with no clamping.

## Derived states — formulas and why

OKLCH's `L` channel is perceptually uniform, so state changes are expressed as `L`/`C` deltas rather than
picked by eye. Two different rules, because surfaces and interactive fills serve different purposes:

- **Surfaces** (`bone`, `parchment`, `paper` — backgrounds, not controls): hover/active nudge `L` *away
  from the extreme the surface stack sits at in that mode*. Light-mode surfaces are all high-`L`, so
  hover darkens slightly (`L −0.03`, active `L −0.05`); dark-mode surfaces are all low-`L`, so hover
  lightens slightly (`L +0.03`/`+0.05`). This distinguishes a hovered row/card from its resting state
  without fighting the mode's own lightness direction.
- **Interactive fills** (`mycelium`, `lichen` — buttons, active nav, accents): hover/active always
  *darken* (`L −0.05` / `L −0.10`) — the conventional "press" affordance, applied the same way in both
  modes since both tokens sit at a workable mid-range `L` in both modes (verified: no state clips below
  0 or crosses into the surface range).
- **Focus ring**: same `L` as base, `C +0.02`. Deliberately *not* the same as hover — a keyboard user
  tabbing through the UI needs focus to look visually distinct from mouse hover, not identical to it.
- **Disabled**: chroma cut to 25% of base, `L` pulled halfway toward `hypha` (the nearest neutral). Reads
  as "washed out" rather than just a dimmer version of the same color.
- **`hypha` (border) "strong"**: `C +0.03`, `L` pulled 30% toward `bark` (primary text) — for emphasized
  borders (focused input outline, selected card) without introducing a whole new hue.

## Contrast validation

Checked with the standard WCAG relative-luminance formula, not assumed:

| Pair | Light | Dark |
|---|---|---|
| Primary text on default background | 13.45 (AAA) | 13.70 (AAA) |
| Primary text on paper surface | 11.91 (AAA) | 11.90 (AAA) |
| Secondary text on default background | 7.17 (AAA) | 7.81 (AAA) |
| Brand-primary button label | white, 8.26 (AAA) | dark `bone`, 7.92 (AAA) |

**Important asymmetry**: in dark mode `mycelium` is bright (`L 0.75`), so a *light* button label fails
badly (1.73:1) — it needs a *dark* label instead. `tokens.json`'s `contrastOnBrandPrimary` captures this
per-mode (`white` in light mode, dark `bone` in dark mode), matching exactly what MUI calls
`palette.primary.contrastText`. Disabled state contrast (~3.25:1) is intentionally below AA-normal —
WCAG 1.4.3 explicitly exempts inactive/disabled controls from the contrast requirement.

## Regenerating

The conversion is a straightforward OKLCH→sRGB implementation (no external color library needed — see
the matrices in the CSS Color 4 spec / Björn Ottosson's OKLab post if reimplementing). If a base token
changes, rerun the same derivation formulas above against the new value rather than hand-adjusting the
derived states, so the whole ramp stays internally consistent.
