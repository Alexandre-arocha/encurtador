CREATE TABLE clicks (
    id          uuid        PRIMARY KEY,
    link_id     uuid        NOT NULL REFERENCES links(id) ON DELETE CASCADE,
    created_at  timestamptz NOT NULL DEFAULT now(),
    ip_hash     text        NOT NULL,
    referrer    text,
    ua_raw      text        NOT NULL,
    device_type text,
    browser     text,
    os          text,
    country     text,
    city        text,
    enriched_at timestamptz
);

CREATE INDEX idx_clicks_link_created_at ON clicks (link_id, created_at DESC);
CREATE INDEX idx_clicks_link_referrer ON clicks (link_id, referrer);
CREATE INDEX idx_clicks_link_device_type ON clicks (link_id, device_type);
CREATE INDEX idx_clicks_link_country ON clicks (link_id, country);

CREATE TABLE link_daily (
    link_id uuid    NOT NULL REFERENCES links(id) ON DELETE CASCADE,
    day     date    NOT NULL,
    clicks  integer NOT NULL DEFAULT 0 CHECK (clicks >= 0),
    PRIMARY KEY (link_id, day)
);
