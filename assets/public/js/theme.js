(function(){
  const KEY = 'theme';
  function apply(theme){
    document.documentElement.setAttribute('data-theme', theme);
  }
  function current(){
    return document.documentElement.getAttribute('data-theme');
  }
  function init(){
    let saved = localStorage.getItem(KEY);
    if(!saved){
      saved = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
    }
    apply(saved);
    const toggle = document.querySelector('[data-theme-toggle]');
    if(toggle){
      toggle.setAttribute('aria-pressed', saved === 'dark');
      toggle.textContent = saved === 'dark' ? '☀️' : '🌙';
    }
    // Emit initial theme change for components needing dynamic styles
    document.dispatchEvent(new CustomEvent('themechange', { detail: { theme: saved } }));
  }
  document.addEventListener('DOMContentLoaded', function(){
    init();
    const btn = document.querySelector('[data-theme-toggle]');
    if(!btn) return;
    btn.addEventListener('click', function(){
      const next = current() === 'dark' ? 'light' : 'dark';
      apply(next);
      localStorage.setItem(KEY, next);
      btn.setAttribute('aria-pressed', next === 'dark');
      btn.textContent = next === 'dark' ? '☀️' : '🌙';
      document.dispatchEvent(new CustomEvent('themechange', { detail: { theme: next } }));
    });
  });
})();
