// Monaco initialization and edit workflow with diff review step
const form = document.getElementById('edit-form');
const container = document.getElementById('monaco-container');
const reviewBtn = document.getElementById('review-btn');
const restoreDraftBtn = document.getElementById('restore-draft-btn');
const diffPanel = document.getElementById('diff-panel');
const diffContainer = document.getElementById('monaco-diff-container');
const updateTsCheckbox = document.getElementById('update-timestamp');
const tsPreview = document.getElementById('timestamp-preview');
const applySaveBtn = document.getElementById('apply-save');
const backBtn = document.getElementById('back-to-edit');
const csrfTokenEl = document.getElementById('csrf-token');
const csrfToken = csrfTokenEl ? csrfTokenEl.value : null;
const draftStatusEl = document.getElementById('draft-status');
let editor; // main editor
let diffEditor; // diff editor instance
let originalContent = null; // captured original for diff
let draftTimer;
// Load injected post contract for future language/schema validation.
const contractEl = document.getElementById('post-contract');
let postContract = {};
if (contractEl) {
  try { postContract = JSON.parse(contractEl.textContent || '{}'); }
  catch (e) { console.warn('Invalid post contract JSON', e); }
}

function initMonaco() {
  if (!window.require || !container) return;
  const initial = container.textContent;
  originalContent = initial; // store original for diff
  container.textContent = ''; // clear raw content placeholder
  window.require.config({ paths: { 'vs': 'https://cdnjs.cloudflare.com/ajax/libs/monaco-editor/0.44.0/min/vs' } });
  window.require(['vs/editor/editor.main'], () => {
    editor = monaco.editor.create(container, {
      value: initial,
      language: 'markdown',
      theme: 'vs-dark',
      automaticLayout: true,
      minimap: { enabled: false },
      wordWrap: 'on'
    });
    startAutosave();
  });ö
}

function showDiffView() {
  if (!editor || !diffContainer) return;
  // Hide main editor container while diff is shown for clarity
  container.style.display = 'none';
  diffPanel.style.display = 'block';
  form.querySelector('.edit-actions').style.display = 'none';
  // Timestamp preview
  if (tsPreview) tsPreview.textContent = new Date().toISOString();
  if (diffEditor) {
    // Update modified editor value
    const modifiedEditor = diffEditor.getModifiedEditor();
    if (modifiedEditor) modifiedEditor.setValue(editor.getValue());
    diffEditor.layout();
    return;
  }
  const originalModel = monaco.editor.createModel(originalContent || editor.getValue(), 'markdown');
  const modifiedModel = monaco.editor.createModel(editor.getValue(), 'markdown');
  diffEditor = monaco.editor.createDiffEditor(diffContainer, {
    theme: 'vs-dark',
    renderSideBySide: false,
    enableSplitViewResizing: false,
    automaticLayout: true,
    originalEditable: false,
    minimap: { enabled: false },
    wordWrap: 'on'
  });
  diffEditor.setModel({ original: originalModel, modified: modifiedModel });
  diffEditor.layout();
}

function startAutosave() {
  if (!editor) return;
  const slug = form.getAttribute('data-slug');
  draftTimer = setInterval(() => {
    const val = editor.getValue();
    localStorage.setItem(`draft:${slug}`, val);
    updateDraftStatus(false);
  }, 30000);
}

function restoreDraft() {
  if (!editor) return;
  const slug = form.getAttribute('data-slug');
  const draft = localStorage.getItem(`draft:${slug}`);
  if (draft) {
    // Only restore if content differs (avoid stomping loaded version)
    if (draft !== editor.getValue()) {
      editor.setValue(draft);
      updateDraftStatus(true);
    }
  }
}

