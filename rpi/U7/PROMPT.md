You own U7 on branch feature/demo-ui. Files: web/, rpi/U7/. This thin UI is allowed
by a written exception to the UI-deferred rule. Use Dino's design note if it
exists. Build against api/openapi.yaml with mock data, then switch to the real API
through one environment variable.

Stack: React, Vite, TypeScript, KaTeX. Target desktop Chrome only.

Tabs
1. Plan: add subjects with weekly hours, level and exam date. Generate and edit
   the week. Each teach block reads "You will teach <concept> on <day>".
2. Teach: a large microphone button using webkitSpeechRecognition. An editable
   live transcript, a LaTeX input with KaTeX preview, and AI student replies read
   aloud with speechSynthesis. Source material is never shown here. If speech is
   unavailable, or the mic is denied, fall back to typing with a clear message.
3. Report: after Finish, show each claim's label, cited correction and quote, new
   gap cards, and the before and after next review date.
4. Review: due concepts and cards, grade buttons, card editing and moving a date.
5. Further Planning: per subject and concept, recall and explanation side by side,
   gaps, next week's hours and the reason for each change.
6. Digest: the daily digest with a Send digest now button.

Technical notes
- EventSource cannot POST, so read the turn stream with fetch and a stream reader.
- Send the corrected transcript, not the raw speech result.
- Show a visible recording state, and a thinking state until the first token.
- Disclose once that Chrome speech may send audio to its provider.
- Show loading, empty and error states on every tab.

The demo must look intentional: clear hierarchy, one accent color, no default
template look. Skip extras that the tabs above do not need.
Test with Playwright: add a subject, generate a plan and finish a typed teach
session against mocks. Report in under 200 words.
