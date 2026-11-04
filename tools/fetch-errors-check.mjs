#!/usr/bin/env node
// fetch-errors-check: load the demo in Playwright's Chromium and check what
// a failed htmx request shows the user (see tools/README.md):
//
//   refresh  on the showcase (/), click "Refresh stats": a button with
//            hx-get and no hx-target, answered errorpage.NotFound. The
//            "Could not load that" toast shows, the button keeps its label
//            and the page keeps its content. At 1280 and at 390.
//   boom     click "Request a plain 500" (a text/plain error that
//            htmx-errors.js refuses to swap): the same toast shows and the
//            sentence beside the button does not change.
//   noanswer click "Refresh stats" while its request gets no answer
//            (Playwright aborts it, as a stopped server would: htmx fires
//            htmx:sendError), then while it hangs past htmx.config.timeout
//            (htmx:timeout): each time the same toast shows, and only one.
//   repeat   on /settings, a refused "Delete account" (a 409 answered by
//            toast.Respond) twice leaves one toast; on the showcase the
//            Error toast button twice leaves one, and the Warning button
//            then adds a second (different toasts still stack).
//   signin   on the sign-in page (a RenderBare page), an htmx POST button
//            added to the page whose target is itself gets the 404 page,
//            and the page lands in the page's <main id="wf-main">, not
//            inside the button.
//
//   node tools/fetch-errors-check.mjs --url http://localhost:3333
//
// Exit codes: 0 ok, 1 a check failed, 2 usage.

import { chromium } from "playwright";

const USAGE = `usage: node tools/fetch-errors-check.mjs --url <base> [--shot <prefix>]
  --url <base>     the demo's base URL, such as http://localhost:3333 (required)
  --shot <prefix>  also save <prefix>-1280.png, <prefix>-390.png, <prefix>-noanswer.png, <prefix>-repeat.png and <prefix>-signin.png`;

function usage(msg) {
  process.stderr.write(`fetch-errors-check: ${msg}\n${USAGE}\n`);
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
  opts.url = opts.url.replace(/\/+$/, "");
  return opts;
}

const failures = [];
function check(ok, what) {
  process.stdout.write(`${ok ? "ok  " : "FAIL"} ${what}\n`);
  if (!ok) failures.push(what);
}

const opts = parseArgs(process.argv.slice(2));
const NOTICE = "#toast-container [data-wf-load-failed-notice], #toast-container [role=alert]";

// watchConsole collects console errors other than the ones a failed
// request logs by itself (the browser's "Failed to load resource" and
// htmx's "Response Status Error Code", and its own htmx:sendError and
// htmx:timeout log lines): those are the point of the check.
function watchConsole(page) {
  const errors = [];
  page.on("console", (m) => {
    if (m.type() === "error" && !/Failed to load resource|Response Status Error Code (404|409|500)|^htmx:(afterRequest|sendError|timeout)$/.test(m.text())) errors.push(m.text());
  });
  return errors;
}

// toastText waits for a toast in #toast-container and returns its text, or
// "" when none appears.
async function toastText(page) {
  const t = page.locator(NOTICE);
  try {
    await t.first().waitFor({ state: "visible", timeout: 3000 });
  } catch {
    return "";
  }
  return (await t.allInnerTexts()).join(" | ");
}

const clearToasts = (page) => page.evaluate(() => document.querySelectorAll("#toast-container > *").forEach((t) => t.remove()));

// refresh clicks the showcase's Refresh stats button and checks the toast,
// the button and the page.
async function refresh(page, width) {
  const button = page.locator("#refresh-stats");
  await button.scrollIntoViewIfNeeded();
  const before = (await button.innerText()).trim();
  const mainBefore = await page.locator("#wf-main").innerText();
  await button.click();
  const text = await toastText(page);
  check(/Could not load that/.test(text), `${width} refresh: toast reads "${text}"`);
  await page.waitForTimeout(200);
  check((await button.innerText()).trim() === before && before === "Refresh stats", `${width} refresh: button still reads "${before}"`);
  check(!/Page not found/.test(await button.innerHTML()), `${width} refresh: no 404 page inside the button`);
  check((await page.locator("#wf-main").innerText()) === mainBefore, `${width} refresh: page content unchanged`);
}

