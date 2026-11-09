# Test plan for Dino

Hi Dino. This page tells you exactly what to do to test our project. You do not need to understand the code. Follow each step in order.

**The golden rule:** if anything does not look like the "What you should see" line, do not try to fix it. Copy everything on your screen, send it to Cris, and stop.

**What we are testing:** a student explains a topic out loud, a small AI "student" asks questions, a checker finds what the student got wrong, and a wrong answer makes the review date come sooner and changes next week's plan. We can show that this loop works. We cannot say that students learn more. Never say that.

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
git grep -n -e AIza -e sk-ant -e sk-or $(git rev-list --all)
```

What you should see: nothing at all. If lines appear, tell Cris right away and do not share them.

**B5. Setting up twice is safe**

```bash
make migrate
```

What you should see: nothing printed at all (the first time ever it prints two "applied" lines, which is also fine). It must not show an error.

**B6. The AI student never sees the course text**

This one only works after Cris says the teaching part is added to `qa`.

```bash
go test ./internal/teach/...
```

What you should see: a line that starts with `ok`.

## Part C. The Hour 3 gate (the most important test)

This test decides whether the checker is good enough to change the review dates. If it fails, the checker's findings are shown only as suggestions.

**The 20 explanations.** We prepare 20 short student explanations: about 10 correct, 5 partly correct, 5 wrong. You and Cris each give every one a label, working alone:
- `supported`: the explanation is correct
- `partial`: partly correct
- `contradicted`: it says something that is wrong
- `absent`: it leaves out something important

Steps:
1. Label your sheet by yourself. Do not look at Cris's labels.
2. Then you and Cris sit together and agree on every difference. The agreed list is saved as `labels_final.csv`.
3. After that file is saved, nobody changes it, even if a result looks bad.

**What counts as a pass.** Cris runs the checker. You read the report at `eval/results/gate.md`.

| Question | Pass |
| --- | --- |
| Of the 5 wrong explanations, how many did the checker catch? | At least 4 |
| Of the 10 correct explanations, how many did the checker wrongly call wrong? | At most 1 |
| Every correction shows a quote from the course PDF | All of them |
| Does the AI student ever say something it was never told? (10 trial questions) | Never |

Also read every failed example in full, not just the count. If any line fails, stop and decide with Cris. The plan is to use fewer topics and try a different checker model once. If it still fails, we show findings as suggestions only, and we say so in the demo.

## Part D. The full demo run (later, when the screens are ready)

Cris will tell you when the screens are ready. Use Chrome. Take a screenshot at every step.

Start from a clean copy of the data. Cris will give you the exact command.

| Step | What you do | What you should see |
| --- | --- | --- |
| D1 | Add two school subjects with weekly hours, level and exam date | A weekly study plan appears |
| D2 | Look at the plan | Each study block shows a "teach it" session coming up |
| D3 | Pick the topic Cris names, explain it out loud, and say one wrong thing on purpose | The AI student asks one question at a time and never corrects you |
| D4 | Finish the session | A report that shows the mistake, with a quote from the course PDF |
| D5 | Look at that topic's next review date | The date moved sooner, and you can see the change |
| D6 | Open the study cards | The mistake created at least one card, and you can edit it |
| D7 | Open "Further Planning" | Remembering and explaining are shown as two separate scores, and next week's hours changed |
| D8 | Open the daily summary | One line per subject with how many reviews are due and how many minutes |
| D9 | Change a review date by hand | It saves, and the change shows in the history |
| D10 | Explain the same topic again with no mistakes | No false mistake appears, and the date does not move sooner |

D3, D4 and D5 are the heart of the project. If they fail, nothing else matters.

## Part E. Things that can go wrong on stage

| Problem | What to test |
| --- | --- |
| Spoken math comes out garbled | Say a formula out loud and read what the screen wrote. Practice using the edit box and the formula box. On stage, explain the idea in words. |
| Microphone | Use Chrome only. Allow the microphone on the demo laptop and test it. Chrome sends the voice to Google to turn it into text, so say that out loud in the demo. |
| Bad internet | Turn Wi-Fi off and see what the screen does. Keep a screen recording of D1 to D9 as a backup. |
| Missing or wrong key | Start with a wrong key and check that you see a clear message, not a blank screen. |
| Slow answers | Count the seconds until the first word and until the report, for 10 turns. These numbers include the AI company's own delay. Do not compare them with tests that used a pretend AI. |

## Part F. How to report a problem

Send Cris one message per problem, with these four lines:
1. What I did (steps, in order)
2. What I expected
3. What happened (paste the message or a screenshot)
4. How bad it is: stops the demo, wrong but we can work around it, or only looks bad

## Order of work

1. Part A now.
2. Part B now. B6 waits until Cris says the teaching part is in `qa`.
3. Label your sheet for Part C as soon as the explanations are ready. This takes the longest, so start early.
4. Part C gate run, then stop and decide with Cris.
5. Parts D and E when the screens are ready. The full run must pass on `qa` before anything goes to the final version.
6. You tell Cris pass or fail. Cris decides what goes to the final version.
