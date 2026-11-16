# P1-POLISH impl log (issue #113)

Polish only, no new content. Reviewed with the ui-ux-pro-max skill (ux domain: focus, loading, spacing, touch).

Changed (web/src/style.css, plus 3 headings and the loading class in the .tsx files):
- Focus ring: amber #f2a900 (about 1.9:1 on the background, below the 3:1 for focus indicators) to the accent #1f5fbf (about 6:1).
- Heading scale and spacing: h1 26, h2 20, h3 17, h4 16, fixed margins, space between sections, no stray top margin inside cards.
- Teach, Self-grade and Report titles are now h1 like the other screens (they had no h1).
- Button hover states (darker accent, light accent tint for plain), reduced-motion respected.
- Form fields: labels above full-width inputs, white field background, file input padding, inline "New topic" row aligned.
- Mistakes table scrolls inside its box on a phone instead of squeezing to one word per line.
- Start button error shows on its own line. Loading text uses the muted colour.

Not done: Inter is not loaded (no font file is bundled; the stack falls back to system-ui). Loading it needs a self-hosted font (new dependency) or Google Fonts (sends student IPs to Google). Ask Cris.

Tests: npm run build ok; vitest 9/9; Playwright mock 6/6; Playwright full API 2/2 (fresh DB polish_e2e, dropped after).
Shots: rpi/P1-POLISH/shots, made by shots.ts (copy into web/ with a config pointing at it to rerun; SHOT=after).
