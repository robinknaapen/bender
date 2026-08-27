-- name: ListEnabledServices :many
SELECT * FROM services WHERE enabled = 1 ORDER BY position, id;

-- name: CountServices :one
SELECT COUNT(*) FROM services;

-- name: CreateService :one
INSERT INTO services (preset, name, url, profile, position)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateServicePosition :exec
UPDATE services SET position = ? WHERE id = ?;

-- name: SetServiceEnabled :exec
UPDATE services SET enabled = ? WHERE id = ?;

-- name: GetSetting :one
SELECT value FROM settings WHERE key = ?;

-- name: PutSetting :exec
INSERT INTO settings (key, value) VALUES (?, ?)
ON CONFLICT (key) DO UPDATE SET value = excluded.value;
