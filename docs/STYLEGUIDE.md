# Style guide

The page is one HTML template (`web/templates/index.html`), one stylesheet (`web/static/css/styles.css`) and a few vanilla ES modules (`web/static/scripts/`). No framework, no bundler, no vendored libraries.

## Identity
- **Accent:** emerald, the same in both themes: `--accent` `#10b981`, `--accent-bright` `#34d399`, `--accent-deep` `#0d9488`. Text on the accent uses `--accent-contrast` `#06231b`. Primary buttons and the brand mark use the bright → accent → deep gradient.
- **Font:** Inter (Google Fonts) with the system UI stack as fallback; `--font-mono` for code and format IDs.
- **Shape:** rounded cards (`--radius-xl`), inputs and buttons `--radius-lg`, 44 px minimum touch targets.

## Themes
- The theme is `data-theme="light|dark"` on `<html>`. An inline script in the template sets it before first paint: a stored choice (`localStorage.theme`) wins, otherwise `prefers-color-scheme`.
- Components only use **semantic tokens**, never raw colours: `--surface`, `--surface-raised`, `--surface-border`, `--input-bg`, `--input-border`, `--track`, `--text`, `--text-secondary`, `--text-muted`, `--label`, `--accent-text`, `--error-text`, `--error-soft`, … Light values live in `:root`, dark overrides in `:root[data-theme="dark"]`. A new colour means a new token in both blocks.
- Status colours: `--success`, `--error`, `--warning`, `--info`.

## Layout
- Mobile first: one column, max 720 px wide, works at 375 px with no horizontal scroll. The `640px` breakpoint only adds padding; `420px` puts the start and end fields side by side.
- Sections of the form are `<fieldset class="section">` with a `.field-label` legend.

## Components
| Class | Use |
|---|---|
| `.button-primary` / `.button-secondary` / `.button-sm` | Main action / secondary action / compact |
| `.chip-button` | Small inline action next to an input ("Now") |
| `.segmented` + `.segment` | Radio cards (Video/Audio preset, streams) |
| `.format-list` + `.fmt` | Advanced format radio list |
| `.trimmer` | Two-handle range slider (`web/static/scripts/trimmer.js`) |
| `.field-error` | Inline validation message under a field (`role="alert"`) |
| `.job` | Job status panel; `data-state` = `queued`, `processing`, `completed`, `error` |
| `.toast` | Transient notice (`web/static/scripts/toast.js`); prefer inline errors for things the user can fix |

## Accessibility
- Every control has a label; the trimmer handles are `role="slider"` with `aria-valuenow`/`aria-valuetext` and keyboard support.
- Visible focus via `:focus-visible` (accent outline). Respect `prefers-reduced-motion`.
