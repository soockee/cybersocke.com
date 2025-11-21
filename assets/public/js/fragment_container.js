// Floating fragment containers logic
// Creates resizable, scrollable containers that persist across navigation.
// Multiple containers can be opened. Each container fetches /posts/{slug}/fragment
// and wraps it with draggable header + close/collapse actions.
(function() {
  const STORAGE_KEY = 'floatingFragments.v1';

  function loadSaved() {
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      if (!raw) return [];
      const arr = JSON.parse(raw);
      if (!Array.isArray(arr)) return [];
      return arr.filter(x => x && typeof x.slug === 'string');
    } catch { return []; }
  }

  function saveAll() {
    const states = Array.from(document.querySelectorAll('.floating-fragment')).map(c => serializeContainer(c));
    try { localStorage.setItem(STORAGE_KEY, JSON.stringify(states)); } catch {}
  }

  function serializeContainer(container) {
    const rect = container.getBoundingClientRect();
    return {
      id: container.dataset.id,
      slug: container.dataset.slug,
      top: container.style.top || rect.top + 'px',
      left: container.style.left || rect.left + 'px',
      width: container.style.width || rect.width + 'px',
      height: container.style.height || rect.height + 'px',
      collapsed: container.dataset.collapsed === 'true'
    };
  }

  function persist(container) { // persist single container state
    const all = loadSaved();
    const state = serializeContainer(container);
    const idx = all.findIndex(s => s.id === state.id);
    if (idx >= 0) all[idx] = state; else all.push(state);
    try { localStorage.setItem(STORAGE_KEY, JSON.stringify(all)); } catch {}
  }
  function nextOffset(count) {
    // Stagger containers diagonally
    return 32 + count * 28;
  }

  let idCounter = Date.now();
  function createContainer(slug, preset) {
    const existingCount = document.querySelectorAll('.floating-fragment').length;
    const container = document.createElement('div');
    container.className = 'floating-fragment';
    container.dataset.slug = slug;
    container.dataset.id = String(idCounter++);
    container.style.top = preset?.top || (nextOffset(existingCount) + 'px');
    container.style.left = preset?.left || (nextOffset(existingCount) + 'px');
    if (preset?.width) container.style.width = preset.width;
    if (preset?.height) container.style.height = preset.height;
    if (preset?.collapsed) container.dataset.collapsed = 'true';

    const header = document.createElement('div');
    header.className = 'floating-fragment-header';
    const titleSpan = document.createElement('span');
    titleSpan.className = 'frag-title';
    titleSpan.textContent = slug.replace(/\.md$/, '');
    const actions = document.createElement('div');
    actions.className = 'frag-actions';
    const closeBtn = document.createElement('button');
    closeBtn.type = 'button';
    closeBtn.textContent = '×';
    closeBtn.title = 'Close';
    closeBtn.setAttribute('aria-label', 'Close fragment');
    const collapseBtn = document.createElement('button');
    collapseBtn.type = 'button';
    collapseBtn.textContent = '–';
    collapseBtn.title = 'Collapse/Expand';
    collapseBtn.setAttribute('aria-label', 'Collapse fragment');
    actions.appendChild(collapseBtn);
    actions.appendChild(closeBtn);
    header.appendChild(titleSpan);
    header.appendChild(actions);
    const body = document.createElement('div');
    body.className = 'floating-fragment-body';
    container.appendChild(header);
    container.appendChild(body);

    // Drag logic
    let drag = null;
    header.addEventListener('mousedown', (e) => {
      if (e.target.tagName === 'BUTTON') return; // allow button clicks
      drag = { startX: e.clientX, startY: e.clientY, origX: container.offsetLeft, origY: container.offsetTop };
      document.body.classList.add('no-select');
      e.preventDefault();
    });
    window.addEventListener('mousemove', (e) => {
      if (!drag) return;
      const dx = e.clientX - drag.startX;
      const dy = e.clientY - drag.startY;
      container.style.left = drag.origX + dx + 'px';
      container.style.top = drag.origY + dy + 'px';
    });
    window.addEventListener('mouseup', () => {
      drag = null;
      document.body.classList.remove('no-select');
    });

    collapseBtn.addEventListener('click', () => {
      const collapsed = container.dataset.collapsed === 'true';
      container.dataset.collapsed = (!collapsed).toString();
      collapseBtn.textContent = collapsed ? '–' : '+';
      persist(container);
    });
    closeBtn.addEventListener('click', () => { container.remove(); saveAll(); });

    document.body.appendChild(container);
    fetchFragmentInto(slug, body, titleSpan);
    // Persist after initial creation & after interactions
    persist(container);
    container.addEventListener('mouseup', () => persist(container)); // resize end
    window.addEventListener('mouseup', () => persist(container)); // drag end
    return container;
  }

  function fetchFragmentInto(slug, target, titleSpan) {
    fetch('/posts/' + encodeURIComponent(slug) + '/fragment')
      .then(r => { if (!r.ok) throw new Error('HTTP ' + r.status); return r.text(); })
      .then(html => {
        target.innerHTML = html;
        // Update title if fragment provides <h2>
        const h2 = target.querySelector('.note-fragment h2');
        if (h2) titleSpan.textContent = h2.textContent;
        wireInnerLinks(target);
      })
      .catch(err => {
        target.innerHTML = '<p style="color:#b91c1c">Failed to load fragment: ' + err.message + '</p>';
      });
  }

  function wireInnerLinks(scope) {
    // Open related links in new containers instead of navigating
    scope.querySelectorAll('a.open-related[data-slug]').forEach(a => {
      a.addEventListener('click', (e) => {
        e.preventDefault();
        const slug = a.getAttribute('data-slug');
        if (slug) createContainer(slug);
      });
    });
    // Provide close button inside fragment if present
    scope.querySelectorAll('button.close-fragment[data-action="close"]').forEach(btn => {
      btn.addEventListener('click', (e) => {
        // Close the outer floating container (two levels up)
        const outer = btn.closest('.floating-fragment');
        if (outer) outer.remove();
      });
    });
  }

  function initPopButtons() {
    document.querySelectorAll('.pop-fragment-btn[data-slug]').forEach(btn => {
      btn.addEventListener('click', () => {
        const slug = btn.getAttribute('data-slug');
        if (slug) createContainer(slug);
      });
    });
    // Rehydrate saved containers
    const saved = loadSaved();
    saved.forEach(state => {
      const c = createContainer(state.slug, state);
      if (state.collapsed) c.dataset.collapsed = 'true';
    });
    // Save on unload (final positions)
    window.addEventListener('beforeunload', saveAll);
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initPopButtons);
  } else {
    initPopButtons();
  }
})();
