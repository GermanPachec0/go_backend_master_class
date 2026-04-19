-- todo: implement

BEGIN;
CREATE SCHEMA IF NOT EXISTS orders;
CREATE TABLE IF NOT EXISTS orders.customers (
    customer_uuid UUID NOT NULL PRIMARY KEY,
    name varchar(255) NOT NULL,
    email varchar(255) NOT NULL,
    address JSONB NOT NULL,
    phone_number varchar(50) NOT NULL
);
COMMIT;