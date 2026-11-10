-- Demo history for the concept "Exponential races" (00000000-0000-0000-0000-000000000101).
-- Load content/seed_pack.sql first. Safe to run twice (rows keep their first timestamps).
--
-- Why: without an earlier date there is nothing to show moving. This gives the concept
-- 3 earlier reviews and a next review 7 days from now.
-- Values were computed with go-fsrs v4.0.0 default parameters and short-term steps off,
-- the same setup as internal/srs. Gaps between reviews are 1 and 3 days, then 8 days,
-- close to the 1, 2, 3, 4, 5, 7 day rhythm in the "Study - F26" sheet.
--   5 days ago  teach_back  again   next review 1 day later   (the gap found: claim 3)
--   4 days ago  card_review good    next review 3 days later
--   1 day ago   card_review good    next review 8 days later  = 7 days from now

INSERT INTO cards (id, concept_id, claim_id, front, back) VALUES
 ('00000000-0000-0000-0000-000000000301', '00000000-0000-0000-0000-000000000101', '00000000-0000-0000-0000-000000020101',
  'Two independent exponential waiting times have rates lambda 1 and lambda 2. What is the chance the first one happens before the second?',
  'Lambda 1 divided by (lambda 1 plus lambda 2).'),
 ('00000000-0000-0000-0000-000000000302', '00000000-0000-0000-0000-000000000101', '00000000-0000-0000-0000-000000020102',
  'What is the distribution of the smaller of two independent exponential times with rates lambda 1 and lambda 2?',
  'Exponential with rate lambda 1 plus lambda 2.'),
 ('00000000-0000-0000-0000-000000000303', '00000000-0000-0000-0000-000000000101', '00000000-0000-0000-0000-000000020103',
  'How do you find P(X1 < X2) when X1 and X2 are independent exponential service times?',
  'Condition on X1 = t. Then X2 must be bigger than t, which has probability e^(-lambda 2 t). Integrate lambda 1 e^(-lambda 1 t) times that, which gives lambda 1 / (lambda 1 + lambda 2).')
ON CONFLICT (id) DO NOTHING;

INSERT INTO teach_sessions (id, concept_id, status, started_at, finished_at) VALUES
 ('00000000-0000-0000-0000-000000000401', '00000000-0000-0000-0000-000000000101', 'finished',
  now() - interval '5 days 10 minutes', now() - interval '5 days')
ON CONFLICT (id) DO NOTHING;

INSERT INTO concept_schedules (concept_id, due_at, stability, difficulty, state, reps, lapses, last_review_at) VALUES
 ('00000000-0000-0000-0000-000000000101', now() + interval '7 days', 7.638231, 6.390941, 2, 3, 0, now() - interval '1 day')
ON CONFLICT (concept_id) DO NOTHING;

INSERT INTO review_events (id, event_type, concept_id, card_id, grade, due_before, due_after, stability_before, stability_after, payload, created_at) VALUES
 ('00000000-0000-0000-0000-000000000501', 'teach_back', '00000000-0000-0000-0000-000000000101', NULL, 'again',
  now() - interval '5 days', now() - interval '4 days', 0, 0.212,
  '{"results": 3, "gap_cards": ["00000000-0000-0000-0000-000000000303"]}', now() - interval '5 days'),
 ('00000000-0000-0000-0000-000000000502', 'card_review', '00000000-0000-0000-0000-000000000101', '00000000-0000-0000-0000-000000000301', 'good',
  now() - interval '4 days', now() - interval '1 day', 0.212, 1.886788,
  '{}', now() - interval '4 days'),
 ('00000000-0000-0000-0000-000000000503', 'card_review', '00000000-0000-0000-0000-000000000101', '00000000-0000-0000-0000-000000000302', 'good',
  now() - interval '1 day', now() + interval '7 days', 1.886788, 7.638231,
  '{}', now() - interval '1 day')
ON CONFLICT (id) DO NOTHING;
