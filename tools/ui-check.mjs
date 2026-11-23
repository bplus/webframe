#!/usr/bin/env node
// ui-check: load one demo page in Playwright's Chromium at a device-style
// viewport, measure it, and fail loud on horizontal overflow, load failures,
// console errors (minus any matched by --ignore-console) and a false --assert.
// See tools/README.md.
//
//   node tools/ui-check.mjs --url http://localhost:3333/items --width 390 --shot tools/shots/items-390.png
//
// Always prints one JSON line (last line of stdout):
//   {url, width, height, scrollWidth, horizontalOverflow}
// Exit codes: 0 ok, 1 overflow / load failure / console error / false assert, 2 usage.

import { chromium } from "playwright";
import { mkdir } from "node:fs/promises";
import { dirname } from "node:path";

const USAGE = `usage: node tools/ui-check.mjs --url <u> [options]
  --url <u>          page to load (required)
  --width <n>        viewport width in CSS px (default 1280)
  --height <n>       viewport height (default 844 when width <= 500, else 800)
  --shot <path>      write a full-page PNG here (directories are created)
  --eval <js>        evaluate this expression in the page after load; prints the JSON result
  --assert <js>      repeatable; evaluate this expression in the page after load and
                     fail the run unless the result is truthy
  --click <selector> click this element before --eval/--shot (waits 300ms after)
  --hover <selector> hover this element before --eval/--shot (waits 300ms after)
  --wait <ms>        extra wait after networkidle, before click/hover/eval/shot
  --ignore-console <substring>
                     repeatable; a console/page error whose text (incl. resource URL)
                     contains the substring is printed as "(ignored)" and does not fail the run`;

function fail(code, msg) {
  process.stderr.write(`ui-check: ${msg}\n`);
  process.exit(code);
}

function parseArgs(argv) {
  const opts = { width: 1280, height: null, shot: null, eval: null, click: null, hover: null, wait: 0, url: null, ignoreConsole: [], assert: [] };
  const takesValue = new Set(["--url", "--width", "--height", "--shot", "--eval", "--assert", "--click", "--hover", "--wait", "--ignore-console"]);
  for (let i = 0; i < argv.length; i++) {
    const flag = argv[i];
    if (flag === "-h" || flag === "--help") {
      process.stdout.write(USAGE + "\n");
      process.exit(0);
    }
    if (!takesValue.has(flag)) fail(2, `unknown argument ${flag}\n${USAGE}`);
    const value = argv[++i];
    if (value === undefined) fail(2, `${flag} needs a value\n${USAGE}`);
    const key = flag.slice(2);
    if (key === "ignore-console") {
      if (value === "") fail(2, `${flag} needs a non-empty substring`);
      opts.ignoreConsole.push(value);
      continue;
    }
    if (key === "assert") {
      opts.assert.push(value);
      continue;
    }
    if (key === "width" || key === "height" || key === "wait") {
      const n = Number(value);
      if (!Number.isInteger(n) || n < 0 || (key !== "wait" && n === 0)) fail(2, `${flag} must be a positive integer, got ${JSON.stringify(value)}`);
      opts[key] = n;
    } else {
      opts[key] = value;
    }
  }
  if (!opts.url) fail(2, `--url is required\n${USAGE}`);
  if (opts.height === null) opts.height = opts.width <= 500 ? 844 : 800;
  return opts;
}

async function main() {
  const opts = parseArgs(process.argv.slice(2));
  const phone = opts.width <= 500;
  const consoleErrors = [];
  let browser;
  let failed = false;
  try {
    browser = await chromium.launch();
    const context = await browser.newContext({
      viewport: { width: opts.width, height: opts.height },
      deviceScaleFactor: 2,
      isMobile: phone,
      hasTouch: phone,
      colorScheme: "light",
    });
    const page = await context.newPage();
    page.on("console", (msg) => {
      if (msg.type() !== "error") return;
      const loc = msg.location();
      const where = loc && loc.url ? ` (${loc.url}${loc.lineNumber ? ":" + (loc.lineNumber + 1) : ""})` : "";
      consoleErrors.push(`console.error: ${msg.text()}${where}`);
    });
    page.on("pageerror", (err) => consoleErrors.push(`uncaught: ${err.message}`));

    let response;
    try {
      response = await page.goto(opts.url, { waitUntil: "load", timeout: 15000 });
    } catch (err) {
      fail(1, `failed to load ${opts.url}: ${err.message}`);
    }
    if (!response) fail(1, `failed to load ${opts.url}: no response`);
    if (response.status() >= 400) fail(1, `failed to load ${opts.url}: HTTP ${response.status()}`);
    await page.waitForLoadState("networkidle", { timeout: 15000 });
    if (opts.wait > 0) await page.waitForTimeout(opts.wait);

    if (opts.click) {
      await page.click(opts.click, { timeout: 5000 });
      await page.waitForTimeout(300);
    }
    if (opts.hover) {
      await page.hover(opts.hover, { timeout: 5000 });
      await page.waitForTimeout(300);
    }
    if (opts.eval) {
      const result = await page.evaluate(opts.eval);
      process.stdout.write(JSON.stringify(result === undefined ? null : result) + "\n");
    }
    for (const expr of opts.assert) {
      if (!(await page.evaluate(expr))) {
        process.stderr.write(`ui-check: assert failed: ${expr}\n`);
        failed = true;
      }
    }
    if (opts.shot) {
      await mkdir(dirname(opts.shot), { recursive: true });
      await page.screenshot({ path: opts.shot, fullPage: true });
    }

    const scrollWidth = await page.evaluate(() => document.documentElement.scrollWidth);
    const horizontalOverflow = scrollWidth > opts.width;
    process.stdout.write(JSON.stringify({
      url: opts.url,
      width: opts.width,
      height: opts.height,
      scrollWidth,
      horizontalOverflow,
    }) + "\n");

    if (horizontalOverflow) {
      process.stderr.write(`ui-check: horizontal overflow: scrollWidth ${scrollWidth} > viewport ${opts.width}\n`);
      failed = true;
    }
    for (const line of consoleErrors) {
      const ignored = opts.ignoreConsole.some((sub) => line.includes(sub));
      process.stderr.write(`ui-check: ${line}${ignored ? " (ignored)" : ""}\n`);
      if (!ignored) failed = true;
    }
  } finally {
    if (browser) await browser.close();
  }
  process.exit(failed ? 1 : 0);
}

main().catch((err) => fail(1, err.stack || String(err)));
