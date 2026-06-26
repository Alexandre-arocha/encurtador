-- name: CreateLink :one
INSERT INTO links (id, slug, target_url, title, expires_at, is_active)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetLinkByID :one
SELECT * FROM links WHERE id = $1;

-- name: GetLinkBySlug :one
SELECT * FROM links WHERE slug = $1;

-- name: ListLinks :many
SELECT * FROM links
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountLinks :one
SELECT count(*) FROM links;

-- name: SlugExists :one
SELECT EXISTS (SELECT 1 FROM links WHERE slug = $1);

-- name: UpdateLink :one
UPDATE links
SET target_url = $2,
    title      = $3,
    expires_at = $4,
    is_active  = $5
WHERE id = $1
RETURNING *;

-- name: DeleteLink :execrows
DELETE FROM links WHERE id = $1;
