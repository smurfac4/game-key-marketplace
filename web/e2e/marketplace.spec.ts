import { test, expect, type Page } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";
async function accessible(page: Page) {
  const result = await new AxeBuilder({ page })
    .withTags(["wcag2a", "wcag2aa", "wcag21aa"])
    .analyze();
  expect(result.violations).toEqual([]);
}
const password = "e2e-safe-test-password-123";
async function register(page: Page, next = "/") {
  const email = `e2e-${Date.now()}-${Math.random().toString(16).slice(2)}@example.invalid`;
  await page.goto(`/register?next=${encodeURIComponent(next)}`);
  await page.getByLabel("Email address").fill(email);
  await page.getByLabel("Password", { exact: true }).fill(password);
  await page.getByLabel("Confirm password").fill(password);
  await accessible(page);
  await page
    .getByRole("button", { name: "Create account", exact: true })
    .click();
  await expect(page).not.toHaveURL(/\/register/);
  return email;
}
test("search, fuzzy hint, keyboard suggestions and URL-backed filters", async ({
  page,
}) => {
  await page.goto("/search?q=cyberpnk");
  await expect(page.getByText(/Did you mean one of these/)).toBeVisible();
  const search = page.getByRole("combobox", { name: "Search games" });
  await search.fill("cyberpunk");
  await expect(
    page.getByRole("listbox").getByRole("option").first(),
  ).toBeVisible();
  await search.press("ArrowDown");
  await search.press("Enter");
  await expect(page).toHaveURL(/\/products\//);
  await expect(page.getByRole("heading", { level: 1 })).toContainText(
    "Cyberpunk",
  );
  await search.fill("");
  await search.focus();
  await expect(
    page.getByText("Recent searches", { exact: true }),
  ).toBeVisible();
  await search.press("Escape");
  await expect(page.getByRole("listbox")).not.toBeVisible();
  await page.goto("/store?page_size=12&sort=price_asc");
  await expect(page.locator('a[href^="/products/"]')).toHaveCount(12);
  await page.getByRole("button", { name: "Next", exact: true }).click();
  await expect(page).toHaveURL(/page=2/);
  await page.goBack();
  await expect(page).not.toHaveURL(/page=2/);
  await page.getByLabel("Genre", { exact: true }).selectOption("RPG");
  await page.getByRole("button", { name: "Apply", exact: true }).click();
  await expect(page).toHaveURL(/genre=/);
  await expect(page.locator('a[href^="/products/"]').first()).toBeVisible();
  await expect(page.getByLabel("Genre", { exact: true })).toHaveValue("RPG");
  await accessible(page);
  await page.screenshot({
    path: "test-results/store-desktop.png",
    fullPage: true,
  });
});
test("register, cart quantity, declined payment rollback, checkout, keys, orders and logout", async ({
  page,
  browser,
}) => {
  const email = await register(page, "/products/demo-game-001");
  await expect(
    page.getByRole("heading", { name: "Compare sellers" }),
  ).toBeVisible();
  await accessible(page);
  await page.screenshot({
    path: "test-results/product-desktop.png",
    fullPage: true,
  });
  await page.reload();
  await expect(
    page.getByRole("heading", { name: "Compare sellers" }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Add to cart", exact: true })
    .first()
    .click();
  await expect(page.getByRole("status")).toContainText("Added to cart");
  await page.goto("/cart");
  await page.getByRole("combobox", { name: /Quantity:/ }).selectOption("2");
  await expect(page.getByRole("combobox", { name: /Quantity:/ })).toHaveValue(
    "2",
  );
  await expect(page.locator('a[href="/checkout"]')).not.toHaveClass(
    /pointer-events-none/,
  );
  await accessible(page);
  await page.screenshot({
    path: "test-results/cart-desktop.png",
    fullPage: true,
  });
  await page.getByRole("link", { name: "Checkout", exact: true }).click();
  await page.getByLabel("Test card number").waitFor();
  await accessible(page);
  await page.screenshot({
    path: "test-results/checkout-desktop.png",
    fullPage: true,
  });
  await page.getByLabel("Test card number").fill("4000000000000002");
  await page.getByRole("button", { name: "Pay & reveal keys" }).click();
  await expect(page.getByRole("alert")).toContainText("Mock payment declined");
  await page.getByLabel("Test card number").fill("4242424242424242");
  await page.getByRole("button", { name: "Pay & reveal keys" }).click();
  await expect(
    page.getByRole("heading", { name: "Your keys are ready" }),
  ).toBeVisible();
  await expect(page.getByTestId("digital-key")).toHaveCount(2);
  await expect(page.getByTestId("digital-key").first()).toContainText(
    "FAKE-NOT-REDEEMABLE",
  );
  await page.context().grantPermissions(["clipboard-read", "clipboard-write"]);
  await page
    .getByRole("button", { name: "Copy key", exact: true })
    .first()
    .click();
  await expect(
    page.getByRole("button", { name: "Copied", exact: true }),
  ).toBeVisible();
  await accessible(page);
  const orderID = new URL(page.url()).pathname.split("/").at(-1)!;
  await page.screenshot({
    path: "test-results/order-desktop.png",
    fullPage: true,
  });
  // A different authenticated buyer cannot retrieve the purchased keys (IDOR).
  const other = await browser.newContext({
    baseURL: new URL(page.url()).origin,
  });
  const headers = { "X-CSRF-Protection": "1" };
  const otherEmail = `other-${Date.now()}@example.invalid`;
  expect(
    (
      await other.request.post("/api/auth/register", {
        headers,
        data: { email: otherEmail, password },
      })
    ).status(),
  ).toBe(201);
  expect(
    (
      await other.request.post("/api/auth/login", {
        headers,
        data: { email: otherEmail, password },
      })
    ).status(),
  ).toBe(200);
  const denied = await other.request.get(`/api/orders/${orderID}`);
  expect(denied.status()).toBe(404);
  expect(await denied.text()).not.toContain("FAKE-");
  await other.close();
  await page.goto("/orders");
  await expect(page.getByRole("link", { name: "View order" })).toHaveCount(1);
  await accessible(page);
  await page.screenshot({
    path: "test-results/orders-desktop.png",
    fullPage: true,
  });
  await page.goto("/cart");
  await expect(
    page.getByText("Your cart is empty", { exact: true }),
  ).toBeVisible();
  await page.getByText("Account ▾", { exact: true }).click();
  await page.getByRole("button", { name: "Log out", exact: true }).click();
  await page.goto("/orders");
  await expect(page).toHaveURL(/\/login\?next=/);
  await accessible(page);
  await page.screenshot({
    path: "test-results/login-desktop.png",
    fullPage: true,
  });
  await page.getByLabel("Email address").fill(email);
  await page.getByLabel("Password", { exact: true }).fill(password);
  await page.getByRole("button", { name: "Log in", exact: true }).click();
  await expect(page).toHaveURL(/\/orders$/);
});
test("homepage blocks fail independently and carousel is controllable", async ({
  page,
}) => {
  await page.route("**/api/home/deals", (route) =>
    route.fulfill({
      status: 503,
      contentType: "application/json",
      body: JSON.stringify({
        error: { code: "unavailable", message: "Demo block unavailable" },
      }),
    }),
  );
  await page.goto("/");
  await expect(
    page.getByRole("heading", { name: "Best sellers", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("heading", { name: "New releases", exact: true }),
  ).toBeVisible();
  await expect(page.getByRole("alert")).toContainText("Demo block unavailable");
  await page.getByRole("button", { name: "Next banner", exact: true }).click();
  await expect(page.getByRole("heading", { level: 1 })).toHaveText(
    "Explore this week's deals",
  );
  await accessible(page);
  await page.screenshot({
    path: "test-results/home-desktop.png",
    fullPage: true,
  });
});
test("mobile filters, modal focus, empty and 404 states", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/store");
  await page.getByRole("button", { name: "Filters", exact: true }).click();
  const dialog = page.getByRole("dialog", { name: "Filters" });
  await expect(dialog).toBeVisible();
  await accessible(page);
  await page.screenshot({
    path: "test-results/filters-mobile.png",
    fullPage: true,
  });
  await dialog.getByLabel("Platform", { exact: true }).selectOption("Steam");
  await dialog.getByRole("button", { name: "Apply", exact: true }).click();
  await expect(dialog).not.toBeVisible();
  await expect(page).toHaveURL(/platform=Steam/);
  await page.screenshot({
    path: "test-results/store-mobile.png",
    fullPage: true,
  });
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth,
    ),
  ).toBeTruthy();
  await page.goto("/search?q=zzzzzz-no-matching-game");
  await expect(
    page.getByRole("heading", { name: "No games found" }),
  ).toBeVisible();
  await page.goto("/products/no-such-product");
  await expect(page.getByText("This game could not be found.")).toBeVisible();
});

test("showcase artwork, live categories and featured purchase", async ({
  page,
}) => {
  await page.goto("/");
  await expect(
    page.getByRole("heading", { name: "Best sellers", exact: true }),
  ).toBeVisible();
  const covers = page.locator("#best-sellers img");
  await expect(covers.first()).toHaveAttribute("src", /\/media\/games\//);
  await expect
    .poll(() =>
      covers
        .first()
        .evaluate(
          (img: HTMLImageElement) => img.complete && img.naturalWidth > 0,
        ),
    )
    .toBeTruthy();
  const hero = page.locator(".hero-main img");
  await expect
    .poll(() =>
      hero.evaluate(
        (img: HTMLImageElement) => img.complete && img.naturalWidth > 0,
      ),
    )
    .toBeTruthy();
  await page
    .getByRole("button", { name: "Next: Best sellers", exact: true })
    .click();
  await expect
    .poll(() =>
      page
        .locator("#best-sellers .home-product-row")
        .evaluate((el) => el.scrollLeft),
    )
    .toBeGreaterThan(0);
  await page
    .getByRole("button", { name: "Previous: Best sellers", exact: true })
    .click();
  await page.screenshot({
    path: "test-results/showcase-home-desktop.png",
    fullPage: true,
  });
  await page
    .locator(".category-rail")
    .getByRole("link", { name: "RPG", exact: true })
    .click();
  await expect(page).toHaveURL(/genre=RPG/);
  await expect(page.getByLabel("Genre", { exact: true })).toHaveValue("RPG");
  await register(page, "/products/demo-game-001");
  await page
    .getByRole("button", { name: "Buy this offer", exact: true })
    .click();
  await expect(page).toHaveURL(/\/cart$/);
  await expect(page.getByRole("combobox", { name: /Quantity:/ })).toHaveValue(
    "1",
  );
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/products/demo-game-001");
  await expect(
    page.getByRole("button", { name: "Buy this offer", exact: true }),
  ).toBeVisible();
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBeTruthy();
  await expect(
    page.getByRole("button", { name: "Add to cart", exact: true }).first(),
  ).toBeVisible();
  await accessible(page);
  await page.screenshot({
    path: "test-results/showcase-product-mobile.png",
    fullPage: true,
  });
  await page.goto("/");
  await expect(
    page.getByRole("heading", { name: "Best sellers", exact: true }),
  ).toBeVisible();
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBeTruthy();
  await accessible(page);
  await page.screenshot({
    path: "test-results/showcase-home-mobile.png",
    fullPage: true,
  });
});

test("catalog network failures show retry controls", async ({ page }) => {
  for (const failure of ["500", "drop", "timeout"]) {
    await page.route("**/api/products?**", (route) =>
      failure === "drop"
        ? route.abort("failed")
        : route.fulfill({
            status: failure === "timeout" ? 504 : 500,
            contentType: "application/json",
            body: JSON.stringify({
              error: {
                code: failure === "timeout" ? "timeout" : "internal_error",
                message:
                  failure === "timeout"
                    ? "request timed out"
                    : "Unable to complete request.",
              },
            }),
          }),
    );
    await page.goto("/store");
    await expect(page.getByRole("alert")).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Try again", exact: true }),
    ).toBeVisible();
    await page.unroute("**/api/products?**");
  }
  await page.goto("/login");
  await page.getByLabel("Email address").focus();
  await page.keyboard.press("Tab");
  await expect(page.getByLabel("Password", { exact: true })).toBeFocused();
});

// Navigation uses Tab/Enter/typed keys only: no pointer clicks or programmatic focus.
for (const mobile of [false, true]) {
  test(`complete keyboard purchase and HTTP idempotency ${mobile ? "mobile" : "desktop"}`, async ({
    page,
  }) => {
    if (mobile) await page.setViewportSize({ width: 390, height: 844 });
    async function tabTo(target: ReturnType<Page["getByRole"]>) {
      await expect(target).toBeVisible();
      for (let i = 0; i < 150; i++) {
        if (await target.evaluate((el) => el === document.activeElement))
          return;
        await page.keyboard.press("Tab");
      }
      throw new Error("Keyboard target unreachable");
    }
    await page.goto("/register");
    await tabTo(page.getByLabel("Email address"));
    await page.keyboard.type(
      `keyboard-${Date.now()}-${mobile}@example.invalid`,
    );
    await tabTo(page.getByLabel("Password", { exact: true }));
    await page.keyboard.type(password);
    await tabTo(page.getByLabel("Confirm password"));
    await page.keyboard.type(password);
    await tabTo(
      page.getByRole("button", { name: "Create account", exact: true }),
    );
    await page.keyboard.press("Enter");
    await expect(page).toHaveURL(/\/$/);
    await tabTo(page.getByRole("combobox", { name: "Search games" }));
    await page.keyboard.type("cyberpnk");
    await page.keyboard.press("Enter");
    await expect(page).toHaveURL(/\/search\?q=cyberpnk/);
    await expect(page.getByText(/Did you mean one of these/)).toBeVisible();
    await tabTo(page.locator('a[href^="/products/"]').first());
    await page.keyboard.press("Enter");
    await expect(
      page.getByRole("heading", { name: "Compare sellers" }),
    ).toBeVisible();
    await tabTo(
      page.getByRole("button", { name: "Add to cart", exact: true }).first(),
    );
    await page.keyboard.press("Enter");
    await expect(page.getByRole("status")).toContainText("Added to cart");
    await tabTo(page.getByRole("link", { name: /^Cart \(/ }));
    await page.keyboard.press("Enter");
    await expect(page).toHaveURL(/\/cart$/);
    await tabTo(page.getByRole("link", { name: "Checkout", exact: true }));
    await page.keyboard.press("Enter");
    await tabTo(page.getByLabel("Test card number"));
    await page.keyboard.type("4242424242424242");
    let captured: { body: string; token: string } | undefined;
    page.on("request", (request) => {
      if (
        request.url().endsWith("/api/checkout") &&
        request.method() === "POST"
      ) {
        captured = {
          body: request.postData()!,
          token: request.headers()["idempotency-key"],
        };
      }
    });
    await tabTo(page.getByRole("button", { name: "Pay & reveal keys" }));
    await page.keyboard.press("Enter");
    await expect(
      page.getByRole("heading", { name: "Your keys are ready" }),
    ).toBeVisible();
    await expect(page.getByTestId("digital-key")).toHaveCount(1);
    const id = new URL(page.url()).pathname.split("/").at(-1);
    expect(captured).toBeDefined();
    const headers = {
      "X-CSRF-Protection": "1",
      "Idempotency-Key": captured!.token,
      "Content-Type": "application/json",
    };
    const replay = await page.request.post("/api/checkout", {
      headers,
      data: JSON.parse(captured!.body),
    });
    expect(replay.status()).toBe(200);
    const result = await replay.json();
    expect(String(result.data.id)).toBe(id);
    expect(replay.headers()["cache-control"]).toBe("no-store");
    const conflict = await page.request.post("/api/checkout", {
      headers,
      data: { card_number: "4000000000000002" },
    });
    expect(conflict.status()).toBe(409);
    expect((await conflict.json()).error.code).toBe("idempotency_conflict");
    await accessible(page);
    await tabTo(page.getByText("Account ▾", { exact: true }));
    await page.keyboard.press("Enter");
    await tabTo(page.getByRole("button", { name: "Log out", exact: true }));
    await page.keyboard.press("Enter");
    await expect
      .poll(async () => (await page.request.get("/api/me")).status())
      .toBe(401);
  });
}

test("language selector persists and real GBP checkout", async ({ page }) => {
  await register(page, "/store");
  await expect(
    page.getByRole("heading", { name: "Store", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("img", { name: "English language flag", exact: true }),
  ).toBeVisible();
  await page.getByLabel("Language", { exact: true }).selectOption("ru");
  await expect(
    page.getByRole("img", { name: "Russian language flag", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("heading", { name: "Магазин", exact: true }),
  ).toBeVisible();
  await page.reload();
  await expect(page.locator("html")).toHaveAttribute("lang", "ru");
  await expect(page.getByLabel("Language", { exact: true })).toHaveValue("ru");
  await expect(
    page.getByRole("img", { name: "Russian language flag", exact: true }),
  ).toBeVisible();
  await page.goto("/");
  await expect(page.getByRole("heading", { level: 1 })).toHaveText(
    "Твоя следующая игровая вселенная",
  );
  await page.goto("/missing-localized-page");
  await expect(
    page.getByRole("heading", { name: "Страница не найдена", exact: true }),
  ).toBeVisible();
  await page.goto("/store");
  await page.getByLabel("Language", { exact: true }).selectOption("en");
  await expect(
    page.getByRole("img", { name: "English language flag", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("heading", { name: "Store", exact: true }),
  ).toBeVisible();
  await page.getByLabel("Currency", { exact: true }).selectOption("USD");
  await expect(page.locator('a[href^="/products/"]').first()).toContainText(
    "$",
  );
  await page.getByLabel("Currency", { exact: true }).selectOption("GBP");
  await expect(page.locator('a[href^="/products/"]').first()).toContainText(
    "£",
  );
  await page.locator('a[href^="/products/"]').first().click();
  await page
    .getByRole("button", { name: "Add to cart", exact: true })
    .first()
    .click();
  await page.goto("/cart");
  await page.getByRole("link", { name: "Checkout", exact: true }).click();
  await page.getByLabel("Test card number").fill("4242424242424242");
  const quote = (
    await (
      await page.request.get("/api/cart", {
        headers: { "X-Display-Currency": "GBP" },
      })
    ).json()
  ).data;
  let requestCurrency = "";
  page.on("request", (r) => {
    if (r.url().endsWith("/api/checkout") && r.method() === "POST")
      requestCurrency = JSON.parse(r.postData()!).currency;
  });
  await page.getByRole("button", { name: "Pay & reveal keys" }).click();
  await expect(
    page.getByRole("heading", { name: "Your keys are ready" }),
  ).toBeVisible();
  expect(requestCurrency).toBe("GBP");
  const id = new URL(page.url()).pathname.split("/").at(-1);
  const response = await page.request.get(`/api/orders/${id}`);
  const purchased = (await response.json()).data;
  expect(purchased.currency).toBe("GBP");
  expect(purchased.total).toBe(quote.total);
  expect(purchased.items[0].unit_price).toBe(quote.items[0].unit_price);
  const csrf = { "X-CSRF-Protection": "1", "X-Display-Currency": "USD" };
  const usdCart = await page.request.post("/api/cart/items", {
    headers: csrf,
    data: { offer_id: quote.items[0].offer_id, quantity: 1 },
  });
  expect(usdCart.status()).toBe(200);
  const usdQuote = (await usdCart.json()).data;
  const usdOrder = await page.request.post("/api/checkout", {
    headers: { ...csrf, "Idempotency-Key": "currency-usd-check-123" },
    data: { card_number: "4242424242424242", currency: "USD" },
  });
  expect(usdOrder.status()).toBe(200);
  const usdPurchase = (await usdOrder.json()).data;
  expect(usdPurchase.currency).toBe("USD");
  expect(usdPurchase.total).toBe(usdQuote.total);
  expect(usdPurchase.total).not.toBe(quote.total);
  const conflicting = await page.request.post("/api/checkout", {
    headers: { ...csrf, "Idempotency-Key": "currency-usd-check-123" },
    data: { card_number: "4242424242424242", currency: "GBP" },
  });
  expect(conflicting.status()).toBe(409);
  await page.setViewportSize({ width: 390, height: 844 });
  await expect(page.getByLabel("Currency", { exact: true })).toBeVisible();
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBeTruthy();
  await accessible(page);
  const image = await page.request.get("/media/games/1091500.svg");
  expect(image.status()).toBe(200);
  expect(image.headers()["content-type"]).toContain("image/svg");
});

test("select arrows are inset and English hero uses the original raster scene", async ({
  page,
}) => {
  await page.goto("/");
  await expect(page.getByRole("heading", { level: 1 })).toHaveText(
    "Discover your next adventure",
  );
  const hero = page.locator(".hero-main img");
  await expect(hero).toHaveAttribute("src", "/media/banners/portal-v2.png");
  await expect
    .poll(() =>
      hero.evaluate(
        (image: HTMLImageElement) => image.complete && image.naturalWidth > 0,
      ),
    )
    .toBeTruthy();
  for (const label of ["Currency"]) {
    const control = page.getByLabel(label, { exact: true });
    expect(
      await control.evaluate((el) =>
        parseFloat(getComputedStyle(el).paddingRight),
      ),
    ).toBeGreaterThanOrEqual(40);
    expect(
      await control.evaluate((el) => getComputedStyle(el).appearance),
    ).toBe("none");
  }
  await page.screenshot({
    path: "test-results/hero-v3-desktop.png",
    fullPage: false,
  });
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/store");
  await page.getByRole("button", { name: "Filters", exact: true }).click();
  const dialog = page.getByRole("dialog", { name: "Filters" });
  const platform = dialog.getByLabel("Platform", { exact: true });
  expect(
    await platform.evaluate((el) =>
      parseFloat(getComputedStyle(el).paddingRight),
    ),
  ).toBeGreaterThanOrEqual(40);
  await accessible(page);
  await page.screenshot({
    path: "test-results/select-v3-mobile.png",
    fullPage: false,
  });
});

test("category menu and sidebar remain interchangeable with history and zero results", async ({
  page,
}) => {
  await page.goto("/store?page=2&page_size=12&sort=price_asc");
  await page.locator(".categories-dropdown summary").click();
  await page
    .locator(".category-menu")
    .getByRole("link", { name: "GOG", exact: true })
    .click();
  await expect(page).toHaveURL(/platform=GOG/);
  await expect(page).not.toHaveURL(/page=2/);
  await expect(page).toHaveURL(/sort=price_asc/);
  await expect(page.getByLabel("Platform", { exact: true })).toHaveValue("GOG");
  await expect(page.locator(".product-grid a").first()).toBeVisible();
  const platforms = page.getByLabel("Platform", { exact: true });
  await expect(platforms.locator('option[value="Steam"]')).toHaveCount(1);
  await expect(platforms.locator('option[value="Epic Games"]')).toHaveCount(1);
  await platforms.selectOption("Steam");
  await page.getByRole("button", { name: "Apply", exact: true }).click();
  await expect(page).toHaveURL(/platform=Steam/);
  await expect(platforms).toHaveValue("Steam");
  await page.goBack();
  await expect(page).toHaveURL(/platform=GOG/);
  await expect(platforms).toHaveValue("GOG");
  await page.goForward();
  await expect(platforms).toHaveValue("Steam");
  await page.locator(".categories-dropdown summary").click();
  await page
    .locator(".category-menu")
    .getByRole("link", { name: "RPG", exact: true })
    .click();
  await expect(page).toHaveURL(/genre=RPG/);
  await expect(page).toHaveURL(/platform=Steam/);
  await expect(page.getByLabel("Genre", { exact: true })).toHaveValue("RPG");
  await page.getByLabel("Genre", { exact: true }).selectOption("Action");
  await page.getByRole("button", { name: "Apply", exact: true }).click();
  await expect(page).toHaveURL(/genre=Action/);
  await expect(page.getByLabel("Genre", { exact: true })).toHaveValue("Action");
  await page.goto("/store?platform=GOG&genre=Racing");
  await expect(
    page.getByRole("heading", { name: "No games found" }),
  ).toBeVisible();
  await expect(platforms).toHaveValue("GOG");
  await expect(page.getByLabel("Genre", { exact: true })).toHaveValue("Racing");
  await page.getByLabel("Genre", { exact: true }).selectOption("RPG");
  await page.getByRole("button", { name: "Apply", exact: true }).click();
  await expect(page.locator(".product-grid a").first()).toBeVisible();
  await page
    .getByRole("button", { name: "Clear filters", exact: true })
    .click();
  await expect(platforms).toHaveValue("");
  await expect(page.getByLabel("Genre", { exact: true })).toHaveValue("");
  await expect(page).not.toHaveURL(/platform=|genre=/);
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/store?platform=GOG&genre=RPG");
  await page.getByRole("button", { name: "Filters", exact: true }).click();
  const dialog = page.getByRole("dialog", { name: "Filters" });
  await expect(dialog.getByLabel("Platform", { exact: true })).toHaveValue(
    "GOG",
  );
  await dialog.getByLabel("Platform", { exact: true }).selectOption("Steam");
  await dialog.getByRole("button", { name: "Apply", exact: true }).click();
  await expect(dialog).not.toBeVisible();
  await expect(page).toHaveURL(/platform=Steam/);
  await expect(page).toHaveURL(/genre=RPG/);
  await page.getByRole("button", { name: "Filters", exact: true }).click();
  await expect(
    page.getByRole("dialog").getByLabel("Platform", { exact: true }),
  ).toHaveValue("Steam");
  await accessible(page);
});
