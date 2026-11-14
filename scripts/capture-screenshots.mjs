import assert from "node:assert/strict";
import { randomBytes } from "node:crypto";
import { mkdir } from "node:fs/promises";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright";

assert.equal(
  process.env.SCREENSHOT_DISPOSABLE,
  "1",
  "Use capture-screenshots.sh",
);
const directory = fileURLToPath(
  new URL("../docs/screenshots/", import.meta.url),
);
await mkdir(directory, { recursive: true });
const browser = await chromium.launch();
let stage = "initialization";
try {
  const context = await browser.newContext({
    baseURL: "http://localhost:5173",
    viewport: { width: 1280, height: 1000 },
    deviceScaleFactor: 1,
    locale: "en-GB",
    reducedMotion: "reduce",
  });
  const page = await context.newPage();
  const headers = { "X-CSRF-Protection": "1" };
  async function shot(name, fullPage = false) {
    await page.evaluate(async () => {
      await document.fonts.ready;
      await Promise.all(
        Array.from(document.images)
          .filter((img) => img.getBoundingClientRect().top < innerHeight)
          .map((img) => img.decode().catch(() => {})),
      );
    });
    await page.mouse.move(0, 0);
    await page.evaluate(() => {
      window.scrollTo(0, 0);
      return new Promise((resolve) =>
        requestAnimationFrame(() => requestAnimationFrame(resolve)),
      );
    });
    await page.screenshot({
      path: `${directory}${name}.png`,
      fullPage,
      animations: "disabled",
      caret: "hide",
      scale: "css",
    });
    console.log(`Captured ${name}.png at 1280px`);
  }
  stage = "homepage";
  await page.goto("/");
  await page.getByRole("heading", { level: 1 }).waitFor();
  await page
    .getByRole("heading", { name: "Best sellers", exact: true })
    .waitFor();
  await page
    .getByRole("heading", { name: "New releases", exact: true })
    .waitFor();
  await page
    .getByRole("banner")
    .getByRole("link", { name: "Sign up", exact: true })
    .waitFor();
  await shot("homepage");

  stage = "store filters";
  await page.goto(
    "/store?platform=Steam&genre=RPG&min_price=5&max_price=50&sort=price_asc&page_size=12",
  );
  await page.locator(".product-grid a").first().waitFor();
  assert.equal(
    await page.getByLabel("Platform", { exact: true }).inputValue(),
    "Steam",
  );
  assert.equal(
    await page.getByLabel("Genre", { exact: true }).inputValue(),
    "RPG",
  );
  await shot("store-filters");

  stage = "typo search";
  await page.goto("/search?q=cyberpnk");
  await page.getByText(/Did you mean one of these/).waitFor();
  await page.locator(".product-grid a").first().waitFor();
  await shot("typo-search");

  stage = "product sellers";
  await page.goto("/products/demo-game-001");
  await page
    .getByRole("heading", { name: "Compare sellers", exact: true })
    .waitFor();
  await page
    .getByRole("button", { name: "Add to cart", exact: true })
    .first()
    .waitFor();
  await shot("product-sellers", true);

  stage = "synthetic account";
  const credentials = {
    email: "showcase@example.invalid",
    password: randomBytes(24).toString("hex"),
  };
  assert.equal(
    (
      await page.request.post("/api/auth/register", {
        headers,
        data: credentials,
      })
    ).status(),
    201,
  );
  assert.equal(
    (
      await page.request.post("/api/auth/login", { headers, data: credentials })
    ).status(),
    200,
  );
  await page.goto("/products/demo-game-001");
  await page
    .getByRole("button", { name: "Buy this offer", exact: true })
    .click();
  await page.getByRole("link", { name: "Checkout", exact: true }).click();

  stage = "checkout";
  await page.getByLabel("Test card number").waitFor();
  await shot("checkout");
  await page.getByLabel("Test card number").fill("4242424242424242");
  await page
    .getByRole("button", { name: "Pay & reveal keys", exact: true })
    .click();
  await page
    .getByRole("heading", { name: "Your keys are ready", exact: true })
    .waitFor();

  stage = "keys library";
  await page.goto("/library");
  await page.getByTestId("digital-key").first().waitFor();
  assert.match(
    await page.getByTestId("digital-key").first().innerText(),
    /^FAKE-NOT-REDEEMABLE-/,
  );
  await shot("keys-library");
} catch {
  console.error(
    `Screenshot capture failed at ${stage}; no request or session data written.`,
  );
  process.exitCode = 1;
} finally {
  await browser.close();
}
