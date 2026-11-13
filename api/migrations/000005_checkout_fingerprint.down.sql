ALTER TABLE orders DROP COLUMN checkout_fingerprint;
-- Keep local SVG support on rollback: it is a safe extension of the media allowlist.
