#!/usr/bin/env node
// filterbar-check: drive the demo's filter bars in Playwright's Chromium and
// check what filterbar.min.js sends and what lands (see tools/README.md):
//
//   blur     on /items, type in Search, wait for the answer, click
//            elsewhere on the page: the search box's change event sends no
//            second request;
//   pill     type in Search, wait, click a toggle pill: exactly one more
//            request, carrying the search text and the pill;
//   enter    type and press Enter before the debounce time is over: one
//            request at once, none when the timer would have fired;
//   newest   with the answer to the search request held back, a pill click
//            made meanwhile goes out at once and wins: after every answer
//            has landed the results are those of the newest request and so
//            is the address bar;
//   stale    the same with every request that lacks the pill held back, so
//            a stale request sent after the first answer would land last;
//   choice   a click on a single-choice button sends one request with that
//            value alone, a click on another sends that one alone, a click
//            on the selected one or on "Clear filters" sends no value;
//   line     on the showcase at 390, a group of 16 buttons stays on one
//            line that scrolls inside the bar: the page has no horizontal
//            overflow, the cut-off edge is marked, and the selected button
//            is brought into view after a click and after a render.
//
//   node tools/filterbar-check.mjs --url http://localhost:3333
//
// Exit codes: 0 ok, 1 a check failed, 2 usage.

import { chromium } from "playwright";

const USAGE = `usage: node tools/filterbar-check.mjs --url <u>
  --url <u>  the demo's base URL, for example http://localhost:3333 (required)`;

function usage(msg) {
  process.stderr.write(`filterbar-check: ${msg}\n${USAGE}\n`);
  process.exit(2);
}

