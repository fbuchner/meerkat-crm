-- Completing a recurring reminder with reoccur_from_completion = false used to mark it completed
-- for good (9c7e93f, shipped from v0.6.0 to v1.7.0). Nothing else leaves a recurring reminder
-- completed, so every such row is one of those. They come back at the first occurrence after
-- today on their original schedule: an overdue date would send a mail for each at once.
-- Skipped: rows of deleted users, deleted or deceased contacts, rows the user has already replaced
-- with an active reminder, all but the newest of identical ended rows, and dates too old for the
-- step limit below, which would otherwise leave no next date and fail the migration.
UPDATE reminders
SET completed = 0,
    email_sent = 0,
    updated_at = strftime('%Y-%m-%d %H:%M:%S+00:00', 'now'),
    remind_at = (
        WITH RECURSIVE steps(i) AS (
            SELECT 1 UNION ALL SELECT i + 1 FROM steps WHERE i < 5200
        )
        SELECT strftime('%Y-%m-%d %H:%M:%S+00:00', next_at)
        FROM (
            SELECT i, CASE reminders.recurrence
                WHEN 'weekly' THEN datetime(reminders.remind_at, '+' || (i * 7) || ' days')
                WHEN 'monthly' THEN datetime(reminders.remind_at, '+' || i || ' months', 'floor')
                WHEN 'quarterly' THEN datetime(reminders.remind_at, '+' || (i * 3) || ' months', 'floor')
                WHEN 'six-months' THEN datetime(reminders.remind_at, '+' || (i * 6) || ' months', 'floor')
                WHEN 'yearly' THEN datetime(reminders.remind_at, '+' || (i * 12) || ' months', 'floor')
            END AS next_at
            FROM steps
        )
        WHERE date(next_at) > date('now')
        ORDER BY i
        LIMIT 1
    )
WHERE completed = 1
  AND deleted_at IS NULL
  AND recurrence IN ('weekly', 'monthly', 'quarterly', 'six-months', 'yearly')
  AND date(remind_at) > date('now', '-90 years')
  AND contact_id IN (SELECT id FROM contacts WHERE deleted_at IS NULL AND is_deceased = 0)
  AND user_id IN (SELECT id FROM users WHERE deleted_at IS NULL)
  AND NOT EXISTS (
      SELECT 1 FROM reminders AS other
      WHERE other.deleted_at IS NULL
        AND other.id <> reminders.id
        AND other.user_id = reminders.user_id
        AND other.contact_id = reminders.contact_id
        AND other.recurrence = reminders.recurrence
        AND (
            -- An active reminder with the same message, or the same yearly date (a birthday
            -- reminder is matched by date, and its message depends on the language).
            (other.completed = 0 AND (
                other.message = reminders.message
                OR (reminders.recurrence = 'yearly'
                    AND strftime('%m-%d', other.remind_at) = strftime('%m-%d', reminders.remind_at))
            ))
            OR (other.completed = 1 AND other.message = reminders.message AND other.id > reminders.id)
        )
  );
