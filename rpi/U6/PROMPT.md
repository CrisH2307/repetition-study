You own U6 on branch feature/digest. Files: internal/digest/, rpi/U6/.

Build Digester.Today. One digest per user per local day, in America/Toronto, with
per-subject lines: subject, due count and estimated minutes.
Estimated minutes = due cards * 0.5 + scheduled teach-backs * 15. These are
estimates, so label them that way.
Suppress it when the student already has a review_events row today. Store sent
digests. Today's delivery is the in-app digest plus one browser notification
triggered by the UI, with a Send digest now control for the demo.

Test suppression, the America/Toronto day boundary including daylight saving
change, and that a day with nothing due sends nothing. Report in under 200 words.
