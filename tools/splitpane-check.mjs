#!/usr/bin/env node
// splitpane-check: load the demo's split pane (/browse) in Playwright's
// Chromium and check what a failed list click does (see tools/README.md):
//
//   gone    at 1280, open one item, delete another behind the page's back
//           (DELETE /items/{id}), click it: the "no longer here" toast
//           shows, the detail region and the address bar keep the first
//           item;
//   failed  a click answered 500, and a click that gets no answer, show
//           the "could not load" toast and no other (htmx-errors.js
//           leaves list links to splitpane.js), detail and address
//           unchanged;
//   phone   at 390 the same click on a deleted item is a full page load
//           that shows the 404 page.
//
//   node tools/splitpane-check.mjs --url http://localhost:3333/browse
//
// It deletes one item from the running demo (the demo's data lives in
// memory; restart it to get the item back). Exit codes: 0 ok, 1 a check
// failed, 2 usage.

import { chromium } from "playwright";

const USAGE = `usage: node tools/splitpane-check.mjs --url <u> [--shot <prefix>]
  --url <u>        the demo's /browse page (required)
  --shot <prefix>  also save <prefix>-1280.png and <prefix>-390.png`;

function usage(msg) {
  process.stderr.write(`splitpane-check: ${msg}\n${USAGE}\n`);
  process.exit(2);
}

function parseArgs(argv) {
  const opts = { url: null, shot: null };
  for (let i = 0; i < argv.length; i++) {
    const flag = argv[i];
    if (flag === "-h" || flag === "--help") {
      process.stdout.write(USAGE + "\n");
      process.exit(0);
    }
    if (flag !== "--url" && flag !== "--shot") usage(`unknown argument ${flag}`);
    const value = argv[++i];
    if (!value) usage(`${flag} needs a value`);
    opts[flag.slice(2)] = value;
  }
  if (!opts.url) usage("--url is required");
  return opts;
}

const failures = [];
function check(ok, what) {
  process.stdout.write(`${ok ? "ok  " : "FAIL"} ${what}\n`);
  if (!ok) failures.push(what);
}

const opts = parseArgs(process.argv.slice(2));
const LINK = "a[data-wf-splitpane-link]";
const DETAIL = "#browse-detail";

// oneToast checks that a failed click left exactly one toast of any kind
// in #toast-container: the pane's, not also htmx-errors.js's.
async function oneToast(page, what) {
  await page.waitForTimeout(300);
  const n = await page.locator("#toast-container > *").count();
  check(n === 1, `${what}: one toast in the container, found ${n}`);
}

// toastText waits for a toast in #toast-container and returns its text, or
// "" when none appears.
async function toastText(page) {
  const t = page.locator("#toast-container [data-wf-splitpane-notice]");
  try {
    await t.first().waitFor({ state: "visible", timeout: 3000 });
  } catch {
    return "";
  }
  return (await t.allInnerTexts()).join(" | ");
}

// unchanged checks that a failed click left the detail and the URL alone.
async function unchanged(page, what, detail, url) {
  check((await page.locator(DETAIL).innerText()) === detail, `${what}: detail region unchanged`);
  check(page.url() === url, `${what}: address bar unchanged (${page.url()})`);
}

const browser = await chromium.launch();
try {
  // The phone page is loaded first, so its list still holds the item the
  // desktop part deletes.
  const phone = await browser.newPage({ viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true, deviceScaleFactor: 2 });
  await phone.goto(opts.url);

  // --- desktop ---
  const page = await browser.newPage({ viewport: { width: 1280, height: 800 } });
  const errors = [];
  page.on("console", (m) => {
    // A failed request logs console errors by itself (the browser's
    // "Failed to load resource", htmx's "Response Status Error Code 404"
    // and, for no answer, "htmx:sendError"); those are the point of the
    // check. Anything else is a real error.
    if (m.type() === "error" && !/Failed to load resource|Response Status Error Code (404|500)|^htmx:(afterRequest|sendError)$/.test(m.text())) errors.push(m.text());
  });
  await page.goto(opts.url);
  const hrefs = await page.locator(LINK).evaluateAll((as) => as.map((a) => a.getAttribute("href")));
  if (hrefs.length < 4) usage(`expected at least 4 list links on ${opts.url}, found ${hrefs.length}`);
  const [first, , failing, gone] = hrefs.slice(-4);

  await page.locator(`${LINK}[href="${first}"]`).click();
  await page.waitForURL((u) => u.pathname === first);
  await page.locator(DETAIL).getByRole("heading").first().waitFor();
  const detail = await page.locator(DETAIL).innerText();
  const url = page.url();

  // Gone: the item is deleted after the list was drawn.
  const del = await page.evaluate(async (h) => (await fetch("/items/" + h.split("/").pop(), { method: "DELETE", redirect: "manual" })).status, gone);
  check(del === 303 || del === 0, `DELETE /items/${gone.split("/").pop()} answered (status ${del})`);
  await page.locator(`${LINK}[href="${gone}"]`).click();
  const goneToast = await toastText(page);
  check(/no longer here/.test(goneToast) && /may have been deleted/.test(goneToast), `gone: toast reads "${goneToast}"`);
  await unchanged(page, "gone", detail, url);
  // A second click replaces the toast rather than stacking a second one.
  await page.locator(`${LINK}[href="${gone}"]`).click();
  await page.waitForTimeout(200);
  check((await page.locator("#toast-container [data-wf-splitpane-notice]").count()) === 1, "gone: a second click keeps one toast");
  if (opts.shot) await page.screenshot({ path: `${opts.shot}-1280.png` });

  // Failed: a 500, then no answer at all. clearToasts first, so the gone
  // toast still on screen is not read as the answer.
  const clearToasts = () => page.evaluate(() => document.querySelectorAll("#toast-container > *").forEach((t) => t.remove()));
  await clearToasts();
  await page.route("**" + failing, (route) => route.fulfill({ status: 500, contentType: "text/plain", body: "Internal Server Error" }));
  await page.locator(`${LINK}[href="${failing}"]`).click();
  const failedToast = await toastText(page);
  check(/Could not load that item/.test(failedToast) && !/no longer here/.test(failedToast), `500: toast reads "${failedToast}"`);
  await oneToast(page, "500");
  await unchanged(page, "500", detail, url);
  await page.unroute("**" + failing);
  await clearToasts();
  await page.route("**" + failing, (route) => route.abort("connectionrefused"));
  await page.locator(`${LINK}[href="${failing}"]`).click();
  const abortToast = await toastText(page);
  check(/Could not load that item/.test(abortToast), `no answer: toast reads "${abortToast}"`);
  await oneToast(page, "no answer");
  await unchanged(page, "no answer", detail, url);
  check(errors.length === 0, `desktop: no other console errors ${JSON.stringify(errors)}`);
  await page.close();

  // --- phone: a plain navigation, so the 404 page itself ---
  await phone.locator(`${LINK}[href="${gone}"]`).click();
  await phone.waitForURL((u) => u.pathname === gone);
  const body = await phone.locator("main").innerText();
  check(/not found/i.test(body), "phone: a deleted item's link shows the 404 page");
  if (opts.shot) await phone.screenshot({ path: `${opts.shot}-390.png` });
  await phone.close();
} finally {
  await browser.close();
}

if (failures.length) {
  process.stderr.write(`splitpane-check: ${failures.length} check(s) failed\n`);
  process.exit(1);
}