function patchUpdatedTimestamp(raw) {
  const now = new Date().toISOString();
  // frontmatter match (--- then content then ---)
  const fmMatch = raw.match(/^---\n([\s\S]*?)\n---\n?/);
  if (!fmMatch) return raw; // no frontmatter
  const fmBody = fmMatch[1];
  const lines = fmBody.split(/\n/);
  let found = false;
  for (let i = 0; i < lines.length; i++) {
    if (/^updated:/i.test(lines[i])) {
      lines[i] = `updated: ${now}`;
      found = true;
      break;
    }
  }
  if (!found) {
    // insert before end
    lines.push(`updated: ${now}`);
  }
  const newFm = `---\n${lines.join("\n")}\n---\n`;
  const remainder = raw.slice(fmMatch[0].length);
  return newFm + remainder;
}

async function saveRaw(raw) {
  try {
    const resp = await fetch(form.action, {
      method: 'POST',
      headers: {
        'X-CSRF-Token': csrfToken || '',
        'Content-Type': 'text/plain; charset=utf-8'
      },
      body: raw
    });
    if (resp.status === 303 || resp.ok) {
      const loc = resp.headers.get('Location') || '/admin';
      window.location = loc;
    } else {
      alert('Save failed (' + resp.status + ')');
      console.error('Save failed response', resp);
    }
  } catch (err) {
    console.error('Save failed', err);
    alert('Save failed due to network error');
  }
}

if (form) {
  form.addEventListener('submit', (e) => {
    // Default submit path (without diff review) - raw save
    e.preventDefault();
    if (!editor) return;
    saveRaw(editor.getValue());
  });
}

if (reviewBtn) {
  reviewBtn.addEventListener('click', () => {
    showDiffView();
  });
}

if (backBtn) {
  backBtn.addEventListener('click', () => {
    diffPanel.style.display = 'none';
    form.querySelector('.edit-actions').style.display = 'flex';
    container.style.display = 'block';
    // Dispose diff editor and its models so a fresh diff is created next time
    if (diffEditor) {
      const modelPair = diffEditor.getModel();
      try { if (modelPair && modelPair.original) modelPair.original.dispose(); } catch (e) { /* noop */ }
      try { if (modelPair && modelPair.modified) modelPair.modified.dispose(); } catch (e) { /* noop */ }
      diffEditor.dispose();
      diffEditor = null;
    }
  });
}

if (applySaveBtn) {
  applySaveBtn.addEventListener('click', () => {
    if (!editor) return;
    let raw = editor.getValue();
    if (updateTsCheckbox && updateTsCheckbox.checked) {
      raw = patchUpdatedTimestamp(raw);
    }
    saveRaw(raw);
  });
}

if (restoreDraftBtn) {
  restoreDraftBtn.addEventListener('click', () => {
    if (!editor) return;
    const slug = form.getAttribute('data-slug');
    const draft = localStorage.getItem(`draft:${slug}`);
    if (!draft) {
      alert('No draft found for this post.');
      return;
    }
    if (draft === editor.getValue()) {
      alert('Draft is identical to current editor content.');
      return;
    }
    if (confirm('Restore draft from local storage? This will overwrite current editor content.')) {
      editor.setValue(draft);
      updateDraftStatus(true);
    }
  });
}

function updateDraftStatus(manual) {
  if (!draftStatusEl) return;
  const ts = new Date().toLocaleTimeString();
  draftStatusEl.textContent = (manual ? 'Draft saved ' : 'Autosaved draft ') + ts + '. Ctrl+S saves.';
}

function saveDraftExplicit() {
  if (!editor) return;
  const slug = form.getAttribute('data-slug');
  localStorage.setItem(`draft:${slug}`, editor.getValue());
  updateDraftStatus(true);
}

window.addEventListener('keydown', (e) => {
  if ((e.ctrlKey || e.metaKey) && e.key === 's') {
    e.preventDefault();
    saveDraftExplicit();
  }
});

window.addEventListener('load', () => {
  // Initialize editor with current remote content only; do not auto-restore
  // potentially stale local draft that would mask the latest saved version.
  initMonaco();
});