const browser = await chromium.launch();
try {
  // --- desktop ---
  const page = await browser.newPage({ viewport: { width: 1280, height: 800 } });
  const errors = watchConsole(page);
  await page.goto(opts.url + "/");
  await refresh(page, "1280");
  if (opts.shot) await page.screenshot({ path: `${opts.shot}-1280.png` });

  await clearToasts(page);
  const boomText = await page.locator("#boom-result").innerText();
  await page.getByRole("button", { name: "Request a plain 500" }).click();
  const t = await toastText(page);
  check(/Could not load that/.test(t), `boom: toast reads "${t}"`);
  check((await page.locator("#boom-result").innerText()) === boomText, "boom: the sentence beside the button is unchanged");
  // A second failure replaces the toast rather than stacking a second one.
  await page.getByRole("button", { name: "Request a plain 500" }).click();
  await page.waitForTimeout(300);
  check((await page.locator("#toast-container [data-wf-load-failed-notice]").count()) === 1, "boom: a second click keeps one toast");

  // No answer at all: the request is aborted, then it times out.
  await clearToasts(page);
  await page.route("**/api/stats", (route) => route.abort("connectionrefused"));
  const refreshButton = page.locator("#refresh-stats");
  await refreshButton.click();
  const sent = await toastText(page);
  check(/Could not load that/.test(sent), `noanswer: aborted request, toast reads "${sent}"`);
  await refreshButton.click();
  await page.waitForTimeout(300);
  check((await page.locator("#toast-container > *").count()) === 1, "noanswer: a second aborted click keeps one toast");
  check((await refreshButton.innerText()).trim() === "Refresh stats", "noanswer: the button keeps its label");
  await page.unroute("**/api/stats");
  await clearToasts(page);
  await page.evaluate(() => { window.htmx.config.timeout = 200; });
  await page.route("**/api/stats", async (route) => {
    await new Promise((r) => setTimeout(r, 1500));
    try { await route.abort(); } catch { /* htmx gave up first */ }
  });
  await refreshButton.click();
  const timedOut = await toastText(page);
  check(/Could not load that/.test(timedOut), `noanswer: timed-out request, toast reads "${timedOut}"`);
  await page.waitForTimeout(1600);
  check((await page.locator("#toast-container > *").count()) === 1, "noanswer: a timeout shows one toast");
  if (opts.shot) await page.screenshot({ path: `${opts.shot}-noanswer.png` });
  await page.evaluate(() => { window.htmx.config.timeout = 0; });
  await page.unroute("**/api/stats");

  check(errors.length === 0, `desktop: no other console errors ${JSON.stringify(errors)}`);
  await page.close();

  // --- phone ---
  const phone = await browser.newPage({ viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true, deviceScaleFactor: 2 });
  const phoneErrors = watchConsole(phone);
  await phone.goto(opts.url + "/");
  await refresh(phone, "390");
  if (opts.shot) await phone.screenshot({ path: `${opts.shot}-390.png` });
  check(phoneErrors.length === 0, `phone: no other console errors ${JSON.stringify(phoneErrors)}`);
  await phone.close();

  // --- repeat: an identical server toast replaces the earlier copy ---
  const rep = await browser.newPage({ viewport: { width: 1280, height: 800 } });
  const repErrors = watchConsole(rep);
  const toastCount = async () => { await rep.waitForTimeout(400); return rep.locator("#toast-container > *").count(); };
  await rep.goto(opts.url + "/settings");
  for (let i = 0; i < 2; i++) {
    await rep.locator("#wf-main").getByRole("button", { name: "Delete account" }).click();
    await rep.waitForTimeout(300); // the dialog's open transition
    const confirm = rep.locator("#wf-confirm [data-wf-confirm-part=confirm-destructive]");
    await confirm.click({ timeout: 5000 });
    await confirm.waitFor({ state: "hidden", timeout: 3000 });
    await rep.locator("#toast-container").getByText("Could not delete the account").first().waitFor({ timeout: 3000 });
    await rep.waitForTimeout(300); // the dialog's close transition
  }
  check((await toastCount()) === 1, "repeat: two refused deletes leave one toast");
  if (opts.shot) await rep.screenshot({ path: `${opts.shot}-repeat.png` });
  await rep.goto(opts.url + "/");
  const toastSection = rep.locator("#toast");
  await toastSection.getByRole("button", { name: "Error", exact: true }).click();
  await toastSection.getByRole("button", { name: "Error", exact: true }).click();
  check((await toastCount()) === 1, "repeat: the Error toast button twice leaves one toast");
  await toastSection.getByRole("button", { name: "Warning", exact: true }).click();
  check((await toastCount()) === 2, "repeat: a different toast still stacks");
  check(repErrors.length === 0, `repeat: no other console errors ${JSON.stringify(repErrors)}`);
  await rep.close();

  // --- sign-in page: an action that targets itself lands in #wf-main ---
  const bare = await browser.newPage({ viewport: { width: 1280, height: 800 } });
  const bareErrors = watchConsole(bare);
  await bare.goto(opts.url + "/signin");
  check((await bare.locator("main#wf-main .wf-authpage").count()) === 1, "signin: the page sits in <main id=\"wf-main\">");
  // The probe is a POST with no hx-target to an address the demo does not
  // serve, so errorpage.Handler answers it with the 404 page.
  await bare.evaluate(() => {
    const b = document.createElement("button");
    b.id = "probe";
    b.textContent = "Probe";
    b.setAttribute("hx-post", "/no-such-action");
    document.querySelector(".wf-authpage").appendChild(b);
    window.htmx.process(b);
  });
  await bare.locator("#probe").click();
  try {
    await bare.locator("#wf-main").getByText("Page not found").first().waitFor({ timeout: 3000 });
  } catch { /* checked below */ }
  const main = await bare.locator("#wf-main").innerText();
  check(/Page not found/.test(main), "signin: the 404 page replaced the main region");
  check((await bare.locator("#probe").count()) === 0, "signin: the button that sent it is gone with the old content");
  check((await bare.locator("button#probe .wf-error-page, #probe main").count()) === 0, "signin: nothing landed inside the button");
  if (opts.shot) await bare.screenshot({ path: `${opts.shot}-signin.png` });
  check(bareErrors.length === 0, `signin: no other console errors ${JSON.stringify(bareErrors)}`);
  await bare.close();
} finally {
  await browser.close();
}

if (failures.length) {
  process.stderr.write(`fetch-errors-check: ${failures.length} check(s) failed\n`);
  process.exit(1);
}
