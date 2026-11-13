BEGIN;
ALTER TABLE products DROP CONSTRAINT products_image_url_check;
ALTER TABLE products ADD CONSTRAINT products_image_url_check CHECK (image_url ~ '^https://' OR image_url ~ '^/media/(games|banners)/[a-zA-Z0-9_-]+\.(jpg|webp|png)$');
ALTER TABLE banners DROP CONSTRAINT banners_image_url_check;
ALTER TABLE banners ADD CONSTRAINT banners_image_url_check CHECK (image_url ~ '^https://' OR image_url ~ '^/media/(games|banners)/[a-zA-Z0-9_-]+\.(jpg|webp|png)$');
COMMIT;
