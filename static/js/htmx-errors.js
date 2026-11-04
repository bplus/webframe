// htmx-errors.js - webframe's error-swap guard (first-party, not templUI).
//
// appshell.Document ships an htmx-config meta that makes htmx swap 4xx and
// 5xx responses like 2xx ones, so a 422 form re-render, a 409 whose only
// body is a toast and a 404 error page all reach the user. Two kinds of
// error answer must not be swapped, though, and htmx cannot tell them apart
// by status alone:
//
//   1. A body that is not HTML: a plain Go http.Error 500 ("Internal Server
//      Error" as text/plain), a JSON error, an XML error, an empty body with
//      no Content-Type. Swapped, it would land as raw text or stray markup
//      inside the target. A body that IS HTML but was never labelled is
//      still swapped: a handler that calls WriteHeader(422) and renders a
//      fragment without appshell.SetHeaders gets Go's content sniffer,
//      which does not know fragments starting with <form, <section, <nav,
//      <span, <ul or <tr and labels them text/plain. So a missing or
//      non-HTML Content-Type is overruled when the body starts with a known
//      HTML opening: <!doctype html, an HTML comment, an HTML element name
//      (<form, <div, <tr, ...) or a custom element (a name with a hyphen).
//      Anything else that starts with "<" is not HTML: an XML declaration
//      (<?xml), an XML element such as <error>, or text that merely starts
//      with a bracket, like Go printing <nil>. A declared text/plain body
//      gets the same test, because the browser cannot tell Go's sniffed
//      text/plain from a handler's own; to be sure an error swaps, label it
//      text/html (appshell.SetHeaders, or answer it through errorpage).
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
// response is not HTML (its Content-Type header is not text/html, missing
// counts as not HTML, and its body does not start with a known HTML
// opening) OR the target
// or one of its ancestors carries data-wf-swallow-errors. In every other case shouldSwap is left exactly as
// htmx computed it, so the default (swap the error body) stands for
// an HTML error answer to an ordinary target. htmx still fires
// htmx:responseError after a refused swap, so an app that listens for it is
// unaffected.
//
// No error is silent, except on a swallowing target. When the answer is
// refused because it is not HTML (and the target does not swallow errors),
// this script shows the "could not load" toast: appshell.Document renders
// it, already worded (appshell.LoadFailed), inside
// <template data-wf-load-failed>, and the copy goes into #toast-container,
// where Alpine starts it like any toast. A toast an earlier refused answer
// left there is replaced, so repeated failures show one toast, not a
// stack. An HTML error answer to a GET that targets the element that sent
// it is handled on the server: errorpage.Respond answers it with the same
// toast and HX-Reswap none.
(function () {
  'use strict';
  // htmx re-runs <script src> tags that arrive in swapped content (a boosted
  // navigation swaps the whole body, bundle included); a second copy of
  // this script would add a second listener. Same guard as confirm.js.
  if (window.wf && window.wf.errors) return;
  window.wf = window.wf || {};
  window.wf.errors = {};

  // declaredHTML reports whether the response declared itself text/html. A
  // missing header, or any other media type (text/plain, application/json),
  // is not HTML. The check is on the media type only, so "text/html;
  // charset=utf-8" passes.
  function declaredHTML(xhr) {
    if (!xhr || typeof xhr.getResponseHeader !== 'function') return false;
    var ct = xhr.getResponseHeader('Content-Type');
    if (!ct) return false;
    return /^\s*text\/html\s*(;|$)/i.test(ct);
  }

  // htmlStart matches the openings looksLikeHTML accepts, after leading
  // whitespace and case-insensitively: <!doctype html, <!--, or "<" and an
  // HTML element name (or a custom element name, which has a hyphen)
  // followed by whitespace, "/" or ">". <?xml, <error> and <nil> fail.
  var htmlStart = new RegExp(
    '^\\s*(?:<!doctype\\s+html|<!--|<(?:' +
    'a|abbr|article|aside|audio|b|blockquote|body|br|button|canvas|caption|code|col|colgroup|' +
    'dd|details|dialog|div|dl|dt|em|fieldset|figcaption|figure|footer|form|' +
    'h[1-6]|head|header|hr|html|i|iframe|img|input|kbd|label|legend|li|link|main|mark|' +
    'meta|meter|nav|noscript|ol|optgroup|option|output|p|picture|pre|progress|' +
    's|script|section|select|small|span|strong|style|sub|summary|sup|svg|' +
    'table|tbody|td|template|textarea|tfoot|th|thead|time|title|tr|u|ul|video|' +
    '[a-z][a-z0-9]*-[a-z0-9-]*' +
    ')(?=[\\s/>]))',
    'i'
  );

  // looksLikeHTML reports whether the body starts with a known HTML
  // opening: a fragment the handler forgot to label. responseText throws
  // for a non-text responseType; htmx uses the default, but guard anyway.
  function looksLikeHTML(xhr) {
    if (!xhr) return false;
    var text;
    try { text = xhr.responseText; } catch (e) { return false; }
    return typeof text === 'string' && htmlStart.test(text);
  }

  // isHTML: declared text/html, or a body that starts with a known HTML
  // opening whatever its label.
  function isHTML(xhr) {
    return declaredHTML(xhr) || looksLikeHTML(xhr);
  }

  // swallows reports whether the swap target, or an ancestor, is marked
  // data-wf-swallow-errors.
  function swallows(target) {
    return !!(target && typeof target.closest === 'function' && target.closest('[data-wf-swallow-errors]'));
  }

  // notify copies the page's LoadFailed toast into #toast-container,
  // replacing one an earlier refused answer left there.
  function notify() {
    if (typeof document.querySelector !== 'function') return;
    var tpl = document.querySelector('template[data-wf-load-failed]');
    if (!tpl || !tpl.content || !tpl.content.firstElementChild) return;
    var box = document.getElementById('toast-container') || document.body;
    if (!box) return;
    var old = box.querySelectorAll('[data-wf-load-failed-notice]');
    for (var i = 0; i < old.length; i++) old[i].remove();
    var el = tpl.content.firstElementChild.cloneNode(true);
    el.setAttribute('data-wf-load-failed-notice', '');
    box.appendChild(el);
  }

  document.addEventListener('htmx:beforeSwap', function (evt) {
    var d = evt.detail;
    if (!d || !d.isError) return;             // success and 204: htmx's own decision stands
    if (swallows(d.target)) {
      d.shouldSwap = false;                   // a live element: the next poll repairs it
    } else if (!isHTML(d.xhr)) {
      d.shouldSwap = false;
      notify();
    }
  });
})();
