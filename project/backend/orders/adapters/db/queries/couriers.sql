-- name: InsertCourier :exec
INSERT INTO orders.couriers (
	courier_uuid,
	name,
	phone_number,
	city
)
VALUES
	($1, $2, $3, $4);


-- name: GetCourierByUUID :one
SELECT
	courier_uuid,
	name,
	phone_number,
	city
FROM orders.couriers
WHERE courier_uuid = $1;