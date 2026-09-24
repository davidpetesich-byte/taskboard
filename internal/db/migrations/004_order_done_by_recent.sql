-- Done lists the most recently finished ticket first, and the store now places
-- each newly finished ticket on top. No completion time was recorded before
-- this, so rank existing Done tickets once by updated_at, newest first. The
-- stored values are Go time strings with a single local offset, so text
-- comparison orders them chronologically; ties fall back to the ULID id.
UPDATE tickets
SET position = 1000 * (
    SELECT COUNT(*) FROM tickets AS newer
    WHERE newer.status = 'done'
      AND (newer.updated_at > tickets.updated_at
           OR (newer.updated_at = tickets.updated_at AND newer.id > tickets.id))
)
WHERE status = 'done';
