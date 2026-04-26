-- name: UpsertRestaurant :one
INSERT INTO orders.restaurants (restaurant_uuid, name, description, address, currency)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (restaurant_uuid) DO UPDATE
SET name = EXCLUDED.name,	
	description = EXCLUDED.description,
	address = EXCLUDED.address,
	currency = EXCLUDED.currency
RETURNING *;

-- name: GetRestaurant :one
SELECT
	*
FROM
	orders.restaurants
WHERE
	restaurant_uuid = $1
;

-- name: GetRestaurantMenu :many
SELECT sqlc.embed(r) 
FROM orders.restaurant_menu_items r
WHERE is_archived = FALSE
AND restaurant_uuid = $1
ORDER BY ordering ASC;

-- name: UpsertRestaurantMenuItem :one
INSERT INTO orders.restaurant_menu_items (restaurant_menu_item_uuid, restaurant_uuid, name, gross_price, ordering, is_archived)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (restaurant_menu_item_uuid) DO UPDATE
SET restaurant_uuid = EXCLUDED.restaurant_uuid,
	name = EXCLUDED.name,
	gross_price = EXCLUDED.gross_price,
	ordering = EXCLUDED.ordering,
	is_archived = EXCLUDED.is_archived
RETURNING *;