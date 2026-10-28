// autocomplete.js — behavior for components/autocomplete (first-party, webframe).
// HTMX fetches the results (the text box carries hx-get and swaps the
// server's Results fragment into the listbox). This script handles the
// rest: MinChars gating, opening, keyboard navigation, picking, closing,
// and clearing the picked value when the text is edited. Every listener
// is delegated on document, so markup swapped in later just works.
(function () {
  'use strict';

  function rootOf(el) { return el.closest('[data-wf-autocomplete]'); }
  function inputOf(root) { return root.querySelector('[data-wf-ac-input]'); }
  function listOf(root) { return root.querySelector('[data-wf-ac-list]'); }
  function hiddenOf(root) { return root.querySelector('[data-wf-ac-value]'); }
  // Pickable options only: aria-disabled rows (e.g. "No matches") are skipped
  // by the arrows, Enter and click.
  var OPTION = '[data-wf-ac-option]:not([aria-disabled="true"])';
  function options(root) { return Array.prototype.slice.call(root.querySelectorAll(OPTION)); }
  function isOpen(root) { return !listOf(root).classList.contains('hidden'); }
  function active(root) { return root.querySelector(OPTION + '[data-active]'); }

  function setActive(root, opt) {
    options(root).forEach(function (o) {
      o.removeAttribute('data-active');
      o.removeAttribute('aria-selected');
    });
    var input = inputOf(root);
    if (!opt) { input.removeAttribute('aria-activedescendant'); return; }
    opt.setAttribute('data-active', '');
    opt.setAttribute('aria-selected', 'true');
    opt.scrollIntoView({ block: 'nearest' });
    input.setAttribute('aria-activedescendant', opt.id);
  }

  function open(root) {
    listOf(root).classList.remove('hidden');
    inputOf(root).setAttribute('aria-expanded', 'true');
  }

  function close(root) {
    listOf(root).classList.add('hidden');
    inputOf(root).setAttribute('aria-expanded', 'false');
    setActive(root, null);
  }

  function setHidden(root, value) {
    var hidden = hiddenOf(root);
    if (!hidden || hidden.value === value) return;
    hidden.value = value;
    hidden.dispatchEvent(new Event('change', { bubbles: true }));
  }

  // pick: label into the text box, value into the hidden field, close,
  // and announce the choice on the root. The label is remembered on the
  // root so a late results swap does not reopen the list over the pick.
  function pick(option) {
    var root = rootOf(option);
    var value = option.getAttribute('data-wf-ac-item-value') || '';
    var label = option.getAttribute('data-wf-ac-item-label') || '';
    inputOf(root).value = label;
    root.wfAcPicked = label;
    setHidden(root, value);
    close(root);
    inputOf(root).focus();
    root.dispatchEvent(new CustomEvent('wf:autocomplete:select', {
      bubbles: true, detail: { value: value, label: label }
    }));
  }

  // 1. Too few characters: cancel the request and close the list.
  document.addEventListener('htmx:beforeRequest', function (e) {
    var input = e.detail && e.detail.elt;
    if (!input || !input.matches || !input.matches('[data-wf-ac-input]')) return;
    var root = rootOf(input);
    var min = parseInt(root.getAttribute('data-wf-ac-min'), 10) || 1;
    if (input.value.trim().length < min) {
      e.preventDefault();
      listOf(root).innerHTML = '';
      close(root);
    }
  });

  // 2. Results arrived: show the list with nothing highlighted, but only
  // while the user is still in the field (a late response must not reopen
  // a list they already left or closed) and the text is not the label they
  // just picked.
  document.addEventListener('htmx:afterSwap', function (e) {
    var list = e.detail && e.detail.target;
    if (!list || !list.matches || !list.matches('[data-wf-ac-list]')) return;
    var root = rootOf(list);
    setActive(root, null);
    if (root.wfAcPicked !== undefined && inputOf(root).value === root.wfAcPicked) return;
    if (root.contains(document.activeElement)) open(root);
  });

  // 3. Editing the text invalidates any earlier pick.
  document.addEventListener('input', function (e) {
    if (!e.target.matches('[data-wf-ac-input]')) return;
    var root = rootOf(e.target);
    root.wfAcPicked = undefined;
    setHidden(root, '');
  });

  // 4. Keyboard: arrows move the highlight (wrapping), Enter picks it,
  // Escape closes. Registered on window in the capture phase so it runs
  // before the vendored dialog/slideover scripts, which close on any
  // document-level Escape and ignore defaultPrevented: when Escape closes
  // an open list, stopPropagation keeps the dialog underneath open. Every
  // other key passes through untouched.
  window.addEventListener('keydown', function (e) {
    // IME composition owns Enter/arrows (keyCode 229 covers browsers that
    // do not set isComposing on the composing keydown).
    if (e.isComposing || e.keyCode === 229) return;
    if (!e.target.matches || !e.target.matches('[data-wf-ac-input]')) return;
    var root = rootOf(e.target);
    var opts = options(root);
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
      if (!isOpen(root) || opts.length === 0) return;
      e.preventDefault();
      opts.forEach(function (o, i) { if (!o.id) o.id = e.target.id + '-opt-' + i; });
      var i = opts.indexOf(active(root));
      var step = e.key === 'ArrowDown' ? 1 : -1;
      if (i === -1) i = step === 1 ? 0 : opts.length - 1;
      else i = (i + step + opts.length) % opts.length;
      setActive(root, opts[i]);
    } else if (e.key === 'Enter') {
      var cur = active(root);
      if (isOpen(root) && cur) { e.preventDefault(); pick(cur); }
    } else if (e.key === 'Escape') {
      // preventDefault keeps a type=search box from also clearing its text.
      if (isOpen(root)) { e.preventDefault(); e.stopPropagation(); close(root); }
    }
  }, true);

  // 5. Keep focus in the text box when anything in the list is pressed
  // (an option, the scrollbar, padding, the "No matches" row), so the
  // focusout handler never closes the list before the click lands and the
  // user can keep typing after a pick (also covers touch browsers that
  // would not focus a tabindex=-1 div).
  document.addEventListener('mousedown', function (e) {
    if (e.target.closest && e.target.closest('[data-wf-ac-list]')) e.preventDefault();
  });

  // 6. Click an option to pick it; a click outside any root closes all.
  document.addEventListener('click', function (e) {
    var option = e.target.closest && e.target.closest(OPTION);
    if (option) { pick(option); return; }
    var inside = rootOf(e.target);
    document.querySelectorAll('[data-wf-autocomplete]').forEach(function (root) {
      if (root !== inside && isOpen(root)) close(root);
    });
  });

  // 7. Focus leaving the root closes its list.
  document.addEventListener('focusout', function (e) {
    var root = e.target.closest && rootOf(e.target);
    if (!root) return;
    if (e.relatedTarget && root.contains(e.relatedTarget)) return;
    if (isOpen(root)) close(root);
  });

  window.wf = window.wf || {};
  window.wf.autocomplete = { pick: pick, close: close };
})();
