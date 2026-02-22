(function () {
  function getTheme() {
    try {
      return localStorage.getItem('wf-theme') || 'dark';
    } catch (e) {
      return 'dark';
    }
  }

  function setTheme(theme) {
    document.documentElement.setAttribute('data-theme', theme);
    try {
      localStorage.setItem('wf-theme', theme);
    } catch (e) {}
    updateLabels(theme);
  }

  function updateLabels(theme) {
    var label = theme === 'light' ? 'Dark Mode' : 'Light Mode';
    document.querySelectorAll('[data-theme-toggle]').forEach(function (btn) {
      btn.textContent = label;
    });
  }

  function initToggles() {
    document.querySelectorAll('[data-theme-toggle]').forEach(function (btn) {
      if (btn.dataset.themeBound === '1') return;
      btn.dataset.themeBound = '1';
      btn.addEventListener('click', function () {
        var current = document.documentElement.getAttribute('data-theme') || 'dark';
        setTheme(current === 'light' ? 'dark' : 'light');
      });
    });
  }

  document.addEventListener('DOMContentLoaded', function () {
    setTheme(getTheme());
    initToggles();
  });

  document.addEventListener('htmx:afterSwap', function () {
    initToggles();
  });
})();
