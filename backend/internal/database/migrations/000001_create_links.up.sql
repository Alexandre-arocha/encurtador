CREATE TABLE links (
    id         uuid        PRIMARY KEY,
    slug       text        NOT NULL UNIQUE,
    target_url text        NOT NULL,
    title      text,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz,
    is_active  boolean     NOT NULL DEFAULT true
);

CREATE INDEX idx_links_created_at ON links (created_at DESC);
