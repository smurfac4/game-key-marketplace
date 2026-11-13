BEGIN;
ALTER TABLE orders ADD COLUMN checkout_token text;
CREATE UNIQUE INDEX orders_checkout_token_unique ON orders(user_id,checkout_token) WHERE checkout_token IS NOT NULL;
COMMIT;
