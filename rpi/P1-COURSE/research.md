# P1-COURSE research
- poppler 26.04: `pdftotext -layout -f 1 -l 201 file -` ends each page with a form feed. `pdftoppm` flag is `-singlefile` (not `-singlepage`).
- Text is read in one pdftotext call (not one per page); images use one pdftoppm call per page, 80 dpi PNG.
- `domain.CourseStore` has no GetClaim and Subject has no concepts, so EditClaim cannot find a claim's course.
- `SetPageNotReady` (store contract) already refuses while a live claim anchors to the page; `fakes.CourseStore` does the same.
