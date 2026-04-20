/**
 * Opens /app/agents/create after login. Requires localhost:3000 and web/.env.playwright.local
 * (see .env.playwright.example). Run: node scripts/verify-agent-wizard.mjs
 */
import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";
import { chromium } from "playwright";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const webRoot = path.join(__dirname, "..");
const envPath = path.join(webRoot, ".env.playwright.local");

function loadEnvFile(filePath) {
  if (!fs.existsSync(filePath)) {
    return null;
  }
  /** @type {Record<string, string>} */
  const out = {};
  for (const line of fs.readFileSync(filePath, "utf8").split("\n")) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith("#")) continue;
    const eq = trimmed.indexOf("=");
    if (eq === -1) continue;
    const key = trimmed.slice(0, eq).trim();
    let val = trimmed.slice(eq + 1).trim();
    if (
      (val.startsWith('"') && val.endsWith('"')) ||
      (val.startsWith("'") && val.endsWith("'"))
    ) {
      val = val.slice(1, -1);
    }
    out[key] = val;
  }
  return out;
}

const env = loadEnvFile(envPath);
const email = env?.PLAYWRIGHT_TEST_EMAIL ?? process.env.PLAYWRIGHT_TEST_EMAIL;
const password =
  env?.PLAYWRIGHT_TEST_PASSWORD ?? process.env.PLAYWRIGHT_TEST_PASSWORD;

if (!email || !password) {
  console.error(
    "Missing PLAYWRIGHT_TEST_EMAIL / PLAYWRIGHT_TEST_PASSWORD. Copy web/.env.playwright.example to web/.env.playwright.local"
  );
  process.exit(1);
}

const base = process.env.PLAYWRIGHT_BASE_URL ?? "http://localhost:3000";
const target = `${base}/auth/login?next=${encodeURIComponent("/app/agents/create")}`;

const browser = await chromium.launch();
const page = await browser.newPage();
const logs = [];
page.on("console", (msg) =>
  logs.push({ type: msg.type(), text: msg.text() })
);
page.on("pageerror", (err) =>
  logs.push({ type: "pageerror", text: err.message })
);

await page.goto(target, { waitUntil: "domcontentloaded", timeout: 60000 });
await page.locator('[data-testid="email"]').fill(email);
await page.locator('[data-testid="password"]').fill(password);
await page.getByRole("button", { name: "Sign In" }).click();
await page.waitForURL(/\/app\/agents\/create/, { timeout: 60000 });

console.log("OK:", page.url());
const errors = logs.filter(
  (l) => l.type === "error" || l.type === "pageerror"
);
if (errors.length) {
  console.log("Console errors:");
  errors.forEach((e) => console.log(e.type, e.text));
}

const outPath = path.join(webRoot, "verify-agent-wizard.png");
await page.screenshot({ path: outPath, fullPage: true });
console.log("Screenshot:", outPath);

await browser.close();
