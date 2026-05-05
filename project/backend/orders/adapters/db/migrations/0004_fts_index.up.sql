BEGIN;  
CREATE INDEX idx_menu_items_fts
    ON orders.restaurant_menu_items
    USING gin (to_tsvector('english', name));
COMMIT;