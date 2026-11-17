import disputes_to_eval as m

def test_convert():
    src = [
        {"dispute_id": "1", "concept_id": "c", "claim_id": "k", "student_text": "x", "expected_label": "Supported", "outcome": "upheld"},
        {"dispute_id": "2", "concept_id": "c", "claim_id": "k", "student_text": "x", "expected_label": "supported", "outcome": "rejected"},
        {"dispute_id": "3", "concept_id": "c", "claim_id": "", "student_text": "x", "expected_label": "absent", "outcome": "upheld"},
        {"dispute_id": "4", "concept_id": "c", "claim_id": "k", "student_text": "x", "expected_label": "maybe", "outcome": "upheld"},
        {"dispute_id": "1", "concept_id": "c", "claim_id": "k", "student_text": "x", "expected_label": "absent", "outcome": "upheld"},
    ]
    out, skipped = m.convert(src)
    assert out == [{"id": "d-1", "concept_id": "c", "claim_id": "k", "text": "x", "expected": "supported"}]
    assert [s[0] for s in skipped] == ["2", "3", "4", "1"]

if __name__ == "__main__":
    test_convert(); print("ok")
