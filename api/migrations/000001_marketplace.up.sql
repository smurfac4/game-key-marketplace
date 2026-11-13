BEGIN;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE users (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email text NOT NULL CHECK (btrim(email) <> ''),
    password_hash text NOT NULL CHECK (btrim(password_hash) <> ''),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX users_email_unique ON users (lower(email));

CREATE TABLE products (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name text NOT NULL CHECK (btrim(name) <> ''),
    slug text NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    description text NOT NULL DEFAULT '',
    platform text NOT NULL CHECK (btrim(platform) <> ''),
    region text NOT NULL CHECK (btrim(region) <> ''),
    genre text NOT NULL CHECK (btrim(genre) <> ''),
    image_url text NOT NULL CHECK (image_url ~ '^https://'),
    release_date date,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX products_name_trgm_idx ON products USING gin (name gin_trgm_ops);
CREATE INDEX products_platform_idx ON products (platform);
CREATE INDEX products_region_idx ON products (region);
CREATE INDEX products_genre_idx ON products (genre);
CREATE INDEX products_release_date_idx ON products (release_date DESC, id);

CREATE TABLE sellers (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name text NOT NULL UNIQUE CHECK (btrim(name) <> ''),
    rating numeric(3,2) NOT NULL DEFAULT 0 CHECK (rating BETWEEN 0 AND 5),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE offers (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    product_id bigint NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    seller_id bigint NOT NULL REFERENCES sellers(id) ON DELETE RESTRICT,
    price numeric(12,2) NOT NULL CHECK (price > 0),
    discount_price numeric(12,2) CHECK (discount_price > 0 AND discount_price < price),
    currency text NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    stock integer NOT NULL DEFAULT 0 CHECK (stock >= 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (product_id, seller_id, currency)
);
CREATE INDEX offers_seller_idx ON offers (seller_id);
CREATE INDEX offers_price_idx ON offers (currency, (coalesce(discount_price, price)), product_id) WHERE stock > 0;

CREATE TABLE carts (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id bigint NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE cart_items (
    cart_id bigint NOT NULL REFERENCES carts(id) ON DELETE CASCADE,
    offer_id bigint NOT NULL REFERENCES offers(id) ON DELETE CASCADE,
    quantity integer NOT NULL CHECK (quantity > 0),
    PRIMARY KEY (cart_id, offer_id)
);
CREATE INDEX cart_items_offer_idx ON cart_items (offer_id);

CREATE TABLE orders (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id bigint NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'paid', 'cancelled')),
    currency text NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    total_amount numeric(12,2) NOT NULL CHECK (total_amount >= 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (id, currency)
);
CREATE INDEX orders_user_created_idx ON orders (user_id, created_at DESC);
CREATE INDEX orders_status_created_idx ON orders (status, created_at);
CREATE TABLE order_items (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id bigint NOT NULL,
    offer_id bigint NOT NULL REFERENCES offers(id) ON DELETE RESTRICT,
    product_name text NOT NULL CHECK (btrim(product_name) <> ''),
    seller_name text NOT NULL CHECK (btrim(seller_name) <> ''),
    quantity integer NOT NULL CHECK (quantity > 0),
    unit_price numeric(12,2) NOT NULL CHECK (unit_price > 0),
    currency text NOT NULL,
    FOREIGN KEY (order_id, currency) REFERENCES orders(id, currency) ON DELETE RESTRICT,
    UNIQUE (id, offer_id)
);
CREATE INDEX order_items_order_idx ON order_items (order_id);
CREATE INDEX order_items_offer_idx ON order_items (offer_id);

CREATE TABLE game_keys (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    offer_id bigint NOT NULL REFERENCES offers(id) ON DELETE RESTRICT,
    key_value text NOT NULL UNIQUE CHECK (btrim(key_value) <> ''),
    status text NOT NULL DEFAULT 'available' CHECK (status IN ('available', 'sold')),
    order_item_id bigint,
    created_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (order_item_id, offer_id) REFERENCES order_items(id, offer_id) ON DELETE RESTRICT,
    CHECK ((status = 'available' AND order_item_id IS NULL) OR
           (status = 'sold' AND order_item_id IS NOT NULL))
);
CREATE INDEX game_keys_offer_status_idx ON game_keys (offer_id, status, id);
CREATE INDEX game_keys_order_item_idx ON game_keys (order_item_id) WHERE order_item_id IS NOT NULL;

CREATE TABLE banners (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    title text NOT NULL CHECK (btrim(title) <> ''),
    image_url text NOT NULL CHECK (image_url ~ '^https://'),
    link_url text NOT NULL CHECK (link_url ~ '^(/[^/]|/$|https://)'),
    position integer NOT NULL DEFAULT 0 CHECK (position >= 0),
    active boolean NOT NULL DEFAULT true,
    starts_at timestamptz,
    ends_at timestamptz,
    CHECK (starts_at IS NULL OR ends_at IS NULL OR ends_at > starts_at)
);
CREATE INDEX banners_active_position_idx ON banners (position, id) WHERE active;
CREATE TABLE homepage_blocks (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    slug text NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    title text NOT NULL CHECK (btrim(title) <> ''),
    kind text NOT NULL CHECK (kind IN ('product_collection', 'banner')),
    banner_id bigint REFERENCES banners(id) ON DELETE RESTRICT,
    config jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(config) = 'object'),
    position integer NOT NULL DEFAULT 0 CHECK (position >= 0),
    active boolean NOT NULL DEFAULT true,
    CHECK ((kind = 'banner' AND banner_id IS NOT NULL) OR
           (kind = 'product_collection' AND banner_id IS NULL))
);
CREATE INDEX homepage_blocks_active_position_idx ON homepage_blocks (position, id) WHERE active;
CREATE INDEX homepage_blocks_banner_idx ON homepage_blocks (banner_id) WHERE banner_id IS NOT NULL;
COMMIT;
