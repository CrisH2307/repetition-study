You own U5 on branch feature/planner. Files: internal/planner/, rpi/U5/.

Build Planner.Generate as a deterministic greedy allocator using 30-minute blocks.

For each subject s:
score = weight * level * urgency * max(0, target - projected_retention)
- target defaults to 0.90
- projected_retention is the mean FSRS retrievability of its concepts at exam_date
- urgency = min(3, 1 + 14 / max(days_to_exam, 1))
- level starts at beginner 1.3, intermediate 1.0, advanced 0.8, and fades as
  evidence grows: level_now = level ^ (1 - min(1, events / 20))

After each assigned block, divide that subject's score by (1 + blocks assigned)
and pick the next highest. Break ties by subject id.
These constants are proposals. Keep them in one config struct.

Reserve 30% of total hours for new material, configurable. Respect each subject's
weekly_hours and never reschedule locked blocks. Place a teach block before
review blocks for its concept, and create it at least a day ahead so the student
expects to teach. Every block stores a plain-English reason.

A subject with no concepts yet is scheduled from weekly_hours and level alone.

FurtherPlanning returns, per subject and per concept:
- recall: share of card reviews rated Good or Easy, plus mean retrievability
- explanation: share of required claims supported in the latest teach-back
- gaps with citations
- next week's hours per subject, and each change from this week with its reason
Never merge recall and explanation into one score. Mark empty data as no data.

Tests: a fixed input produces the same plan every time, weekly hours hold, locked
blocks survive, and worse retention in a subject raises its hours next week.
Report in under 200 words.
