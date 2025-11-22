(function(){
  const side = document.querySelector('[data-explorer-side]');
  const toggle = document.querySelector('[data-side-toggle]');
  if(!side || !toggle) return;
  const KEY = 'explorerSideCollapsed';
  const apply = (collapsed) => {
    side.setAttribute('data-collapsed', collapsed ? 'true' : 'false');
    toggle.setAttribute('aria-expanded', collapsed ? 'false' : 'true');
    toggle.setAttribute('aria-label', collapsed ? 'Expand side panel' : 'Collapse side panel');
    toggle.textContent = collapsed ? '+' : '−';
  };
  let initial = false;
  try { initial = sessionStorage.getItem(KEY) === 'true'; } catch(e) {}
  apply(initial);
  toggle.addEventListener('click', () => {
    const collapsed = side.getAttribute('data-collapsed') === 'true';
    const next = !collapsed;
    apply(next);
    try { sessionStorage.setItem(KEY, next ? 'true' : 'false'); } catch(e) {}
    // If expanding, trigger a resize event so graphs can re-fit
    if(!next) {
      window.dispatchEvent(new Event('resize'));
    }
  });
})();
