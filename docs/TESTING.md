# Test plan for Dino

Hi Dino. This page tells you exactly what to do to test our project. You do not need to understand the code. Follow each step in order.

**The golden rule:** if anything does not look like the "What you should see" line, do not try to fix it. Copy everything on your screen, send it to Cris, and stop.

**What we are testing:** a student types an explanation of a topic, a small AI "student" asks questions, a checker finds what the student got wrong and shows the exact quote from the course notes, and a wrong answer makes the review date come sooner. We can show that this loop works. We cannot say that students learn more. Never say that.

These steps are for a Mac. If you use Windows, tell Cris before you start.

## Words used on this page

| Word | Plain meaning |
| --- | --- |
| Terminal | The app where you type commands. Open it with Cmd+Space, type Terminal, press Enter. |
| Command | A line you copy, paste into Terminal, and run by pressing Enter. |
| Pass | What you see matches "What you should see". |
| `qa` | The test copy of our project. You always test on this copy. |

## Part A. Get set up (about 30 minutes)

Do this once.

**A1. Install three things**
1. Docker Desktop (from docker.com). Open it and wait until it says it is running.
2. Go (from go.dev/dl). Download the Mac installer and click through it.
3. GitHub Desktop (from desktop.github.com). Sign in with your GitHub account (dodino24).

**A2. Get the project**
1. In GitHub Desktop choose File, then Clone Repository, then pick `CrisH2307/repetition-study`, then Clone.
2. At the top, click "Current Branch" and choose `qa`.
3. Click "Fetch origin" or "Pull origin" so you have the latest.
4. Write down the "Local path" shown in GitHub Desktop. You need it in the next step.

**A3. Open Terminal inside the project**

In Terminal, type `cd ` (with a space), then drag the project folder from Finder into the Terminal window, then press Enter. Then run:

```bash
git branch --show-current
```

What you should see: the word `qa`. If you see anything else, stop and tell Cris.

**A4. Check the tools work**

```bash
go version
```

What you should see: a line that starts with `go version go1.26` or a higher number.

```bash
docker --version
```

What you should see: a line that starts with `Docker version`.

**A5. Run everything once**

```bash
make up migrate test
```

What you should see: a few lines, ending with lines that start with `ok` or say `no test files`. The word `FAIL` must not appear.

If it fails: copy everything on the screen, send it to Cris, and stop.

**A6. Keys (only when Cris says so)**

Some tests talk to the real AI. Cris will tell you how to get your own keys. Never paste a key into a chat, an email or a document. Keys go only in a file called `.env` inside the project folder.

## Part B. Safety checks you can run now

Run each one from the project folder in Terminal. Write down pass or fail for each. Run each one twice. If the second run gives a different answer, tell Cris.

Important: a line that says `SKIP` is not a pass. Tell Cris which test was skipped.

**B1. All the project's own tests**

```bash
make test
```

What you should see: every line starts with `ok` or says `no test files`. No `FAIL`.

**B2. A stricter version of the same tests**

```bash
go test -race ./...
```

What you should see: the same, no `FAIL`.

**B3. Nobody can rewrite the review history**

Our review history must only ever grow. It has a safety lock. This command tries to break the lock, and it should be refused:

```bash
docker compose exec db psql -U repetition -d repetition -c "TRUNCATE review_events;"
```

What you should see: an error message that contains the words `review_events is append-only`. That error is the pass. If the command works with no error, that is a serious bug: tell Cris right away.

**B4. No secret keys saved in the project history**

```bash
git grep -n -E "AIza[0-9A-Za-z_-]{30,}|sk-ant-[A-Za-z0-9_-]{20,}|sk-or-[A-Za-z0-9_-]{20,}" $(git rev-list --all) -- . ":!docs"
```

What you should see: nothing at all. If lines appear, tell Cris right away and do not share them.

**B5. Setting up twice is safe**

```bash
make migrate
```

What you should see: nothing printed at all (the first time ever it prints three "applied" lines, which is also fine). It must not show an error.

**B6. The AI student never sees the course text**

```bash
go test ./internal/teach/...
```

What you should see: a line that starts with `ok`.

## Part C. The checker gate (the most important test)

This test decides whether the checker is good enough to change the review dates. If it fails, the checker's findings are shown only as suggestions.

**The 20 explanations.** We prepare 20 short student explanations: about 10 correct, 5 partly correct, 5 wrong, each split into small statements. You and one classmate (not Cris, because you know this subject) each give every statement a label, working alone. Cris builds the tools and does not label:
- `supported`: the explanation is correct
- `partial`: partly correct
- `contradicted`: it says something that is wrong
- `absent`: it leaves out something important

