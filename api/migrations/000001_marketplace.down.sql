BEGIN;
DROP TABLE homepage_blocks, banners, game_keys, order_items, orders, cart_items, carts, offers, sellers, products, users;
-- pg_trgm may predate this app and be shared by other schemas; retain the extension.
COMMIT;
