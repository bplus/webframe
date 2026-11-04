#!/usr/bin/env node
// palette-check: load a page whose app sets appshell.App.Palette in
// Playwright's Chromium and check the command palette's browser-side
// behavior (see tools/README.md):
//
//   hint   the trigger's shortcut hint reads "⌘K" on a Mac and "Ctrl+K"
//          elsewhere, and that chord opens the palette on that platform;
//   sheet  at 390px the open sheet's height cap and top follow
//          window.visualViewport: the whole screen with no keyboard, the
//          free part above it (and its offset) once a keyboard takes room.
//
//   node tools/palette-check.mjs --url http://localhost:3333/
//
// Exit codes: 0 ok, 1 a check failed, 2 usage.

import { chromium } from "playwright";

const USAGE = `usage: node tools/palette-check.mjs --url <u> [--shot <prefix>]
  --url <u>        a page with the palette trigger in its shell (required)
  --shot <prefix>  also save <prefix>-hint-<platform>.png and <prefix>-sheet.png`;

function usage(msg) {
  process.stderr.write(`palette-check: ${msg}\n${USAGE}\n`);
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

// fakePlatform makes navigator report one platform before any page script
// runs; the palette reads userAgentData first, so that goes too.
function fakePlatform(platform) {
  Object.defineProperty(Navigator.prototype, "platform", { get: () => platform, configurable: true });
  Object.defineProperty(Navigator.prototype, "userAgentData", { get: () => undefined, configurable: true });
}

// fakeViewport replaces window.visualViewport with a stand-in the test can
// resize, since headless Chromium has no on-screen keyboard.
function fakeViewport() {
  // Until the test sets a height it is the window's own (read late: this
  // runs before the viewport meta applies, when innerHeight is still wrong).
  const vv = new EventTarget();
  let height = null;
  let offsetTop = 0;
  Object.defineProperty(vv, "height", { get: () => (height === null ? window.innerHeight : height) });
  Object.defineProperty(vv, "offsetTop", { get: () => offsetTop });
  Object.defineProperty(window, "visualViewport", { value: vv, configurable: true });
  window.__setViewport = (h, top) => {
    height = h;
    offsetTop = top;
    vv.dispatchEvent(new Event("resize"));
  };
}

function sheetBox() {
  const c = document.querySelector("[data-wf-palette] [data-tui-dialog-content]");
  const cs = getComputedStyle(c);
  return { open: c.getAttribute("data-tui-dialog-open"), maxHeight: cs.maxHeight, top: cs.top, height: c.getBoundingClientRect().height };
}

async function hintRun(browser, opts, platform, chord, want) {
  const page = await browser.newPage({ viewport: { width: 1280, height: 800 } });
  const errors = [];
  page.on("console", (m) => { if (m.type() === "error") errors.push(m.text()); });
  page.on("pageerror", (e) => errors.push(String(e)));
  await page.addInitScript(fakePlatform, platform);
  await page.goto(opts.url, { waitUntil: "networkidle" });
  // A shell can render more than one trigger (the sidebar layout has one in
  // its phone menu row too); every hint must agree.
  const hints = await page.$$eval("[data-wf-palette-trigger] kbd", (els) => els.map((e) => e.textContent.trim()));
  check(hints.length > 0 && hints.every((h) => h === want), `${platform}: trigger hint is ${JSON.stringify(want)} (got ${JSON.stringify(hints)})`);
  if (opts.shot) await page.locator("[data-wf-palette-trigger]:visible").first().screenshot({ path: `${opts.shot}-hint-${platform}.png` });
  await page.keyboard.press(chord);
  await page.waitForTimeout(300);
  const open = await page.evaluate(() => window.wf.palette.isOpen());
  check(open, `${platform}: ${chord} opens the palette`);
  check(errors.length === 0, `${platform}: no console errors ${JSON.stringify(errors)}`);
  await page.close();
}

async function sheetRun(browser, opts) {
  const page = await browser.newPage({ viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true, deviceScaleFactor: 2 });
  const errors = [];
  page.on("console", (m) => { if (m.type() === "error") errors.push(m.text()); });
  page.on("pageerror", (e) => errors.push(String(e)));
  await page.addInitScript(fakeViewport);
  await page.goto(opts.url, { waitUntil: "networkidle" });
  await page.locator("[data-wf-palette-trigger]:visible").first().click();
  await page.waitForTimeout(400);
  let box = await page.evaluate(sheetBox);
  check(box.open === "true", "390: the trigger opens the sheet");
  check(box.maxHeight === "844px" && box.top === "0px", `390, no keyboard: cap is the whole screen, 844px at top 0 (got ${box.maxHeight} at ${box.top})`);
  // A keyboard takes the bottom 400px and the browser scrolls the visible
  // area down by 100px.
  await page.evaluate(() => window.__setViewport(444, 100));
  await page.waitForTimeout(400); // past the sheet's 200ms transition
  box = await page.evaluate(sheetBox);
  check(box.maxHeight === "444px" && box.top === "100px", `390, keyboard up: cap is the visible 444px at top 100px (got ${box.maxHeight} at ${box.top})`);
  check(box.height <= 444, `390, keyboard up: the sheet is no taller than the visible area (${box.height}px)`);
  await page.evaluate(() => window.__setViewport(844, 0));
  await page.waitForTimeout(400);
  box = await page.evaluate(sheetBox);
  check(box.maxHeight === "844px" && box.top === "0px", `390, keyboard down again: back to 844px at top 0 (got ${box.maxHeight} at ${box.top})`);
  if (opts.shot) await page.screenshot({ path: `${opts.shot}-sheet.png` });
  check(errors.length === 0, `390: no console errors ${JSON.stringify(errors)}`);
  await page.close();
}

const opts = parseArgs(process.argv.slice(2));
const browser = await chromium.launch();
try {
  await hintRun(browser, opts, "MacIntel", "Meta+k", "⌘K");
  await hintRun(browser, opts, "Win32", "Control+k", "Ctrl+K");
  await hintRun(browser, opts, "Linux x86_64", "Control+k", "Ctrl+K");
  await sheetRun(browser, opts);
} finally {
  await browser.close();
}
if (failures.length) {
  process.stderr.write(`palette-check: ${failures.length} check(s) failed\n`);
  process.exit(1);
}
