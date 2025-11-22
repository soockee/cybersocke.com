(function(){
  function read(){
    const s = getComputedStyle(document.documentElement);
    return {
      text: s.getPropertyValue('--color-text').trim() || '#000',
      textMuted: s.getPropertyValue('--color-text-muted').trim() || '#666',
      textInvert: s.getPropertyValue('--color-text-invert').trim() || '#fff',
      accent: s.getPropertyValue('--color-accent').trim() || '#2563eb',
      node: s.getPropertyValue('--color-node-fill').trim() || '#d4d4d8',
      nodeFocus: s.getPropertyValue('--color-node-fill-focus').trim() || '#27272a',
      border: s.getPropertyValue('--color-text-muted').trim() || '#999',
      edge: s.getPropertyValue('--color-text-muted').trim() || '#999'
    };
  }
  window.getGraphThemeColors = read;
})();
