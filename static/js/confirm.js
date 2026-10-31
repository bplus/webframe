// confirm.js - webframe's htmx:confirm interceptor (first-party, not templUI).
//
// htmx fires htmx:confirm on the triggering element before every request;
// evt.detail.question holds the hx-confirm text (null when there is none).
// When the page has the confirm dialog host (#wf-confirm, rendered by
// appshell.Document through confirmdialog.Host), this script stops the
// browser's native confirm() box, fills the dialog from the trigger's
// data-wf-confirm-* attributes (action.Confirm emits them; the host root
// carries the defaults under the same names) and the question, and opens it
// through dialog.min.js's public API, window.tui.dialog.open(id). Confirm
// issues the request exactly once (evt.detail.issueRequest(true)); Cancel,
// Escape and the backdrop close the dialog through dialog.min.js and issue
// nothing. Focus lands on Cancel when the dialog opens and returns to the
// trigger after. One request is pending at a time; a second htmx:confirm
// while the dialog is open is dropped. Without the host, or before
// dialog.min.js has loaded, the handler returns and htmx shows the native box.
(function () {
  'use strict';
  var HOST = 'wf-confirm';
  var pending = null; // { issue: evt.detail.issueRequest, elt: the trigger }

  function host() { return document.getElementById(HOST); }
  function part(name) { return host().querySelector('[data-wf-confirm-part="' + name + '"]'); }
  function dialog() { return window.tui && window.tui.dialog; }

  // finish settles the pending request (issue it or drop it) and gives focus
  // back to the trigger if it is still on the page.
  function finish(confirmed) {
    var p = pending;
    pending = null;
    if (!p) return;
    if (confirmed) p.issue(true);
    if (p.elt && document.body.contains(p.elt) && typeof p.elt.focus === 'function') p.elt.focus();
  }

  document.addEventListener('htmx:confirm', function (evt) {
    var d = evt.detail;
    if (!d || !d.question) return;              // no hx-confirm: nothing to ask
    var h = host(), api = dialog();
    if (!h || !api) return;                     // no host: the native box runs
    evt.preventDefault();
    if (pending) return;                        // one at a time: this ask is dropped
    var t = d.elt;
    function text(name) { return t.getAttribute(name) || h.getAttribute(name) || ''; }
    var destructive = text('data-wf-confirm-destructive') === 'true';
    part('title').textContent = text('data-wf-confirm-title');
    part('message').textContent = d.question;
    part('cancel').textContent = text('data-wf-confirm-cancel');
    part('confirm').textContent = text('data-wf-confirm-label');
    part('confirm-destructive').textContent = text('data-wf-confirm-label');
    part('confirm').hidden = destructive;
    part('confirm-destructive').hidden = !destructive;
    pending = { issue: d.issueRequest, elt: t };
    api.open(HOST);
    // dialog.min.js shows the panel on the next animation frame; focus Cancel
    // once it is visible (dialog's own autofocus is disabled on this panel).
    requestAnimationFrame(function () { requestAnimationFrame(function () { part('cancel').focus(); }); });
  });

  // The confirm buttons issue the request, then close. Cancel is a
  // dialog.Close, so dialog.min.js closes on it; every close path is settled
  // by the observer below.
  document.addEventListener('click', function (evt) {
    var h = host();
    if (!h || !pending || !evt.target.closest) return;
    var b = evt.target.closest('[data-wf-confirm-part="confirm"], [data-wf-confirm-part="confirm-destructive"]');
    if (!b || !h.contains(b)) return;
    finish(true);
    dialog().close(HOST);
  });

  // dialog.min.js owns open/close (Cancel, Escape, backdrop, close()) and
  // records it in data-tui-dialog-open on the panel; a flip to "false" while
  // a request is still pending is a cancel.
  function watch() {
    var h = host();
    var panel = h && h.querySelector('[data-tui-dialog-content]');
    if (!panel) return;
    new MutationObserver(function () {
      if (panel.getAttribute('data-tui-dialog-open') === 'false') finish(false);
    }).observe(panel, { attributes: true, attributeFilter: ['data-tui-dialog-open'] });
  }
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', watch); else watch();
})();
