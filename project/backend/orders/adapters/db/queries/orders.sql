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
