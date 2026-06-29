DROP INDEX IF EXISTS idx_links_updated_at;
DROP INDEX IF EXISTS idx_links_tags;
DROP INDEX IF EXISTS idx_links_campaign;

ALTER TABLE links
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS notes,
    DROP COLUMN IF EXISTS utm_content,
    DROP COLUMN IF EXISTS utm_term,
    DROP COLUMN IF EXISTS utm_campaign,
    DROP COLUMN IF EXISTS utm_medium,
    DROP COLUMN IF EXISTS utm_source,
    DROP COLUMN IF EXISTS tags,
    DROP COLUMN IF EXISTS campaign;
