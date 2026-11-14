# Screenshot captures

Six real Chromium/Playwright captures at 1280 CSS pixels wide, using the English/EUR interface and a dedicated `keyspace-screenshots` Compose project with a fresh seeded database. The product capture is full-page to include the seller comparison; the other captures use a 1280×1000 viewport.

All account details are synthetic. The account menu is closed, no email/password/session data is captured, and library keys are generated non-redeemable fixtures. No development database is used. External SMTP/OIDC is disabled; the disposable project and volume are removed on exit.

| Image | State |
| --- | --- |
| [homepage.png](homepage.png) | Homepage and promotions |
| [store-filters.png](store-filters.png) | Steam/RPG catalog with price bounds and sorting |
| [typo-search.png](typo-search.png) | `cyberpnk` results and fuzzy-match hint |
| [product-sellers.png](product-sellers.png) | Product details and seller comparison |
| [checkout.png](checkout.png) | Mock payment form and cart summary |
| [keys-library.png](keys-library.png) | Purchased non-redeemable demo key |

Regenerate from the repository root with `bash scripts/capture-screenshots.sh`. Requires Docker, installed project npm dependencies, Python 3 and Pillow. PNGs are optimized losslessly with Pillow; pixel equality is checked, and each file keeps the smaller of the original and optimized encodings. [manifest.json](manifest.json) records actual dimensions and byte counts. No browser traces or session exports are written.
