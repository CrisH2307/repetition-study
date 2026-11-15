# UI design (approved)

Approved by Cris and Dino on 2026-11-14. Issue #51.

Prototype: https://claude.ai/artifact/Wf7NtJt8Hm46bJtwsf6oQk (a clickable canvas,
one artboard per screen). All names, dates, quotes and page numbers in it are
sample data. The product name is open (D16). Formulas in the prototype are
hand-styled; the real build renders them with KaTeX through `web/src/Math.tsx`.

## Look

- One accent colour, the blue already in `web/src/style.css` (`#1f5fbf`).
  Background `#fafaf7`, ink `#1d2330`, muted `#566074`.
- Inter for text. Formulas always rendered.
- Text sizes: body 16px, default small text 13px, small notes 10px.
- Claim labels in words, icon and colour:
  Correct (green, tick), Partly right (amber, half circle),
  Not right (red, cross), Not mentioned (grey, dashed, dash).
- Touch targets at least 44px. Visible focus ring.
- No streaks, lives, leaderboards or claims of learning gains.

## Screens

1. **Today.** Heading is the date. One folding card per course with its total
   minutes. Each topic shows name, minutes and "Due <weekday, Mon day>", red and
   bold when due within 2 days ("Due today" for today). One "Start" button for
   both teaching and recall. Side box "Study Time": `1.2 / 3.5 h` per course
   with a bar, plus one 10px reason only when hours changed
   ("Up 1.5 h: 3 gaps on Monday").
2. **Teach.** No main menu. Line: "Course notes are hidden until you finish."
   The AI student's one current question, with a 10px line that it only asks
   questions. Explanation box with live formula preview, a 10px hint on `$`
   formulas, "Speak instead" fills the same box. Send, then
   "I am done, check my explanation". Timer counts up, no limit.
3. **Self-grade.** "How do you feel about this concept?" Per point:
   right, not sure, wrong. Line: "Only for you. Your review dates do not change."
4. **Report (checked).** Next review before and after
   (`due_before` struck through, `due_after` bold) with a 10px reason.
   Your guess next to what the check found. Per claim: label, your words,
   correction, exact quote with page and a page thumbnail, effect on review,
   and "I think this check is wrong".
5. **Report (unchecked).** Amber "Not checked yet" banner: nothing changed,
   the review date stays, the explanation is saved. "Try the check again" and
   "Try later". Each point shows "Not checked" and no corrections.
6. **Course.** Course notes with status (Ready, pages need checking, reading
   the file), with "Only ready notes are quoted in corrections." Study cards
   show Recall and Explanation separately, key ideas in a drop-down, and a
   "Start" button. A "Mistakes" table links each mistake to its passage.
7. **Weekly plan.** Heading is the date range. Hours per course as
   `1.2 / 3.5 h` with a 10px reason. "Schedule:" day by day, today marked.
   "Note:" lists short changes for the week.

## Decisions from review

- Reasons stay, but short (rule: every number explains itself).
- Red for near deadlines is accepted.
- 10px for small notes, 13px default small text.
- Rule reminders (notes hidden, self-grade changes nothing) stay as one line.

## Open, not designed yet

- **Daily notes made automatically after each review.** Deferred. Research
  first how to avoid regenerating every note when one small thing changes,
  for example keep notes per item in a keyed collection and regenerate only
  the item that changed. Unverified whether code alone can write them or a
  model is needed; a model would be new spend and needs Cris.
- **What a dispute does.** The prototype says a person takes a second look and
  the date stays as shown. No decision covers this yet.
- Phone layout was not tested in the prototype.

## Proposed build steps

Each is one issue and one PR into qa. Steps 1 to 3 use only the current API.

1. Shared look: tokens, label pill, buttons, card and folder in
   `web/src/style.css`.
2. Teach and Self-grade: notes-hidden line, three-way self-grade.
3. Report: before and after dates with reason, quote with page, unchecked banner.
4. Today: course folders, due dates with red, Study Time box. Needs due dates
   and minutes from the API; check `/v1/reviews/due` first.
5. Course and study cards, Mistakes table. Needs new API data (contract change,
   ask Cris).
6. Weekly plan. Needs the planner's output in the API (contract change).
7. Playwright run of the full typed loop against the fake model.
