-- name: UpsertRestaurant :one
INSERT INTO orders.restaurants (restaurant_uuid, name, description, address, currency)
VALUES
	($1, $2, $3, $4, $5)
ON CONFLICT (restaurant_uuid) DO UPDATE SET
	name = EXCLUDED.name,
	description = EXCLUDED.description,
	address = EXCLUDED.address
RETURNING *;

-- name: UpsertRestaurantMenuItem :exec
INSERT INTO orders.restaurant_menu_items (
	restaurant_menu_item_uuid,
	restaurant_uuid,
	name,
	gross_price,
	ordering,
	is_archived
)
VALUES
	($1, $2, $3, $4, $5, $6)
ON CONFLICT (restaurant_menu_item_uuid) DO UPDATE SET
	restaurant_uuid = EXCLUDED.restaurant_uuid,
	name = EXCLUDED.name,
	gross_price = EXCLUDED.gross_price,
	ordering = EXCLUDED.ordering,
	is_archived = EXCLUDED.is_archived
;

-- name: GetRestaurant :one
SELECT
	*
FROM
	orders.restaurants
WHERE
	restaurant_uuid = $1
;

-- name: GetRestaurantMenu :many
SELECT
	sqlc.embed(restaurant_menu_items)
FROM
	orders.restaurant_menu_items AS restaurant_menu_items
WHERE
	restaurant_uuid = $1 AND
	is_archived = FALSE
ORDER BY
	ordering ASC
;

-- name: GetMenuItemsByUUIDs :many
SELECT
	restaurant_menu_items.*
FROM
	orders.restaurant_menu_items AS restaurant_menu_items
WHERE
	restaurant_uuid = $1 AND
	restaurant_menu_item_uuid = ANY ($2::UUID[])
;

-- name: ArchiveMenuItems :exec
UPDATE
	orders.restaurant_menu_items
SET
	is_archived = TRUE
WHERE
	restaurant_menu_item_uuid = ANY ($1::UUID[])
;

-- name: ListMenuItems :many
-- Lists menu items with optional restaurant name filter, optional full-text search, and dynamic ordering.
-- Uses CASE WHEN to support multiple ordering options in a single query.
SELECT
    mi.restaurant_menu_item_uuid AS menu_item_uuid,
    mi.name AS menu_item_name,
    mi.gross_price,
    r.currency,
    r.restaurant_uuid,
    r.name AS restaurant_name,
    CASE WHEN sqlc.narg(search_term)::text IS NOT NULL
         THEN ts_rank(
             to_tsvector('english', mi.name),
             plainto_tsquery('english', sqlc.narg(search_term)::text)
         )
         ELSE NULL
    END AS relevance
FROM orders.restaurant_menu_items mi
JOIN orders.restaurants r ON mi.restaurant_uuid = r.restaurant_uuid
WHERE mi.is_archived = false
  AND (sqlc.narg(search_term)::text IS NULL
       OR to_tsvector('english', mi.name) @@ plainto_tsquery('english', sqlc.narg(search_term)::text))
  AND (sqlc.narg(restaurant_name_filter)::text IS NULL
       OR LOWER(r.name) LIKE LOWER('%' || sqlc.narg(restaurant_name_filter)::text || '%'))
ORDER BY
    CASE WHEN sqlc.narg(order_by)::text = 'relevance'
         THEN ts_rank(
             to_tsvector('english', mi.name),
             plainto_tsquery('english', sqlc.narg(search_term)::text)
         )
    END DESC,
    CASE WHEN (sqlc.narg(order_by)::text IS NULL OR sqlc.narg(order_by)::text = 'default')
         THEN r.name END ASC,
    CASE WHEN (sqlc.narg(order_by)::text IS NULL OR sqlc.narg(order_by)::text = 'default')
         THEN mi.ordering END ASC,
    CASE WHEN sqlc.narg(order_by)::text = 'price_asc' THEN mi.gross_price END ASC,
    CASE WHEN sqlc.narg(order_by)::text = 'price_desc' THEN mi.gross_price END DESC,
    CASE WHEN sqlc.narg(order_by)::text = 'name_asc' THEN mi.name END ASC,
    CASE WHEN sqlc.narg(order_by)::text = 'name_desc' THEN mi.name END DESC;
