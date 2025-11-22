const table = document.getElementById('admin-posts-table');
if (table) {
  const csrf = table.getAttribute('data-csrf');
  table.addEventListener('click', async (e) => {
    const btn = e.target.closest('button[data-action="delete"]');
    if (!btn) return;
    const slug = btn.getAttribute('data-slug');
    if (!slug) return;
    if (!confirm(`Delete post ${slug}? This cannot be undone.`)) return;
    try {
      const resp = await fetch(`/admin/posts/${slug}/delete`, {
        method: 'POST',
        headers: { 'X-CSRF-Token': csrf }
      });
      if (!resp.ok) {
        alert('Delete failed');
        return;
      }
      location.href = '/admin';
    } catch (err) {
      console.error(err);
      alert('Network error');
    }
  });
}
