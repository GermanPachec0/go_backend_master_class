-- name: ListMenuItemsWithRestaurant :many
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


-- name: ListRestaurants :many
SELECT
    restaurant_uuid,
    name,
    description,
    address,
    currency
FROM orders.restaurants
ORDER BY name ASC
;

-- name: ListCustomerOrders :many
SELECT orders.* FROM orders.orders as orders
WHERE customer_uuid = $1
ORDER BY orders.ordered_at DESC
;

-- name: ListRestaurantOrders :many
SELECT orders.* FROM orders.orders as orders
WHERE restaurant_uuid = $1
;

-- name: ListAssignedCourierOrders :many
SELECT orders.*, restaurants.name AS restaurant_name FROM orders.orders as orders
INNER JOIN orders.restaurants ON orders.restaurant_uuid = restaurants.restaurant_uuid
WHERE courier_uuid = $1 
;

-- name: ListAvailableOrdersForCourier :many
SELECT orders.* , restaurants.name AS restaurant_name
FROM orders.orders as orders
INNER JOIN orders.couriers ON couriers.courier_uuid = $1
INNER JOIN orders.restaurants ON orders.restaurant_uuid = restaurants.restaurant_uuid
WHERE orders.restaurant_confirmed_at IS NOT NULL
AND orders.courier_uuid IS NULL
AND orders.delivered_at IS NULL
AND orders.courier_accepted_at IS NULL

AND couriers.city = orders.delivery_address ->> 'city'
;