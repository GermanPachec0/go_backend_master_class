-- name: UpsertRestaurant :one
INSERT INTO orders.restaurants (restaurant_uuid, name, description, address, currency)
VALUES
	($1, $2, $3, $4, $5)
ON CONFLICT (restaurant_uuid) DO UPDATE SET
	name = EXCLUDED.name,
	description = EXCLUDED.description,
	address = EXCLUDED.address
RETURNING *;

-- name: GetRestaurant :one
SELECT
	*
FROM
	orders.restaurants
WHERE
	restaurant_uuid = $1
;
