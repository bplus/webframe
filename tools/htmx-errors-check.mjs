#!/usr/bin/env node
// htmx-errors-check: run static/js/htmx-errors.js in a bare Node context
// (no browser, no npm packages) and check which error answers it lets htmx
// swap. Each case hands the script's htmx:beforeSwap listener an error
// response (a Content-Type and a body) and compares the swap decision.
//
//   node tools/htmx-errors-check.mjs
//
// The Go test components/appshell/htmxerrors_test.go runs it when node is on
// PATH. Exit codes: 0 ok, 1 a case failed.

import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import vm from "node:vm";

const script = readFileSync(fileURLToPath(new URL("../static/js/htmx-errors.js", import.meta.url)), "utf8");

// load runs the script in a fresh context and returns its listener.
function load() {
  let listener = null;
  const document = {
    addEventListener(type, fn) {
      if (type === "htmx:beforeSwap") listener = fn;
    },
  };
  const context = { document };
  context.window = context;
  vm.runInNewContext(script, context);
  if (!listener) throw new Error("htmx-errors.js registered no htmx:beforeSwap listener");
  return listener;
}

// swaps reports htmx's decision after the listener: an error response with
// the given Content-Type (null for none) and body, aimed at a plain target
// or, with swallow, at a data-wf-swallow-errors one.
function swaps(listener, contentType, body, { swallow = false, isError = true } = {}) {
  const detail = {
    isError,
    shouldSwap: true,
    xhr: {
      responseText: body,
      getResponseHeader: (name) => (name.toLowerCase() === "content-type" ? contentType : null),
    },
    target: { closest: (sel) => (swallow && sel === "[data-wf-swallow-errors]" ? {} : null) },
  };
  listener({ detail });
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
  const got = swaps(listener, ct, body);
  const ok = got === want;
  if (!ok) failed++;
  process.stdout.write(`${ok ? "ok  " : "FAIL"} ${name}: swaps=${got}, want ${want}\n`);
}
// A data-wf-swallow-errors target never swaps an error, even HTML; a
// success is never touched.
for (const [name, got, want] of [
  ["HTML error into a swallowing target", swaps(listener, html, "<div>x</div>", { swallow: true }), false],
  ["success response left to htmx", swaps(listener, plain, "ok", { isError: false }), true],
]) {
  const ok = got === want;
  if (!ok) failed++;
  process.stdout.write(`${ok ? "ok  " : "FAIL"} ${name}: swaps=${got}, want ${want}\n`);
}
if (failed) {
  process.stderr.write(`htmx-errors-check: ${failed} case(s) failed\n`);
  process.exit(1);
}