function parseArgs(argv) {
  const opts = { url: null };
  for (let i = 0; i < argv.length; i++) {
    const flag = argv[i];
    if (flag === "-h" || flag === "--help") {
      process.stdout.write(USAGE + "\n");
      process.exit(0);
    }
    if (flag !== "--url") usage(`unknown argument ${flag}`);
    const value = argv[++i];
    if (!value) usage(`${flag} needs a value`);
    opts.url = value.replace(/\/+$/, "");
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
const SEARCH = '#items-filterbar input[type="search"]';
const DRAFT = '#items-filterbar [data-tui-filterbar-toggle][value="draft"]';
const RESULTS = "#items-results";
// Longer than the search box's debounce time (400ms) plus an answer.
const SETTLE = 900;

// open loads path in a new page and records every htmx request the page
// sends to reqPath (the filter bar's requests; the page's own path unless
// given), as URL objects, and every console error. ignore is a pattern of
// console errors the page logs on purpose.
async function open(browser, path, { width = 1280, reqPath = path, ignore = null } = {}) {
  const page = await browser.newPage({ viewport: { width, height: 800 } });
  const sent = [];
  const errors = [];
  page.on("request", (req) => {
    const u = new URL(req.url());
    if (u.pathname === reqPath && req.headers()["hx-request"] === "true") {
      u.at = Date.now();
      sent.push(u);
    }
  });
  page.on("console", (m) => {
    if (m.type() === "error" && !(ignore && ignore.test(m.text() + " " + (m.location().url || "")))) errors.push(m.text());
  });
  page.on("pageerror", (e) => errors.push(String(e)));
  await page.goto(opts.url + path);
  return { page, sent, errors };
}

// typeAndWait types text into the search box and waits until the debounced
// request has been answered and swapped.
async function typeAndWait(page, text) {
  const answered = page.waitForResponse((r) => new URL(r.url()).searchParams.get("q") === text);
  await page.locator(SEARCH).pressSequentially(text, { delay: 20 });
  await answered;
  await page.waitForTimeout(SETTLE);
}

// rows is the text of the results region's table body rows.
async function rows(page) {
  return (await page.locator(`${RESULTS} tbody tr`).allInnerTexts()).map((t) => t.replace(/\s+/g, " ").trim());
}

// freshRows loads url in a new page and returns its rows: what the server
// renders for that state with no script involved in the request.
async function freshRows(browser, url) {
  const page = await browser.newPage({ viewport: { width: 1280, height: 800 } });
  await page.goto(url);
  const r = await rows(page);
  await page.close();
  return r;
}

const describe = (sent) => JSON.stringify(sent.map((u) => u.search));

// hold delays every bar request whose URL matches, by ms.
async function hold(page, matches, ms) {
  await page.route("**/items?*", async (route) => {
    const req = route.request();
    if (req.headers()["hx-request"] === "true" && matches(new URL(req.url()))) {
      await new Promise((r) => setTimeout(r, ms));
    }
    try {
      await route.continue();
    } catch {
      // The page aborted the request while it was held.
    }
  });
}

// newestWins checks the state after a search and a pill click with some
// answers held back: results, address bar and toast container.
async function newestWins(browser, page, sent, what) {
  const u = new URL(page.url());
  check(u.searchParams.get("q") === "of" && u.searchParams.get("status") === "draft", `${what}: the address bar shows the newest state (${u.search})`);
  const last = sent[sent.length - 1];
  check(last && last.searchParams.get("q") === "of" && last.searchParams.get("status") === "draft", `${what}: the newest request carries the search text and the pill ${describe(sent)}`);
  const want = await freshRows(browser, page.url());
  const got = await rows(page);
  check(want.length > 0 && JSON.stringify(got) === JSON.stringify(want), `${what}: the results are those of the newest request (${got.length} rows shown, ${want.length} expected)`);
  check((await page.locator("#toast-container > *").count()) === 0, `${what}: no toast`);
}

const browser = await chromium.launch();
try {
  // --- blur: the change event after typing sends nothing ---
  {
    const { page, sent, errors } = await open(browser, "/items");
    await typeAndWait(page, "of");
    check(sent.length === 1 && sent[0].searchParams.get("q") === "of", `blur: typing sends one request ${describe(sent)}`);
    await page.locator("h1").first().click();
    await page.waitForTimeout(SETTLE);
    check(sent.length === 1, `blur: a click elsewhere sends no second request ${describe(sent)}`);
    check(new URL(page.url()).searchParams.get("q") === "of", `blur: the address bar keeps the search (${page.url()})`);
    check(errors.length === 0, `blur: no console errors ${JSON.stringify(errors)}`);
    await page.close();
  }

  // --- pill: one more request, with both values ---
  {
    const { page, sent, errors } = await open(browser, "/items");
    await typeAndWait(page, "of");
    const before = sent.length;
    await page.locator(DRAFT).click();
    await page.waitForTimeout(SETTLE);
    const more = sent.slice(before);
    check(more.length === 1, `pill: a pill click after a search sends exactly one more request ${describe(more)}`);
    const last = sent[sent.length - 1];
    check(last.searchParams.get("q") === "of" && last.searchParams.get("status") === "draft", `pill: that request carries the search text and the pill (${last.search})`);
    check(errors.length === 0, `pill: no console errors ${JSON.stringify(errors)}`);
    await page.close();
  }

  // --- enter: one request at once, and the timer is cancelled ---
  {
    const { page, sent, errors } = await open(browser, "/items");
    await page.locator(SEARCH).pressSequentially("of", { delay: 20 });
    await page.locator(SEARCH).press("Enter");
    await page.waitForTimeout(150);
    check(sent.length === 1 && sent[0].searchParams.get("q") === "of", `enter: Enter sends the search at once, before the debounce time ${describe(sent)}`);
    await page.waitForTimeout(SETTLE);
    check(sent.length === 1, `enter: the cancelled timer sends nothing more ${describe(sent)}`);
    check(errors.length === 0, `enter: no console errors ${JSON.stringify(errors)}`);
    await page.close();
  }

  // --- newest: the search answer is held back, the pill click wins ---
  {
    const { page, sent, errors } = await open(browser, "/items");
    await hold(page, (u) => !u.searchParams.has("status"), 1500);
    await page.locator(SEARCH).pressSequentially("of", { delay: 20 });
    await page.waitForTimeout(600); // the debounced request is out and held
    const clicked = Date.now();
    await page.locator(DRAFT).click();
    await page.waitForTimeout(3500); // every answer has landed
    const pill = sent.find((u) => u.searchParams.has("status"));
    check(pill && pill.at - clicked < 500, `newest: the pill request goes out at once, not after the held answer (${pill ? pill.at - clicked : "never"}ms after the click)`);
    await newestWins(browser, page, sent, "newest");
    // htmx logs two console errors for a request it aborts (the names of
    // the events it fires); anything else is a real error.
    const real = errors.filter((e) => !/^htmx:(afterRequest|sendAbort)$/.test(e));
    check(real.length === 0, `newest: no console errors but htmx's abort lines ${JSON.stringify(real)}`);
    check(sent.length === 2, `newest: two requests in all, the held one and the pill's ${describe(sent)}`);
    await page.close();
  }

  // --- stale: a stale request sent after the first answer lands last ---
  {
    const { page, sent, errors } = await open(browser, "/items");
    await typeAndWait(page, "of");
    await hold(page, (u) => !u.searchParams.has("status"), 1500);
    await page.locator(DRAFT).click();
    await page.waitForTimeout(3500);
    await newestWins(browser, page, sent, "stale");
    check(errors.length === 0, `stale: no console errors ${JSON.stringify(errors)}`);
    await page.close();
  }

  // --- choice: one value at most, one request per click ---
  {
    const { page, sent, errors } = await open(browser, "/items");
    const pack = (v) => page.locator(`#items-filterbar [data-tui-filterbar-choice] button[value="${v}"]`);
    const pressed = async () => (await page.locator('#items-filterbar [data-tui-filterbar-choice] button[aria-pressed="true"]').evaluateAll((bs) => bs.map((b) => b.value))).join(",");
    check((await pressed()) === "", `choice: "All" is selected at first (pressed: "${await pressed()}")`);

    await pack("netherese").click();
    await page.waitForTimeout(SETTLE);
    check(sent.length === 1 && sent[0].searchParams.getAll("pack").join("|") === "netherese", `choice: a click sends exactly one request with the one value ${describe(sent)}`);
    check((await pressed()) === "netherese", `choice: the clicked button alone is pressed ("${await pressed()}")`);
    check(new URL(page.url()).searchParams.get("pack") === "netherese", `choice: the address bar shows the value (${page.url()})`);

    await pack("starfall").click();
    await page.waitForTimeout(SETTLE);
    check(sent.length === 2 && sent[1].searchParams.getAll("pack").join("|") === "starfall", `choice: a click on another option sends that option alone ${describe(sent.slice(1))}`);
    check((await pressed()) === "starfall", `choice: the first button is no longer pressed ("${await pressed()}")`);

    await pack("starfall").click();
    await page.waitForTimeout(SETTLE);
    check(sent.length === 3 && !sent[2].searchParams.has("pack"), `choice: a click on the selected option sends no value ${describe(sent.slice(2))}`);
    check((await pressed()) === "", `choice: "All" is selected again ("${await pressed()}")`);

    await pack("").click();
    await page.waitForTimeout(SETTLE);
    check(sent.length === 3, `choice: a click on the selected "All" sends nothing ${describe(sent.slice(3))}`);

    await pack("netherese").click();
    await page.locator(DRAFT).click();
    await page.waitForTimeout(SETTLE);
    const before = sent.length;
    await page.locator("#items-filterbar [data-tui-filterbar-reset]").click();
    await page.waitForTimeout(SETTLE);
    const reset = sent.slice(before);
    check(reset.length === 1 && !reset[0].searchParams.has("pack") && !reset[0].searchParams.has("status"), `choice: "Clear filters" sends one request with no value for the group ${describe(reset)}`);
    check((await pressed()) === "", `choice: "Clear filters" selects "All" ("${await pressed()}")`);
    check(!new URL(page.url()).searchParams.has("pack"), `choice: "Clear filters" takes the value out of the address bar (${page.url()})`);
    check(errors.length === 0, `choice: no console errors ${JSON.stringify(errors)}`);
    await page.close();
  }

  // --- line: 16 buttons at 390 ---
  {
    // The showcase points one avatar at a missing image on purpose.
    const { page, sent, errors } = await open(browser, "/", { width: 390, reqPath: "/api/filter-demo", ignore: /missing-avatar\.png/ });
    const GROUP = "#filter-demo [data-tui-filterbar-choice]";
    const overflow = () => page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
    // inView measures the selected button against the group's visible box.
    const inView = () =>
      page.evaluate((sel) => {
        const g = document.querySelector(sel);
        const b = g.querySelector('[aria-pressed="true"]');
        const gr = g.getBoundingClientRect();
        const br = b.getBoundingClientRect();
        return { value: b.value, ok: br.left >= gr.left - 1 && br.right <= gr.right + 1, more: g.getAttribute("data-tui-filterbar-more"), scrollLeft: g.scrollLeft };
      }, GROUP);
    check((await page.locator(`${GROUP} button`).count()) === 17, "line: the group has 16 options and All");
    check((await overflow()) <= 0, `line: no horizontal overflow of the page at 390 (${await overflow()}px)`);
    const tops = await page.locator(`${GROUP} button`).evaluateAll((bs) => new Set(bs.map((b) => Math.round(b.getBoundingClientRect().top))).size);
    check(tops === 1, `line: every button is on one line (${tops} line(s))`);
    let v = await inView();
    check(v.more === "end", `line: the cut-off edge is marked (data-tui-filterbar-more="${v.more}")`);

    await page.locator(`${GROUP} button[value="transmutation"]`).click();
    await page.waitForTimeout(SETTLE);
    check(sent.length === 1 && sent[0].searchParams.get("school") === "transmutation", `line: the last option sends its value ${describe(sent)}`);
    v = await inView();
    check(v.value === "transmutation" && v.ok && v.scrollLeft > 0, `line: the selected option is in view after the click ${JSON.stringify(v)}`);
    check(v.more === "start", `line: the other edge is marked now (data-tui-filterbar-more="${v.more}")`);

    // A render (here: the group scrolled away, then a resize) brings it back.
    await page.evaluate((sel) => {
      document.querySelector(sel).scrollLeft = 0;
    }, GROUP);
    await page.setViewportSize({ width: 380, height: 800 });
    await page.waitForTimeout(300);
    v = await inView();
    check(v.ok && v.scrollLeft > 0, `line: the selected option is brought back into view after a render ${JSON.stringify(v)}`);
    check((await overflow()) <= 0, `line: still no horizontal overflow (${await overflow()}px)`);
    check(errors.length === 0, `line: no console errors ${JSON.stringify(errors)}`);
    await page.close();
  }
} finally {
  await browser.close();
}

if (failures.length) {
  process.stderr.write(`filterbar-check: ${failures.length} check(s) failed\n`);
  process.exit(1);
}
