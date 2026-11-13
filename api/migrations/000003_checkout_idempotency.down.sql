BEGIN;
DROP INDEX orders_checkout_token_unique;
ALTER TABLE orders DROP COLUMN checkout_token;
COMMIT;
