# T23 — UI polish: typography, icons, strings

| | |
|---|---|
| **Rating** | 4 — you will use this app daily for years |
| **Size** | M |
| **Depends on** | *soft* — everything with a UI should exist first, hence the late slot |
| **Alpha** | before |
| **Source** | Tier 6 |

## Why this is late despite rating 4

Dependency beats rating. Polishing surfaces that T4, T7, N4, T5 and T1 are still reshaping means doing it
twice. This is the last substantive pre-alpha ticket before [T22](19-T22-legacy-audit.md).

## How to approach it

**This is a method, not a checklist.** Walk the running app flow by flow — contacts, contact detail,
network, notes/inbox, activities, settings, import/export, admin — and fix what reads as unpolished. The
three items below are calibration for what "better" means, not the full scope.

Use the Browser tooling with `.claude/launch.json`'s `frontend-dev` rather than eyeballing source.

### 1. Typography audit

Review which font is used where (headings, body, labels, monospace) and confirm it is consistent and
*intentional* rather than whatever a component happened to inherit. The rebrand established self-hosted
EB Garamond for the wordmark and Source Sans 3 for UI — see `assets/fonts/README.md` and `theme.ts`.

### 2. Icons

The frontend uses `@mui/icons-material` only (confirmed in `package.json` — there is no MDI dependency
yet). Add MDI (`@mdi/js` + `@mdi/react`, Pictogrammers) and use it **where it has a better semantic
match**, without ripping out every MUI icon. Named starting points:

| Surface | Suggested |
|---|---|
| Notes list | `mdi-note-multiple-outline` |
| Add note | `mdi-note-plus-outline` |
| Network / graph page | `mdi-graph-outline` |

Mixing two icon sets is fine if it is *deliberate and consistent per concept* — the same concept must not
appear with two different icons in two places.

### 3. Copy review

Walk each flow and fix labels that do not clearly describe what they do. One concrete known instance: the
Settings page's **"Profile" sub-label doesn't make sense** — it is just Settings, not a distinct Profile
section within it. Needs a clearer label or removal of the redundant sub-naming. Its exact location was
never pre-located; finding it by walking the UI is part of the task.

Remember every string change is **five locale files** with real translations.

## Traps

- Do not restyle in ways that fight `theme.ts`'s OKLCH palette — extend the theme rather than hardcoding
  colours in components. See `assets/colors/README.md`.
- Component tests assert on label text; changing copy breaks them. That is the tests working — update
  them deliberately.
- Dark and light are both supported. Check both.

## Done when

- `npx tsc --noEmit` clean, `npx vitest run` green.
- Every flow walked in a real browser in **both** light and dark, with before/after screenshots for the
  changes made.
- The three calibration items above addressed, plus whatever else the walkthrough turned up — findings
  recorded even where not fixed.
- All five locale files updated for any changed strings.

## Flash implementation notes

### Files to read first
- CLAUDE.md (repo conventions, traps, commands)
- `frontend/src/theme.ts`, `assets/colors/README.md`, `assets/fonts/README.md` (theme/color/font rules)

### Tests you must write before considering it done
- Component tests for any UI changes: follow `MergeContactsDialog.test.tsx` pattern — `afterEach(cleanup)`, mock `fetch` with `vi.stubGlobal`
- If copy changes: update test assertions that reference old label text

### Self-verification checklist
1. `npx tsc --noEmit` clean
2. `npx vitest run` green (ALL tests, not just yours)
3. Walk every flow in a real browser in BOTH light and dark themes
4. All 5 locale files updated for any changed strings

### Common traps
- MUI appends `" *"` to required field labels — tests using `getByLabelText` must account for this
- Do not nest a `<Chip>` inside `<Typography variant="body2">` — invalid HTML, React warns
- Do not hardcode colors in components — extend the OKLCH theme in `theme.ts`
- Component tests need explicit `afterEach(cleanup)` (vitest here has no auto-cleanup)
