# UI-BUILD impl log (issue #61)

Steps 1 to 4 of docs/design/ui-design.md, current API only.

Left out because the API cannot supply it:
- Course folders per course: a due topic has no subject id, so there is one folder (named after the course only when there is exactly one).
- Minutes per topic and the "Study Time" box: no per-topic minutes, no hours used this week.
- Reason for a date change (10px line): the schedule event has no plain-words reason.
- Page thumbnail, "effect on review" per claim, "I think this check is wrong": no data or endpoint.
- "Speak instead": voice is not built.
Also: the three-way self-grade is kept on the page only (no endpoint stores it).
Unverified: calling finish again after an unchecked report (retry) works on the fake-down checker; not tried with a real one.
