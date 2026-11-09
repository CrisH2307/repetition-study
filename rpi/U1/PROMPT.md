You own U1 on branch feature/content-pack. Files: content/, scripts/parse_pdf.py,
eval/data/, rpi/U1/. Do not touch other paths.

Input: content/private/ (Dino drops the file there; it is gitignored) for the MATH unit provided by Dino (test data). Outside pack: none today.

Steps
1. Write scripts/parse_pdf.py in a local virtual environment using Docling. Check
   Docling's license and current API first. Run it once and save text to
   content/source.md with page locators. Inspect the math. If formulas are
   damaged, list them so Cris can fix them by hand.
2. Draft at most 8 concepts with 3 to 5 required claims each in content/unit.json.
   Every claim needs source_id and locator. Show the list to Cris and wait for
   approval before going on.
3. Create content/allowlist.json as an empty list with the fields url, publisher,
   license, reviewed_by and reviewed_at. No outside sources today, because their
   licenses have not been reviewed.
4. Draft 20 student-style explanations in eval/data/explanations.jsonl, mixed
   across concepts: about 10 correct, 5 partly correct, 5 wrong. Do not include
   their labels in that file, and shuffle them.
5. Create eval/data/labels_cris.csv and labels_dino.csv with empty label columns
   (supported, partial, contradicted, absent). Humans fill these separately.
6. Write scripts/check_labels.py to report agreement and list disagreements for
   the two humans to settle in eval/data/labels_final.csv.
7. Write content/seed.sql with demo subjects, one earlier teach-back and review
   history giving the demo concept a next review about 7 days out. Without this
   the demo has no earlier date to show moving.

Done when unit.json validates against internal/domain, every locator resolves,
and labels_final.csv is complete. Report in under 200 words.
