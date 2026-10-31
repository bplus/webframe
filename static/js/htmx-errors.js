// htmx-errors.js - webframe's error-swap guard (first-party, not templUI).
//
// appshell.Document ships an htmx-config meta that makes htmx swap 4xx and
// 5xx responses like 2xx ones, so a 422 form re-render, a 409 whose only
// body is a toast and a 404 error page all reach the user. Two kinds of
// error answer must not be swapped, though, and htmx cannot tell them apart
// by status alone:
//
//   1. A body that is not HTML: a plain Go http.Error 500 ("Internal Server
//      Error" as text/plain), a JSON error, an empty body with no
//      Content-Type. Swapped, it would land as raw text inside the target.
//   2. A target that is one of the page's live elements: the shell's nav
//      badge poll span, a polled status dot, anything an app marks with
//      data-wf-swallow-errors. An error page is a centered column; inside a
//      badge span it is garbage, and the next poll will repair the element
//      anyway.
//
// htmx fires htmx:beforeSwap with evt.detail.isError (true for the
// [45].. codes under the webframe htmx-config), evt.detail.xhr,
// evt.detail.target and evt.detail.shouldSwap (what htmx decided). This
// listener sets shouldSwap = false when isError is true AND either the
// response's Content-Type header is not text/html (missing counts as not
// HTML) OR the target or one of its ancestors carries
// data-wf-swallow-errors. In every other case shouldSwap is left exactly as
// htmx computed it, so the batch 3 behavior (swap the error body) stands for
// an HTML error answer to an ordinary target. htmx still fires
// htmx:responseError after a refused swap, so an app that listens for it is
// unaffected.
(function () {
  'use strict';
  // htmx re-runs <script src> tags that arrive in swapped content (a boosted
  // navigation swaps the whole body, bundle included); a second copy of
  // this script would add a second listener. Same guard as confirm.js.
  if (window.wf && window.wf.errors) return;
  window.wf = window.wf || {};
  window.wf.errors = { version: 1 };

  // isHTML reports whether the response declared itself text/html. A missing
  // header, or any other media type (text/plain, application/json), is not
  // HTML. The check is on the media type only, so "text/html; charset=utf-8"
  // passes.
  function isHTML(xhr) {
    if (!xhr || typeof xhr.getResponseHeader !== 'function') return false;
    var ct = xhr.getResponseHeader('Content-Type');
    if (!ct) return false;
    return /^\s*text\/html\s*(;|$)/i.test(ct);
  }

  // swallows reports whether the swap target, or an ancestor, is marked
  // data-wf-swallow-errors.
  function swallows(target) {
    return !!(target && typeof target.closest === 'function' && target.closest('[data-wf-swallow-errors]'));
  }

  document.addEventListener('htmx:beforeSwap', function (evt) {
    var d = evt.detail;
    if (!d || !d.isError) return;             // success and 204: htmx's own decision stands
    if (!isHTML(d.xhr) || swallows(d.target)) d.shouldSwap = false;
  });
})();
