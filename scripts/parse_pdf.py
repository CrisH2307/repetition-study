#!/usr/bin/env python3
"""One-off PDF to text with page locators. Never on the request path.

Usage: python3 scripts/parse_pdf.py "content/private/A3 MATH4430_6602.pdf" content/source.md
       python3 scripts/parse_pdf.py --sync content/source.md content/unit.json   (after hand-fixing source.md)

Uses Docling if it is importable, otherwise `pdftotext -layout` (poppler).
Docling (MIT, 2.149.0 on PyPI) pulls in torch, so the fallback is the default today.
Output has one "<!-- page N -->" marker per page. Formulas come out damaged
(fractions, sums and limits split across lines); fix them by hand in source.md.
"""
import json
import subprocess
import sys


def pages_docling(pdf):
    from docling.document_converter import DocumentConverter  # unverified API, not run here

    doc = DocumentConverter().convert(pdf).document
    return [doc.export_to_markdown(page_no=n) for n in sorted(doc.pages)]


def pages_pdftotext(pdf):
    out = subprocess.run(["pdftotext", "-layout", pdf, "-"], check=True, capture_output=True, text=True).stdout
    return out.split("\f")[:-1]  # pdftotext ends every page, including the last, with \f


def normalize(text):
    """Trim lines and collapse spaces so quotes are not broken by layout padding. content/unit_test.go mirrors this."""
    lines = [" ".join(l.split()) for l in text.split("\n")]
    return "\n".join(l for l in lines if l)


def sync(src, unit):
    """Copy normalized source.md into sources[0].text of unit.json."""
    with open(unit) as f:
        data = json.load(f)
    data["sources"][0]["text"] = normalize(open(src).read())
    with open(unit, "w") as f:
        json.dump(data, f, ensure_ascii=False, indent=2)
        f.write("\n")
    print(f"synced {src} into {unit}")


def main(pdf, dest):
    try:
        pages, engine = pages_docling(pdf), "docling"
    except ImportError:
        pages, engine = pages_pdftotext(pdf), "pdftotext"
    with open(dest, "w") as f:
        for n, text in enumerate(pages, 1):
            f.write(f"<!-- page {n} -->\n{text.strip(chr(10))}\n\n")
    print(f"{engine}: wrote {len(pages)} pages to {dest}")


if __name__ == "__main__":
    if sys.argv[1] == "--sync":
        sync(*sys.argv[2:4])
    else:
        main(*sys.argv[1:3])
