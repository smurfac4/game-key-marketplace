BEGIN;
-- Older clients only accept HTTPS artwork. Preserve records with a neutral fallback.
UPDATE products SET image_url='https://placehold.co/600x900/png' WHERE image_url LIKE '/media/%';
UPDATE banners SET image_url='https://placehold.co/1400x480/png' WHERE image_url LIKE '/media/%';
ALTER TABLE products DROP CONSTRAINT products_image_url_check;
ALTER TABLE products ADD CONSTRAINT products_image_url_check CHECK (image_url ~ '^https://');
ALTER TABLE banners DROP CONSTRAINT banners_image_url_check;
ALTER TABLE banners ADD CONSTRAINT banners_image_url_check CHECK (image_url ~ '^https://');
COMMIT;
