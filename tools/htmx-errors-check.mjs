#!/usr/bin/env node
// htmx-errors-check: run static/js/htmx-errors.js in a bare Node context
// (no browser, no npm packages) and check which error answers it lets htmx
// swap. Each case hands the script's htmx:beforeSwap listener an error
// response (a Content-Type and a body) and compares the swap decision and
// the toasts shown. Further cases fire htmx:sendError and htmx:timeout (a
// request that got no answer at all) and count the toasts.
//
//   node tools/htmx-errors-check.mjs
//
// The Go test components/appshell/htmxerrors_test.go runs it when node is on
// PATH. Exit codes: 0 ok, 1 a case failed.

import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import vm from "node:vm";

const script = readFileSync(fileURLToPath(new URL("../static/js/htmx-errors.js", import.meta.url)), "utf8");

// notices holds the LoadFailed toasts the script has put in the fake
// #toast-container (one at most: a new one replaces the old).
let notices = [];

// load runs the script in a fresh context and returns its listeners by
// event type. The
// fake document has the <template data-wf-load-failed> and the
// #toast-container appshell.Document renders.
function load() {
  const listeners = {};
  const box = {
    querySelectorAll: () => notices.slice(),
    appendChild: (el) => notices.push(el),
  };
  const toastEl = {
    cloneNode() {
      const el = {
        attrs: {},
        setAttribute(k, v) { el.attrs[k] = v; },
        remove() { notices = notices.filter((n) => n !== el); },
      };
      return el;
    },
  };
  const document = {
    addEventListener(type, fn) {
      (listeners[type] = listeners[type] || []).push(fn);
    },
    querySelector: (sel) => (sel === "template[data-wf-load-failed]" ? { content: { firstElementChild: toastEl } } : null),
    getElementById: (id) => (id === "toast-container" ? box : null),
  };
  const context = { document };
  context.window = context;
  vm.runInNewContext(script, context);
  if (!listeners["htmx:beforeSwap"]) throw new Error("htmx-errors.js registered no htmx:beforeSwap listener");
  return listeners;
}

// fire hands detail to every listener of type (none is fine: the case
// then fails on its toast count).
function fire(listeners, type, detail) {
  for (const fn of listeners[type] || []) fn({ detail });
}

// target and source build the fake elements a detail carries: a swap
// target that may swallow errors, and a request source that may be a
// split pane list link.
const target = (swallow) => ({ closest: (sel) => (swallow && sel === "[data-wf-swallow-errors]" ? {} : null) });
const source = (paneLink) => ({ closest: (sel) => (paneLink && sel === "a[data-wf-splitpane-link]" ? {} : null) });

// swaps reports htmx's decision after the listener: an error response with
// the given Content-Type (null for none) and body, aimed at a plain target
// or, with swallow, at a data-wf-swallow-errors one.
function swaps(listeners, contentType, body, { swallow = false, isError = true, paneLink = false } = {}) {
  const detail = {
    isError,
    shouldSwap: true,
    xhr: {
      responseText: body,
      getResponseHeader: (name) => (name.toLowerCase() === "content-type" ? contentType : null),
    },
    target: target(swallow),
    requestConfig: { elt: source(paneLink) },
  };
  fire(listeners, "htmx:beforeSwap", detail);
  return detail.shouldSwap;
}

const plain = "text/plain; charset=utf-8";
const html = "text/html; charset=utf-8";
const cases = [
  // The five body shapes from webframe#59.
  ["XML declaration, unlabelled", null, '<?xml version="1.0"?><error>gone</error>', false],
  ["XML element, application/xml", "application/xml", "<error>gone</error>", false],
  ["XML element, unlabelled", null, "<error>gone</error>", false],
  ["Go printing <nil> as text/plain", plain, "<nil>", false],
  ["HTML fragment Go sniffed as text/plain", plain, '<form id="f" hx-post="/x"><input name="a"></form>', true],
  ["HTML fragment, unlabelled, leading space", null, "\n  <tr><td>1</td></tr>", true],
  ["full document, unlabelled", null, "<!DOCTYPE html><html><body>x</body></html>", true],
  ["html element, unlabelled", null, "<html lang=en>", true],
  ["HTML comment first", plain, "<!-- 422 --><section>x</section>", true],
  ["custom element", null, "<wf-thing>x</wf-thing>", true],
  // The rest of the contract, unchanged.
  ["declared text/html, any body", html, "Not HTML at all", true],
  ["plain Go http.Error", plain, "Internal Server Error\n", false],
  ["JSON error", "application/json", '{"error":"x"}', false],
  ["empty body, no Content-Type", null, "", false],
  ["tag-like word that is not an element", null, "<formula>", false],
];

