// splitpane.js - webframe's split pane list links (first-party, not templUI).
//
// A splitpane list link is a plain <a href> carrying data-wf-splitpane-link,
// hx-target (the detail region) and hx-push-url="true", but no hx-get. From
// Tailwind's md breakpoint up (48rem, the same media query the layout uses)
// this script takes the click: it stops the navigation and asks htmx to
// fetch the href into the detail region, with the link as the source so
// htmx reads its hx-target and hx-push-url (the address bar follows the
// pick, Back restores the previous detail). Below md it does nothing, so
// the anchor navigates as a plain page load and the server renders the
// detail view with its Back link.
//
// Why not hx-get with a filtered hx-trigger: htmx calls preventDefault on
// an anchor's click before it evaluates the trigger filter, so a filtered
// link never navigates on a phone.
//
// Failed clicks. When the request for a list link fails, the detail region
// keeps what it showed (htmx-errors.js drops the error answer because the
// region carries data-wf-swallow-errors) and the address bar keeps its
// URL. So the user is not left wondering why nothing happened, this script
// then shows a toast: the pane renders two of them, already worded, inside
// <template data-wf-splitpane-toast="gone|failed"> elements. "gone" is for
// a 404 or 410 answer (the item was deleted since the list was drawn),
// "failed" for any other error status and for a request that got no
// answer. The copy goes into #toast-container, where Alpine starts it like
// any toast. When the pane has ShowErrors and the error page was swapped
// into the region, no toast is shown: the page itself says what happened.
// This listener runs after htmx-errors.js's (appshell.Scripts loads that
// script first), so it sees the final shouldSwap.
(function () {
  'use strict';
  if (window.wf && window.wf.splitpane) return;
  window.wf = window.wf || {};
  window.wf.splitpane = {};
  var wide = window.matchMedia('(min-width: 48rem)');
  document.addEventListener('click', function (evt) {
    if (evt.defaultPrevented || evt.button !== 0 || evt.metaKey || evt.ctrlKey || evt.shiftKey || evt.altKey) return;
    var a = evt.target && evt.target.closest ? evt.target.closest('a[data-wf-splitpane-link]') : null;
    if (!a || !wide.matches || !window.htmx) return;
    var target = a.getAttribute('hx-target');
    if (!target || !document.querySelector(target)) return;
    evt.preventDefault();
    var p = window.htmx.ajax('GET', a.getAttribute('href'), { source: a, target: target, swap: 'innerHTML' });
    if (p && p.catch) p.catch(function () { /* htmx already reported the error */ });
  });

  // paneLink returns the splitpane list link that started a request, or
  // null. requestConfig.elt is the request's source in every htmx event;
  // htmx:beforeSwap's own detail.elt is the swap target instead.
  function paneLink(d) {
    var elt = d && (d.requestConfig && d.requestConfig.elt || d.elt);
    return elt && elt.closest ? elt.closest('a[data-wf-splitpane-link]') : null;
  }

  // notify copies the pane's "gone" or "failed" toast into #toast-container,
  // replacing a toast an earlier failed click left there, so repeated
  // clicks on a missing item show one toast, not a stack.
  function notify(link, which) {
    var pane = link.closest('[data-wf-splitpane]');
    var tpl = pane && pane.querySelector('template[data-wf-splitpane-toast="' + which + '"]');
    var box = document.getElementById('toast-container') || document.body;
    if (!tpl || !tpl.content.firstElementChild) return;
    var old = box.querySelectorAll('[data-wf-splitpane-notice]');
    for (var i = 0; i < old.length; i++) old[i].remove();
    var el = tpl.content.firstElementChild.cloneNode(true);
    el.setAttribute('data-wf-splitpane-notice', which);
    box.appendChild(el);
  }

  document.addEventListener('htmx:beforeSwap', function (evt) {
    var d = evt.detail;
    if (!d || !d.isError || d.shouldSwap) return; // success, or the error page is shown
    var link = paneLink(d);
    if (!link) return;
    var status = d.xhr ? d.xhr.status : 0;
    notify(link, status === 404 || status === 410 ? 'gone' : 'failed');
  });

  // No answer at all (network down, server stopped, request timed out).
  function noAnswer(evt) {
    var link = paneLink(evt.detail);
    if (link) notify(link, 'failed');
  }
  document.addEventListener('htmx:sendError', noAnswer);
  document.addEventListener('htmx:timeout', noAnswer);
})();
