-- name: CreateLink :one
INSERT INTO links (
    id,
    slug,
    target_url,
    title,
    expires_at,
    is_active,
    campaign,
    tags,
    utm_source,
    utm_medium,
    utm_campaign,
    utm_term,
    utm_content,
    notes
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
RETURNING *;

-- name: GetLinkByID :one
SELECT * FROM links WHERE id = $1;

-- name: GetLinkBySlug :one
SELECT * FROM links WHERE slug = $1;

-- name: ListLinks :many
SELECT
    l.*,
    COALESCE(ld.total_clicks, 0)::bigint AS total_clicks,
    COALESCE(lc.last_clicked_at, '1970-01-01 00:00:00+00'::timestamptz) AS last_clicked_at,
    (lc.last_clicked_at IS NOT NULL)::boolean AS has_last_clicked_at
FROM links l
LEFT JOIN (
    SELECT link_id, sum(clicks)::bigint AS total_clicks
    FROM link_daily
    GROUP BY link_id
) ld ON ld.link_id = l.id
LEFT JOIN (
    SELECT link_id, max(created_at)::timestamptz AS last_clicked_at
    FROM clicks
    GROUP BY link_id
) lc ON lc.link_id = l.id
WHERE (
        sqlc.arg('q')::text = ''
        OR l.slug ILIKE '%' || sqlc.arg('q') || '%'
        OR l.target_url ILIKE '%' || sqlc.arg('q') || '%'
        OR COALESCE(l.title, '') ILIKE '%' || sqlc.arg('q') || '%'
        OR COALESCE(l.campaign, '') ILIKE '%' || sqlc.arg('q') || '%'
        OR array_to_string(l.tags, ' ') ILIKE '%' || sqlc.arg('q') || '%'
    )
  AND (
        sqlc.arg('status')::text = ''
        OR (sqlc.arg('status') = 'active' AND l.is_active = true AND (l.expires_at IS NULL OR l.expires_at > now()))
        OR (sqlc.arg('status') = 'inactive' AND l.is_active = false)
        OR (sqlc.arg('status') = 'expired' AND l.is_active = true AND l.expires_at IS NOT NULL AND l.expires_at <= now())
    )
  AND (sqlc.arg('tag')::text = '' OR sqlc.arg('tag') = ANY(l.tags))
  AND (sqlc.arg('campaign')::text = '' OR l.campaign = sqlc.arg('campaign'))
ORDER BY l.created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountLinks :one
SELECT count(*)::bigint
FROM links l
WHERE (
        sqlc.arg('q')::text = ''
        OR l.slug ILIKE '%' || sqlc.arg('q') || '%'
        OR l.target_url ILIKE '%' || sqlc.arg('q') || '%'
        OR COALESCE(l.title, '') ILIKE '%' || sqlc.arg('q') || '%'
        OR COALESCE(l.campaign, '') ILIKE '%' || sqlc.arg('q') || '%'
        OR array_to_string(l.tags, ' ') ILIKE '%' || sqlc.arg('q') || '%'
    )
  AND (
        sqlc.arg('status')::text = ''
        OR (sqlc.arg('status') = 'active' AND l.is_active = true AND (l.expires_at IS NULL OR l.expires_at > now()))
        OR (sqlc.arg('status') = 'inactive' AND l.is_active = false)
        OR (sqlc.arg('status') = 'expired' AND l.is_active = true AND l.expires_at IS NOT NULL AND l.expires_at <= now())
    )
  AND (sqlc.arg('tag')::text = '' OR sqlc.arg('tag') = ANY(l.tags))
  AND (sqlc.arg('campaign')::text = '' OR l.campaign = sqlc.arg('campaign'));

-- name: SlugExists :one
SELECT EXISTS (SELECT 1 FROM links WHERE slug = $1);

-- name: UpdateLink :one
UPDATE links
SET target_url   = $2,
    title        = $3,
    expires_at   = $4,
    is_active    = $5,
    campaign     = $6,
    tags         = $7,
    utm_source   = $8,
    utm_medium   = $9,
    utm_campaign = $10,
    utm_term     = $11,
    utm_content  = $12,
    notes        = $13,
    updated_at   = now()
WHERE id = $1
RETURNING *;

-- name: DeleteLink :execrows
DELETE FROM links WHERE id = $1;
