(function(){
  const KEY = 'adjacencyCollapsed';
  function init(){
    const panel = document.querySelector('[data-adjacency-panel]');
    const btn = document.querySelector('[data-adjacency-toggle]');
    if(!panel || !btn) return;

    // Restore state
    const saved = sessionStorage.getItem(KEY);
    const collapsed = saved === 'true';
    apply(collapsed);

    btn.addEventListener('click', function(){
      const next = panel.getAttribute('data-collapsed') === 'true' ? false : true;
      apply(next);
      sessionStorage.setItem(KEY, String(next));
    });

    function apply(isCollapsed){
      panel.setAttribute('data-collapsed', String(isCollapsed));
      btn.setAttribute('aria-expanded', String(!isCollapsed));
      btn.textContent = isCollapsed ? '+' : '−';
      btn.setAttribute('aria-label', isCollapsed ? 'Expand adjacent posts' : 'Collapse adjacent posts');
    }
  }
  document.readyState === 'loading' ? document.addEventListener('DOMContentLoaded', init) : init();
})();