Steps:
1. Label your sheet by yourself. Do not look at your classmate's labels. The guide is `eval/data/LABELLING.md`.
2. Cris runs a tool that measures how often you two agree. It must be at least 0.6 (a score from 0 to 1). If it is lower, you two talk about the differences and label again.
3. You and your classmate agree on every remaining difference. The agreed list is saved as `labels_final.csv`.
4. After that file is saved, nobody changes it, even if a result looks bad.

**What counts as a pass.** Cris runs the checker. You read the report at `eval/results/gate.md`. It starts with a plain summary for you.

| Question | Pass |
| --- | --- |
| Of the statements that really are right, how many did the checker wrongly call wrong? | At most 3 in 100 |
| Of the statements that really are wrong, how many did the checker catch? | At least 80 in 100 |
| Of the statements that were left out, how many did the checker notice? | At least 85 in 100 |
| We change 100 statements on purpose (drop a condition, flip a sign). How many did the checker catch? | At least 85 in 100 |
| Does every correction show an exact quote that really is in the course notes? | All of them |
| Is the quote one a person agrees supports the correction? | At least 9 in 10 |
| Same input three times: does the checker give the same answer? | At least 95 times in 100 |
| An explanation that tries to trick the checker (10 tries): does it change a label? | Never |
| The AI student: does it ever say something it was never told? (30 tries) | Never |
| Cost of one session | At most 8 US cents |

Also read every failed example in full, not just the count. If any line fails, stop and decide with Cris. The plan is to use fewer topics and try a different checker model once. If it still fails, we show findings as suggestions only, and we say so.

## Part D. Using the page yourself (when Cris says the page is ready)

Use Chrome. Take a screenshot at every step. Start from a clean copy of the data. Cris will give you the exact command. You do three real sessions on the practice-set topics and write down what felt wrong.

| Step | What you do | What you should see |
| --- | --- | --- |
| D1 | Pick the topic Cris names | The topic opens, and no course notes are visible |
| D2 | Type your explanation. Put in one wrong thing on purpose | Your formulas show properly as you type |
| D3 | Send your explanation | The AI student asks one question at a time and never tells you the answer |
| D4 | Finish the session and tick which statements you think you covered | A report appears |
| D5 | Read the report | The wrong thing is marked, with an exact quote from the course notes and its page |
| D6 | Look at that topic's next review date | The date moved sooner, and you can see before and after |
| D7 | Open the study cards | The mistake made at least one card, and you can edit it |
| D8 | Explain the same topic again with no mistakes | No false mistake appears, and the date does not move sooner |
| D9 | Change a review date by hand | It saves, and the change shows in the history |
| D10 | Cris turns the checker off. Finish a session | The report says "not checked", and no date changes |

D3, D5 and D6 are the heart of the project. If they fail, nothing else matters. Not tested yet because they are not built: accounts, uploading your own notes, the daily summary.

## Part E. Things that can go wrong on stage

| Problem | What to test |
| --- | --- |
| Formulas look wrong | Type a formula and check the preview matches what you meant |
| Voice (only if it is added later) | We type first. If voice is added, spoken math comes out garbled, so check it. Chrome sends voice to Google, so say that out loud |
| Bad internet | Turn Wi-Fi off and see what the screen does. Keep a screen recording of D1 to D9 as a backup. |
| Missing or wrong key | Start with a wrong key and check that you see a clear message, not a blank screen. |
| Slow answers | Count the seconds until the first word and until the report, for 10 turns. These numbers include the AI company's own delay. Do not compare them with tests that used a pretend AI. |

## Part F. How to report a problem

Open a new issue on GitHub (the Issues tab, then "New issue", then "Problem found while testing") and answer the four questions. One problem per issue. You can also send the same four lines to Cris in a message:
1. What I did (steps, in order)
2. What I expected
3. What happened (paste the message or a screenshot)
4. How bad it is: stops the demo, wrong but we can work around it, or only looks bad

## Order of work

1. Part A now.
2. Part B now.
3. Find one classmate who knows this course to label with you. Label your sheet for Part C as soon as the explanations are ready. This takes the longest, so start early.
4. Part C gate run, then stop and decide with Cris.
5. Parts D and E when the page is ready. The full run must pass on `qa` before anything goes to the final version.
6. You tell Cris pass or fail. Cris decides what goes to the final version.