const listener = load();
let failed = 0;
for (const [name, ct, body, want] of cases) {
  notices = [];
  const got = swaps(listener, ct, body);
  // A refused answer is never silent: the LoadFailed toast shows.
  const toasts = notices.length;
  const ok = got === want && toasts === (want ? 0 : 1);
  if (!ok) failed++;
  process.stdout.write(`${ok ? "ok  " : "FAIL"} ${name}: swaps=${got}, want ${want}; toasts=${toasts}\n`);
}
// Two refused answers in a row leave one toast, not a stack.
notices = [];
swaps(listener, plain, "Internal Server Error\n");
swaps(listener, plain, "Internal Server Error\n");
{
  const ok = notices.length === 1;
  if (!ok) failed++;
  process.stdout.write(`${ok ? "ok  " : "FAIL"} two refused answers: toasts=${notices.length}, want 1\n`);
}
// A data-wf-swallow-errors target never swaps an error, even HTML, and
// shows no toast (the next poll repairs it); a success is never touched;
// a split pane link's refused answer is left to splitpane.js's own toast.
for (const [name, run, want] of [
  ["text error from a split pane link", () => swaps(listener, plain, "Internal Server Error\n", { paneLink: true }), false],
  ["HTML error into a swallowing target", () => swaps(listener, html, "<div>x</div>", { swallow: true }), false],
  ["text error into a swallowing target", () => swaps(listener, plain, "Internal Server Error\n", { swallow: true }), false],
  ["success response left to htmx", () => swaps(listener, plain, "ok", { isError: false }), true],
]) {
  notices = [];
  const got = run();
  const ok = got === want && notices.length === 0;
  if (!ok) failed++;
  process.stdout.write(`${ok ? "ok  " : "FAIL"} ${name}: swaps=${got}, want ${want}\n`);
}
// No answer at all: htmx fires htmx:sendError (network error, server
// down) or htmx:timeout, with no response to swap. The toast shows, once,
// except for a swallowing target and a split pane link.
const noAnswer = (type, { swallow = false, paneLink = false } = {}) => () =>
  fire(listener, type, { xhr: { status: 0 }, target: target(swallow), requestConfig: { elt: source(paneLink) } });
for (const [name, steps, want] of [
  ["sendError on a plain target", [noAnswer("htmx:sendError")], 1],
  ["timeout on a plain target", [noAnswer("htmx:timeout")], 1],
  ["two sendErrors in a row", [noAnswer("htmx:sendError"), noAnswer("htmx:sendError")], 1],
  ["sendError after a refused answer", [() => swaps(listener, plain, "x"), noAnswer("htmx:sendError")], 1],
  ["sendError on a swallowing target", [noAnswer("htmx:sendError", { swallow: true })], 0],
  ["timeout on a swallowing target", [noAnswer("htmx:timeout", { swallow: true })], 0],
  ["sendError from a split pane link", [noAnswer("htmx:sendError", { paneLink: true })], 0],
  ["timeout from a split pane link", [noAnswer("htmx:timeout", { paneLink: true })], 0],
]) {
  notices = [];
  for (const step of steps) step();
  const ok = notices.length === want;
  if (!ok) failed++;
  process.stdout.write(`${ok ? "ok  " : "FAIL"} ${name}: toasts=${notices.length}, want ${want}\n`);
}
if (failed) {
  process.stderr.write(`htmx-errors-check: ${failed} case(s) failed\n`);
  process.exit(1);
}
