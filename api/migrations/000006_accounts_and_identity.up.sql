BEGIN;
CREATE TABLE account_profiles (
 user_id bigint PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
 display_name text NOT NULL DEFAULT '' CHECK(char_length(display_name)<=32),
 country_code text NOT NULL DEFAULT '' CHECK(country_code='' OR country_code ~ '^[A-Z]{2}$'),
 avatar text NOT NULL DEFAULT '' CHECK(avatar IN ('','nebula','pixel','orbit','ember')),
 preferred_platform text NOT NULL DEFAULT '' CHECK(char_length(preferred_platform)<=40),
 marketing_opt_in boolean NOT NULL DEFAULT false,
 wishlist_notifications boolean NOT NULL DEFAULT false,
 updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE wishlists (
 user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 product_id bigint NOT NULL REFERENCES products(id) ON DELETE CASCADE,
 created_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(user_id,product_id)
);
CREATE INDEX wishlists_product_idx ON wishlists(product_id);
CREATE TABLE email_challenges (
 id text PRIMARY KEY,
 email text NOT NULL,
 code_hash text NOT NULL,
 attempts integer NOT NULL DEFAULT 0 CHECK(attempts BETWEEN 0 AND 5),
 expires_at timestamptz NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX email_challenges_expiry_idx ON email_challenges(expires_at);
CREATE TABLE external_identities (
 issuer text NOT NULL,
 subject text NOT NULL,
 user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 PRIMARY KEY(issuer,subject)
);
CREATE INDEX external_identities_user_idx ON external_identities(user_id);
CREATE TABLE oauth_flows (
 state_hash text PRIMARY KEY,
 nonce text NOT NULL,
 verifier text NOT NULL,
 binding_hash text NOT NULL,
 next_path text NOT NULL,
 expires_at timestamptz NOT NULL
);
CREATE INDEX oauth_flows_expiry_idx ON oauth_flows(expires_at);
COMMIT;
