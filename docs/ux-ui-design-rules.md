# UX/UI design rules

## 1. Flow order: top-to-bottom, left-to-right

Every screen reads and operates from top to bottom and from left to right.

- Inputs and data selections come first (above or to the left).
- The action that consumes them (Run, Generate, Continue) comes after (below or to the right).
- Never place an action above or left of the data it needs. Otherwise the user can pick something lower/right and then press an action higher/left, and the action runs with stale or missing input.
- Settings that affect a run (engine, mode, format) sit before the run button, in that same order.

Example: on the script outline step, "1. Chủ đề" and "2. Dàn ý" come first and the engine/AI-run bar sits below them.

## 2. Visual style: Neubrutalism

The web-gui uses **Neubrutalism**: flat colour, thick black outline, hard offset shadow (no blur, no gradients). Tokens live in `services/web-gui/src/styles/theme.css`; use them instead of hard-coded values.

- Colour: `--accent` mustard yellow (primary action / active), `--accent2` pink (secondary highlight), `--run` purple (AI running), `--danger`, `--success`, `--warning`. Cream page background, white `--card` surfaces.
- Outline: `--border-w` (2px) solid `--border-color` (near-black) on cards, buttons, inputs.
- Shadow: hard offset `--shadow` (4px 4px 0) for cards and selected items, `--shadow-sm` (2px 2px 0) for small controls.
- Radius: `--radius` (4px), small and square-ish.
- Type: `--font-display` (Space Grotesk) for titles, `--font-body` (Inter) for text, `--font-mono` (Space Mono) for code and prompts.
- Spacing: `--space-xs/sm/md/lg` (8/16/24/32px).
- Selected vs unselected: see `styles/selectable.module.css` (tinted fill, ring, bolder label, check badge). Long labels shrink to fit their box instead of overflowing.

## 3. Animation: slide out / slide in

Anything that shows or hides (collapsible blocks, panels, banners, revealed fields) slides out from where it was triggered and slides back in when dismissed. It never pops in or vanishes instantly.

- Use the shared `.reveal` (enter, ~0.22s) and `.revealOut` (exit, ~0.18s) classes in `services/web-gui/src/styles/glass.module.css`. They grow down from the top edge (`translateY` + `scaleY` + `clip-path`, fade).
- Keep the block mounted during the exit with `usePresence(open)` (`src/hooks/usePresence.ts`): render while `mounted`, add `.revealOut` while `closing`. See `components/Disclosure.tsx`.
- Do not write a new one-off animation for show/hide. Reuse the shared classes so every screen moves the same way.
- Respect `prefers-reduced-motion`: the shared classes already disable animation, and `usePresence` unmounts immediately in that case.
- Keep motion short (under ~0.25s) and subtle: no bounce, no long fades.
