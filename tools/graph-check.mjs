#!/usr/bin/env node
// graph-check: load a page with graphwidgets in Playwright's Chromium and
// check three runs of it (see tools/README.md):
//
//   render   every [data-tui-graph] gets a <canvas> of non-zero size inside
//            it, is marked ready, shows no error box, and the page logs no
//            console errors;
//   no-lib   with 3d-force-graph.min.js blocked, every widget shows its
//            error box and logs a console error;
//   bad-data with the graph JSON answering HTTP 500, the same.
//
//   node tools/graph-check.mjs --url http://localhost:3333/graph --data /api/graph
//
// Exit codes: 0 ok, 1 a check failed, 2 usage.

import { chromium } from "playwright";

const USAGE = `usage: node tools/graph-check.mjs --url <u> --data <path>
  --url <u>      page with one or more graphwidgets (required)
  --data <path>  URL path of the graph JSON the widgets fetch (required, for the bad-data run)`;

function usage(msg) {
  process.stderr.write(`graph-check: ${msg}\n${USAGE}\n`);
  process.exit(2);
}

function parseArgs(argv) {
  const opts = { url: null, data: null };
  for (let i = 0; i < argv.length; i++) {
    const flag = argv[i];
    if (flag === "-h" || flag === "--help") {
      process.stdout.write(USAGE + "\n");
      process.exit(0);
    }
    if (flag !== "--url" && flag !== "--data") usage(`unknown argument ${flag}`);
    const value = argv[++i];
    if (!value) usage(`${flag} needs a value`);
    opts[flag.slice(2)] = value;
  }
  if (!opts.url || !opts.data) usage("--url and --data are required");
  return opts;
}

// widgetState reports, per widget, its canvases' rendered sizes, whether
// the script marked it ready or failed, and the error box's visible text.
function widgetState() {
  return [...document.querySelectorAll("[data-tui-graph]")].map((el) => {
    const box = el.querySelector("[data-tui-graph-error]");
    const boxVisible = !!box && !box.hidden && box.getBoundingClientRect().height > 0;
    return {
      id: el.id,
      canvases: [...el.querySelectorAll("canvas")].map((c) => {
        const r = c.getBoundingClientRect();
        return [Math.round(r.width), Math.round(r.height)];
      }),
      ready: el.getAttribute("data-tui-graph-ready") === "true",
      error: boxVisible ? box.textContent : null,
    };
  });
}

async function run(browser, opts, mode) {
  const context = await browser.newContext({ viewport: { width: 1280, height: 800 } });
  const page = await context.newPage();
  const errors = [];
  page.on("console", (msg) => {
    if (msg.type() === "error") errors.push(`console.error: ${msg.text()}`);
  });
  page.on("pageerror", (err) => errors.push(`uncaught: ${err.message}`));
  if (mode === "no-lib") {
    await page.route("**/3d-force-graph.min.js*", (route) => route.abort());
  }
  if (mode === "bad-data") {
    await page.route((u) => new URL(u).pathname === opts.data, (route) =>
      route.fulfill({ status: 500, contentType: "text/plain", body: "boom" }));
  }
  const response = await page.goto(opts.url, { waitUntil: "load", timeout: 15000 });
  if (!response || response.status() >= 400) {
    throw new Error(`failed to load ${opts.url}: HTTP ${response ? response.status() : "none"}`);
  }
  // Wait until every widget has settled one way or the other.
  try {
    await page.waitForFunction(() => {
      const els = [...document.querySelectorAll("[data-tui-graph]")];
      return els.length > 0 && els.every((el) =>
        el.getAttribute("data-tui-graph-ready") === "true" ||
        el.getAttribute("data-tui-graph-failed") === "true");
    }, null, { timeout: 10000 });
  } catch {
    // Fall through: the state dump below says which widget never settled.
  }
  await page.waitForTimeout(500);
  const state = await page.evaluate(widgetState);
  await context.close();

  const problems = [];
  if (state.length === 0) problems.push("no [data-tui-graph] element on the page");
  for (const w of state) {
    if (mode === "render") {
      if (!w.canvases.some(([cw, ch]) => cw > 0 && ch > 0)) problems.push(`#${w.id}: no <canvas> with non-zero size (got ${JSON.stringify(w.canvases)})`);
      if (!w.ready) problems.push(`#${w.id}: not marked data-tui-graph-ready`);
      if (w.error) problems.push(`#${w.id}: error box shown: ${w.error}`);
    } else {
      if (!w.error) problems.push(`#${w.id}: no visible error box`);
      if (w.ready) problems.push(`#${w.id}: marked ready despite the fault`);
    }
  }
  if (mode === "render") {
    for (const e of errors) problems.push(e);
  } else if (errors.length === 0) {
    problems.push("no console error logged");
  }
  process.stdout.write(JSON.stringify({ mode, widgets: state, consoleErrors: errors.length }) + "\n");
  for (const p of problems) process.stderr.write(`graph-check [${mode}]: ${p}\n`);
  return problems.length === 0;
}

async function main() {
  const opts = parseArgs(process.argv.slice(2));
  const browser = await chromium.launch();
  let ok = true;
  try {
    for (const mode of ["render", "no-lib", "bad-data"]) {
      if (!(await run(browser, opts, mode))) ok = false;
    }
  } finally {
    await browser.close();
  }
  process.exit(ok ? 0 : 1);
}

main().catch((err) => {
  process.stderr.write(`graph-check: ${err.stack || err}\n`);
  process.exit(1);
});
