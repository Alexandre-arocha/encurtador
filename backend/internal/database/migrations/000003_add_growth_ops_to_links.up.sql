ALTER TABLE links
    ADD COLUMN campaign text,
    ADD COLUMN tags text[] NOT NULL DEFAULT '{}',
    ADD COLUMN utm_source text,
    ADD COLUMN utm_medium text,
    ADD COLUMN utm_campaign text,
    ADD COLUMN utm_term text,
    ADD COLUMN utm_content text,
    ADD COLUMN notes text,
    ADD COLUMN updated_at timestamptz NOT NULL DEFAULT now();

CREATE INDEX idx_links_campaign ON links (campaign);
CREATE INDEX idx_links_tags ON links USING gin (tags);
CREATE INDEX idx_links_updated_at ON links (updated_at DESC);
