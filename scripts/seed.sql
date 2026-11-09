-- Demo seed: one user and one subject. Concepts and sources come from U1.
-- Safe to run twice.
INSERT INTO users (id, name)
VALUES ('00000000-0000-0000-0000-000000000001', 'Demo Student')
ON CONFLICT (id) DO NOTHING;

INSERT INTO subjects (id, user_id, name, weekly_hours, level, exam_date, weight)
VALUES ('00000000-0000-0000-0000-0000000000a1', '00000000-0000-0000-0000-000000000001',
        'MATH 6602', 4, 'intermediate', '2026-12-15', 1)
ON CONFLICT (id) DO NOTHING;
