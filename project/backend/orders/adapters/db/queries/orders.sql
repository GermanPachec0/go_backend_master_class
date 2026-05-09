-- Quotes are immutable - no update query exists. If needed, create a new quote.
-- name: AddQuote :exec
INSERT INTO orders.quotes (
	quote_uuid,
	customer_uuid,
	restaurant_uuid,
	delivery_address,
	items_subtotal_gross,
	service_fee_gross,
	delivery_fee_gross,
	total_amount_gross,
	total_tax,
	created_at,
	currency
)
VALUES
	($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
;

-- :copyfrom uses PostgreSQL COPY for bulk inserts. See: https://docs.sqlc.dev/en/stable/howto/insert.html#using-copyfrom
-- name: AddQuoteItems :copyfrom
INSERT INTO orders.quote_items (
	quote_item_uuid,
	quote_uuid,
	menu_item_uuid,
	gross_price,
	quantity
)
VALUES
	($1, $2, $3, $4, $5);

-- name: InsertOrder :exec
INSERT INTO orders.orders (
	order_uuid,
	quote_uuid,
	customer_uuid,
	restaurant_uuid,
	delivery_address,
	ordered_at,
	items_subtotal_gross,
	service_fee_gross,
	delivery_fee_gross,
	total_amount_gross,
	total_tax,
	courier_uuid,
	currency
)
VALUES
	($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
RETURNING *;

-- name: GetQuoteItems :many
SELECT *
FROM orders.quote_items
WHERE quote_uuid = $1;

-- name: GetQuote :one
SELECT
	*
FROM
	orders.quotes AS quotes
WHERE
	quote_uuid = $1
LIMIT 1;


-- name: GetMenuItemsForQuote :many
SELECT
	mi.*
FROM orders.quote_items AS qi
INNER JOIN  orders.restaurant_menu_items AS mi ON mi.restaurant_menu_item_uuid = qi.menu_item_uuid
WHERE qi.quote_uuid = $1
;
	
