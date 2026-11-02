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
(function () {
  'use strict';
  if (window.wf && window.wf.splitpane) return;
  window.wf = window.wf || {};
  window.wf.splitpane = { version: 1 };
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
})();
