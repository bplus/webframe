#!/usr/bin/env node
// containers-check: open the showcase's long-list dialog and its drawer with
// a pinned line in Playwright's Chromium and check the layout that class
// names cannot prove (see tools/README.md):
//
//   dialog  the panel fits the screen, its Body scrolls, and the count line
//           and the footer do not move while it does; on a screen too short
//           for the pinned parts the Body keeps its minimum height and the
//           panel scrolls instead;
//   drawer  the Pinned line does not move while the Body scrolls, and the
//           Body keeps its minimum height on a short screen.
//
//   node tools/containers-check.mjs --url http://localhost:3333/
//
// Exit codes: 0 ok, 1 a check failed, 2 usage.

import { chromium } from "playwright";

const USAGE = `usage: node tools/containers-check.mjs --url <u> [--shot <prefix>]
  --url <u>        the showcase page (required)
  --shot <prefix>  also save <prefix>-dialog-<w>x<h>.png and <prefix>-drawer-<w>x<h>.png`;

function usage(msg) {
  process.stderr.write(`containers-check: ${msg}\n${USAGE}\n`);
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

// The last viewport is a phone on its side: too short for the dialog's
// pinned parts and a list.
const VIEWPORTS = [
  { width: 390, height: 844 },
  { width: 1280, height: 800 },
  { width: 390, height: 340 },
];

// Minimum heights of the two Body parts (min-h-48 and min-h-32), less 2px
// for rounding.
const DIALOG_BODY_MIN = 190;
const DRAWER_BODY_MIN = 126;

async function checkDialog(page, tag, shot) {
  await page.click("#long-list-dialog-open");
  await page.waitForTimeout(400);
  const m = await page.evaluate(() => {
    const q = (s) => document.querySelector(s);
    const panel = q("#long-list-dialog [data-tui-dialog-content]");
    const body = q("#long-list-dialog-body");
    const count = q("#long-list-dialog-count");
    const footer = q("#long-list-dialog-footer");
    const top = (e) => Math.round(e.getBoundingClientRect().top);
    const before = [top(count), top(footer)];
    body.scrollTop = 700;
    const after = [top(count), top(footer)];
    const rect = panel.getBoundingClientRect();
    const panelScrolls = panel.scrollHeight > panel.clientHeight + 1;
    panel.scrollTop = panel.scrollHeight;
    return {
      panelTop: rect.top, panelBottom: rect.bottom, vh: window.innerHeight,
      bodyHeight: body.clientHeight, bodyScrollTop: body.scrollTop, bodyScrollHeight: body.scrollHeight,
      before, after, panelScrolls,
      footerBottomAtEnd: footer.getBoundingClientRect().bottom,
      pageOverflow: document.documentElement.scrollWidth > window.innerWidth,
    };
  });
  check(m.panelTop >= 0 && m.panelBottom <= m.vh, `${tag} dialog: the panel is inside the screen (${Math.round(m.panelTop)} to ${Math.round(m.panelBottom)} of ${m.vh})`);
  check(m.bodyHeight >= DIALOG_BODY_MIN, `${tag} dialog: the Body is at least ${DIALOG_BODY_MIN}px tall (${m.bodyHeight})`);
  check(m.bodyScrollHeight > m.bodyHeight && m.bodyScrollTop > 0, `${tag} dialog: the Body scrolls (${m.bodyScrollTop} of ${m.bodyScrollHeight})`);
  check(m.before[0] === m.after[0] && m.before[1] === m.after[1], `${tag} dialog: the count line and the footer do not move while the Body scrolls (${m.before} then ${m.after})`);
  check(m.footerBottomAtEnd <= m.vh, `${tag} dialog: the footer can be reached (bottom ${Math.round(m.footerBottomAtEnd)} of ${m.vh}${m.panelScrolls ? ", after the panel scrolls" : ""})`);
  if (m.vh < 500) {
    check(m.panelScrolls, `${tag} dialog: on a short screen the panel scrolls in place of a Body with no height`);
  } else {
    check(!m.panelScrolls, `${tag} dialog: the panel itself does not scroll`);
  }
  check(!m.pageOverflow, `${tag} dialog: no horizontal overflow of the page`);
  if (shot) await page.screenshot({ path: `${shot}-dialog-${tag}.png` });
}

async function checkDrawer(page, tag, shot) {
  await page.click("#long-list-drawer-open");
  await page.waitForTimeout(500);
  const m = await page.evaluate(() => {
    const q = (s) => document.querySelector(s);
    const panel = q("#long-list-drawer [data-tui-slideover-panel]");
    const body = q("#long-list-drawer-body");
    const pinned = q("#long-list-drawer-pinned");
    const top = (e) => Math.round(e.getBoundingClientRect().top);
    const before = top(pinned);
    body.scrollTop = 700;
    const rect = panel.getBoundingClientRect();
    return {
      before, after: top(pinned), vh: window.innerHeight, vw: window.innerWidth,
      panelTop: rect.top, panelBottom: rect.bottom, panelRight: rect.right,
      bodyHeight: body.clientHeight, bodyScrollTop: body.scrollTop,
      pageOverflow: document.documentElement.scrollWidth > window.innerWidth,
    };
  });
  check(m.panelTop >= 0 && m.panelBottom <= m.vh && m.panelRight <= m.vw + 1, `${tag} drawer: the panel is inside the screen`);
  check(m.bodyHeight >= DRAWER_BODY_MIN, `${tag} drawer: the Body is at least ${DRAWER_BODY_MIN}px tall (${m.bodyHeight})`);
  check(m.bodyScrollTop > 0, `${tag} drawer: the Body scrolls (${m.bodyScrollTop})`);
  check(m.before === m.after && m.before > 0, `${tag} drawer: the Pinned line does not move while the Body scrolls (${m.before} then ${m.after})`);
  check(!m.pageOverflow, `${tag} drawer: no horizontal overflow of the page`);
  if (shot) await page.screenshot({ path: `${shot}-drawer-${tag}.png` });
}

async function main() {
  const opts = parseArgs(process.argv.slice(2));
  const browser = await chromium.launch();
  try {
    for (const vp of VIEWPORTS) {
      const tag = `${vp.width}x${vp.height}`;
      for (const run of [checkDialog, checkDrawer]) {
        const page = await browser.newPage({ viewport: vp, colorScheme: "light" });
        const resp = await page.goto(opts.url, { waitUntil: "networkidle" });
        if (!resp || !resp.ok()) {
          check(false, `${tag}: ${opts.url} loads (status ${resp ? resp.status() : "none"})`);
        } else {
          await run(page, tag, opts.shot);
        }
        await page.close();
      }
    }
  } finally {
    await browser.close();
  }
  if (failures.length > 0) {
    process.stderr.write(`containers-check: ${failures.length} check(s) failed\n`);
    process.exit(1);
  }
}

main().catch((err) => {
  process.stderr.write(`containers-check: ${err.stack || err}\n`);
  process.exit(1);
});
