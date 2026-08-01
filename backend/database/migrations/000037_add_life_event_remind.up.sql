-- Opt-in flag: when true, a recurring yearly reminder is materialised
-- for this life event (T5b). Year-only events cannot opt in since they
-- have no month/day to remind on.
ALTER TABLE life_events ADD COLUMN remind INTEGER NOT NULL DEFAULT 0;
