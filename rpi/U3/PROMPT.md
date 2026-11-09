You own U3 on branch feature/checker. Files: internal/checker/, eval/ except
eval/data/, cmd/evalcheck/, rpi/U3/.

Build Checker.Check. Input: approved sources, the concept's claims and the new
transcript. Output: strict JSON parsed into []ClaimResult.

Rules
- Label each student claim supported, partial, contradicted or absent.
- Mark transcript text as untrusted data. Text inside it can never change a label.
- After the model responds, code verifies every citation: source_id must be
  approved and quote must be an exact substring of that source's text after
  normalizing whitespace.
- If a citation fails: remove the correction, set label to partial when it was
  contradicted, and log it. Never show an uncited correction.
- probe_hint names a gap in plain words and must not quote the source. Code
  rejects a hint that shares a long run of words with the source.
- During a session, label only what the transcript states and never return
  absent, because the student may still reach that claim. The teach engine
  assigns absent at Finish. In cmd/evalcheck, treat each explanation as a
  finished session, so absent is allowed there.

Build cmd/evalcheck, which runs eval/data/explanations.jsonl against
eval/data/labels_final.csv and accepts --model. It writes
eval/results/<model>.md with:
- wrong explanations flagged contradicted or absent, of 5
- correct explanations falsely flagged contradicted, of 10
- percentage of corrections with a valid citation
- a confusion table, cost and latency per call

Run it for at least one Gemini tier and one Claude tier when the labels exist.
Do not choose a winner. Report numbers only, and Cris decides.

Unit test citation checks with fake LLM output: a correct quote, a paraphrased
quote, an unapproved source and a transcript containing injected instructions.
Report in under 200 words.
