# Logo

Canonical logo assets — the mushroom-and-mycelium mark, shared across the web app (`frontend/public/`)
and any future mobile clients, same reasoning as `assets/fonts/` and `assets/colors/`.

- `mark-alpha-mask_1024.png` — the shape itself, as a grayscale alpha mask (opaque = mark, transparent =
  background). This is the actual source of truth going forward, not a flat RGB image, because it's
  color-agnostic: recoloring is "flat-fill a color + apply this mask as opacity," not image editing.
- `mark-mycelium-light_1024.png` / `mark-mycelium-dark_1024.png` — the mask recolored to
  `mycelium` (`#3E543E` light-mode value / `#9EB698` dark-mode value, from `assets/colors/tokens.json`),
  transparent background, full 1024px resolution. Every icon/logo file in `frontend/public/` is resized
  down from one of these two.

The original source (a black-mark-on-white-background PNG, generated externally) was only needed to
derive the alpha mask above and wasn't kept — the mask captures 100% of the shape information, so it's
the only source asset actually worth preserving.

## Recoloring to a different color

The mark is a single flat color with anti-aliased edges (not a gradient or multi-color illustration), so
recoloring loses nothing and stays pixel-perfect at the edges:

```bash
convert -size 1024x1024 xc:"#TARGET_HEX" mark-alpha-mask_1024.png -compose CopyOpacity -composite output.png
```

Do not try to recolor by find-and-replacing the flat RGB image's black pixels — the source has ~1,700
unique colors from anti-aliasing (gray fringe pixels at every edge, not a clean 2-color image), so a
naive color swap leaves visible gray fringing. The mask-based approach above avoids that entirely.

## Where each size/variant is used

Generated via ImageMagick (`convert -resize`) from the two full-res colored marks above — see
`frontend/public/`:

| File | Source | Background | Used for |
|---|---|---|---|
| `favicon.ico`, `favicon-16x16.png`, `favicon-32x32.png` | mycelium-light | transparent | Browser tab icon |
| `apple-touch-icon.png` (180px) | mycelium-light | filled `bone` (`#FAF5EA`) | iOS home screen |
| `android-chrome-192x192.png`, `android-chrome-512x512.png` | mycelium-light | filled `bone` | Android/PWA home screen, manifest |
| `mycorrhizal-logo-light_512.png` / `_192.png` | mycelium-light | transparent | Login page / Settings "About" logo, light mode |
| `mycorrhizal-logo-dark_512.png` / `_192.png` | mycelium-dark | transparent | Same, dark mode (`mycelium` flips dark→bright between modes, same asymmetry as everywhere else in the palette — the mark has to flip too or it's a near-invisible dark shape on a dark background) |

**Known limitation**: the mark's fine root/branch detail doesn't hold up at 16×16/32×32 — it reads as a
soft textured blob rather than a crisp shape at true favicon size. Confirmed and accepted as a tradeoff
(2026-07-28) rather than fixed, since a real fix needs a simplified redesign (e.g. a bolder cap-only
silhouette for tiny sizes), not more image processing on this asset. Everything 180px and up (login logo,
app icons) reads clearly.

App/PWA icons (filled) use a generous margin already present in the source composition (~64% of the
1024px canvas) — comfortably inside the ~80% safe zone maskable icons need, no extra padding was added.
