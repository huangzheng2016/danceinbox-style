# Design profiles

The visual system has two independent light-theme choices and one independent dark toggle. Choose once when scaffolding a project or doing an explicit redesign. Do not choose per page, per feature, or per component.

## Decision workflow

1. For a new project or redesign, offer the shape and surface choices below. If the user has no preference, use `micro + soft-white`.
2. Set the same `data-shape` and `data-surface` attributes on every HTML entry before the application CSS loads. The defaults also apply when the attributes are absent.
3. Record the chosen pair in the project's `AGENTS.md` or `CLAUDE.md` with the routing and infrastructure decisions.
4. Keep `:root.dark` as the only dark palette. Theme switching changes the `dark` class and `localStorage['theme']`; it does not change the selected surface profile.
5. Verify a home page and a list page in light and dark themes. Check text contrast, control alignment, sticky columns, dialogs, and mobile stacking.

Existing projects keep their copied styles until the user explicitly requests a visual upgrade. Do not overwrite their tokens as a side effect of a feature task.

## Shape

The token order is `xs / sm / md / lg / xl / 2xl`.

| `data-shape` | Token values | Intended character |
|--------------|--------------|--------------------|
| `micro` | `2 / 4 / 4 / 6 / 8 / 10px` | Compact console, default |
| `rounded` | `8 / 12 / 16 / 20 / 24 / 28px` | Softer containers and controls |
| `square` | `0 / 0 / 0 / 0 / 0 / 0` | Strict technical layouts |

Shape roles stay fixed across every profile:

- Status and risk pills, inline code, and small code blocks: `--radius-xs`.
- Buttons, inputs, selects, toggles, shell chrome buttons, and icon close buttons: `--radius-sm`.
- Cards, tables, dialogs, shell surfaces, textareas, and detail placeholders: `--radius-md`.
- Loading and standalone empty states: `--radius-lg`.
- Spinners and avatars: `--radius-round`.

`--radius-pill` remains only as a legacy compatibility token. Shared controls do not use it. Pages must not introduce it as a local variant.

## Surface

All surfaces stay flat and opaque. Borders, text, accent, status colors, shadows, and overlay masks are shared across profiles.

| `data-surface` | App, elevated, input, sticky | Strong, aside, header | Soft, sticky header | Input hover |
|----------------|------------------------------|-----------------------|---------------------|-------------|
| `white` | `#ffffff` | `#f5f5f3` | `#e9e9e6` | `#f5f5f3` |
| `soft-white` | `#f8f8f6` | `#efefec` | `#e6e6e2` | `#ffffff` |
| `warm-paper` | `#f5f4f0` | `#ecebe6` | `#e3e2dc` | `#ffffff` |

Selection guide:

- `white`: maximum separation from slightly darker structural surfaces; useful when the product already uses pure white elsewhere.
- `soft-white`: neutral near-white with reduced large-area brightness; the default for dense operational interfaces.
- `warm-paper`: the original warm neutral palette; useful for a softer reading surface. This is a color preference, not a medical or accessibility claim.

Every light profile still uses the same text tokens and requires contrast verification. A surface name does not replace a contrast check.

## Useful starting combinations

These are suggestions, not extra CSS profiles. Any shape can pair with any surface.

| Need | Shape | Surface |
|------|-------|---------|
| Dense admin or operations console | `micro` | `soft-white` |
| Softer workspace | `rounded` | `warm-paper` |
| Strict technical tool | `square` | `white` |

## HTML contract

Set both attributes on `index.html` and every additional application entry, such as `admin.html`:

```html
<html lang="zh-CN" data-shape="micro" data-surface="soft-white">
```

Copy the self-hosted font files from `frontend/assets/fonts/` to `public/fonts/`, including their license files. Preload both WOFF2 files from every HTML entry. Runtime profile switching is intentionally unsupported: it adds a second persistence and hydration contract without improving page-level consistency.

## Invariants

- Profiles change only their declared radius or surface tokens. Components continue to consume semantic tokens and shared classes.
- No page-local `data-shape`, `data-surface`, radius override, surface override, or hardcoded color.
- No additional axes for density, shadow, font, or accent without an explicit design-system change.
- Dark mode remains complete and independent. Light surface attributes must not leak into the dark palette.
- An upgrade to `rounded + warm-paper` preserves the old surface palette and larger radii, but does not recreate the retired pill-control geometry.
