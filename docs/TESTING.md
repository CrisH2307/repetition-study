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

**A1. Install four things**
1. Docker Desktop (from docker.com). Open it and wait until it says it is running.
2. Go (from go.dev/dl). Download the Mac installer and click through it.
3. GitHub Desktop (from desktop.github.com). Sign in with your GitHub account (dodino24).
4. Node.js (from nodejs.org). Download the version marked "LTS" and click through the installer. The page in Part D needs it.

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

```bash
node --version
```

What you should see: a line that starts with `v` and a number, like `v24.1.0`.

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

**The labels are done.** You and your classmate labelled the 20 explanations on your own, agreed on the few differences, and the list is frozen in `eval/data/labels_final.csv`. Nobody changes it, even if a result looks bad. You have nothing more to label.

**Your job now has two parts, both after Cris runs the gate** (the steps Cris follows are in `docs/GATE.md`). Cris will tell you when it is ready.
1. Read the report at `eval/results/gate.md`. It starts with a plain summary for you. Compare each line with the table below.
2. Fill in the citation sheet. Cris will send you `citations_to_rate.csv`. For each row, read the correction and the quote. In the last column, `relevant`, write `yes` if the quote really supports the correction, otherwise `no`. Then send the sheet back to Cris. Do not paste course text anywhere public.

**What counts as a pass.** Cris runs the checker and sends you the report.

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

## Part D. Using the page yourself

Today the page uses a **pretend AI**. It costs nothing and needs no keys. Use Chrome. Take a screenshot at every step and write pass or fail for each one.

If anything does not match "What you should see": copy everything on the screen, send it to Cris, and stop.

### D1. Start the project (two Terminal windows)

**Window 1.** Open Terminal in the project folder (like step A3). Run these one at a time.

```bash
git pull
```

What you should see: a few lines, or `Already up to date.` No error.

```bash
docker compose down -v
```

What you should see: a few lines that end with the word `Removed`, or one line saying there was nothing to remove. Both are fine. This gives you a clean copy of the practice data. It only clears the practice data on your own computer.

```bash
make up migrate seed
```

What you should see: some lines from Docker, then three lines that start with `applied`, then a line that starts with `seeded Fixture course`. No error.

```bash
STUDENT_MODEL=fake CHECKER_MODEL=fake make run
```

What you should see: one line that says `listening on :8080`, and the command keeps running. That is correct. Leave this window open.

**Window 2.** Press Cmd+N for a new Terminal window, go to the project folder again (like step A3), then run these one at a time.

```bash
cd web
```

```bash
npm install
```

What you should see: some lines, ending without the word `ERR`. Warnings are fine.

```bash
API_TARGET=http://localhost:8080 npm run dev
```

What you should see: a line with `http://localhost:5173`. Leave this window open too.

Open Chrome and go to **http://localhost:5173**

### D2. Use the page

| Step | What you do | What you should see |
| --- | --- | --- |
| P1 | Look at the first page | Two topics, "Exponential races" and "Poisson superposition". Each says "New topic: not taught yet" |
| P2 | Click "Start teaching" on Exponential races | A box called "Your explanation". No course notes anywhere on the screen |
| P3 | Type: `The minimum of two exponentials with rates $a$ and $b$ has rate $a-b$.` | Under the box, the preview shows the formulas as proper math, not dollar signs |
| P4 | Click "Send" | "The AI student asks:" shows one question. It never tells you the answer |
| P5 | Click "I am done, check my explanation" | A list of what you wrote, with a tick box "I think this was right" |
| P6 | Tick the box, then click "Show my results" | A results page. It says "You ticked 1 of 1 points as right." The "Next review" line shows one date crossed out and a new date after it |
| P7 | Click "Pick another topic" | You are back on the topic list. Only "Poisson superposition" is left, because Exponential races is now set for review on a later day |
| P8 | Do P2 to P7 again on "Poisson superposition" | Same as above. At the end the list says "There are no topics to practise right now." |

