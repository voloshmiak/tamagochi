-- name: EnsureInventory :exec
INSERT INTO inventories (owner_id) VALUES ($1) ON CONFLICT (owner_id) DO NOTHING;

-- name: InventoryByOwner :one
SELECT * FROM inventories WHERE owner_id = $1;

-- name: InventoryByOwnerForUpdate :one
SELECT * FROM inventories WHERE owner_id = $1 FOR UPDATE;

-- name: InventoryItems :many
SELECT item_id, quantity FROM inventory_items WHERE owner_id = $1;

-- name: SetDailyRewardAt :exec
UPDATE inventories SET daily_reward_at = $2 WHERE owner_id = $1;

-- name: DeleteInventoryItems :exec
DELETE FROM inventory_items WHERE owner_id = $1;

-- name: InsertInventoryItems :exec
INSERT INTO inventory_items (owner_id, item_id, quantity)
SELECT sqlc.arg(owner_id), unnest(sqlc.arg(item_ids)::text[]), unnest(sqlc.arg(quantities)::integer[]);