**Expected today, not a bug:** every result says "Not mentioned". That is the pretend checker. Marking the wrong thing with an exact quote from the course notes needs the real AI.

### D10. The checker is turned off

This checks that when the checker is broken, the page says so and no review date moves. It uses a pretend checker that always fails, so it also costs nothing.

**Get ready.** In Window 1, press Ctrl+C to stop the server. Then run these one at a time.

```bash
docker compose down -v
```

```bash
make up migrate seed
```

What you should see: a line that starts with `seeded Fixture course`.

```bash
STUDENT_MODEL=fake CHECKER_MODEL=fake-down make run
```

What you should see: one line that says `listening on :8080`, and the command keeps running. Leave Window 2 as it is.

| Step | What you do | What you should see |
| --- | --- | --- |
| D10a | In Chrome, reload the page | Two topics again, each saying "New topic: not taught yet" |
| D10b | Click "Start teaching" on Exponential races, type the sentence from P3, click "Send" | The AI student asks one question, like before |
| D10c | Click "I am done, check my explanation", tick the box, click "Show my results" | A box that says "We could not check this one. Nothing changed: your review dates are the same as before. Please try again later." There is no "Next review" line and no list of results |
| D10d | Click "Pick another topic" | Both topics are still on the list, and each still says "New topic: not taught yet". That shows no date moved |

**Put it back.** In Window 1, press Ctrl+C, then run:

```bash
STUDENT_MODEL=fake CHECKER_MODEL=fake make run
```

What you should see: `listening on :8080` again.

### Waiting for Cris

Do not test these yet. Cris will tell you when.

| What | Why it waits |
| --- | --- |
| The wrong thing is marked with an exact quote from the course notes and its page | Needs the real AI |
| Explaining a topic again with no mistakes shows no false mistake | Needs the real AI |
| Study cards you can open and edit | Not on the page yet |
| Changing a review date by hand | Not on the page yet |

Also not built yet: accounts, uploading your own notes, the daily summary.

## Part E. Things that can go wrong

Do these after Part D, with both windows still running.

| Step | What you do | What you should see |
| --- | --- | --- |
| E1 | Start a topic and type a formula with a mistake in it, like `$\frac{1}{$` | The preview shows the broken formula as red text. The page does not go blank |
| E2 | In Window 1, press Ctrl+C to stop the server. Then in Chrome, click "Send" | A plain message that starts with "The AI student could not answer just now." Not a blank screen |
| E3 | In Window 1, run `STUDENT_MODEL=fake CHECKER_MODEL=fake make run` again, then reload Chrome | The topic list shows again and the page works. The half-finished session from E2 is gone. That is expected |

For E1 you need a topic on the list. If the list is empty, run the "Get ready" commands from D10 again, but with `CHECKER_MODEL=fake` in the last one.

### Waiting for Cris

These need the real AI, so do not test them yet. Cris will tell you when.

| Problem | What to test |
| --- | --- |
| Missing or wrong key | Start with a wrong key and check that you see a clear message, not a blank screen |
| Slow answers | Count the seconds until the first word and until the results, for 10 turns. These numbers include the AI company's own delay. Do not compare them with tests that used the pretend AI |
| Bad internet | Turn Wi-Fi off and see what the screen does. Keep a screen recording of Part D as a backup |
| Voice (only if it is added later) | We type first. If voice is added, spoken math comes out garbled, so check it. Chrome sends voice to Google, so say that out loud |

## When you are done with Parts D and E

Stop both windows with Ctrl+C. Tell Cris pass or fail for D1, P1 to P8, D10a to D10d and E1 to E3, plus anything that felt confusing. Report each problem the way Part F says.

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
5. Parts D and E now, with the pretend AI. The steps marked "Waiting for Cris" come later. The full run must pass on `qa` before anything goes to the final version.
6. You tell Cris pass or fail. Cris decides what goes to the final version.
