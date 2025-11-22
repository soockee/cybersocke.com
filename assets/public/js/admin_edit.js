const form = document.getElementById('edit-form');
if (form) {
  const csrf = document.getElementById('csrf-token').value;
  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    const slug = form.getAttribute('data-slug');
    const content = document.getElementById('post-content').value;
    try {
      const resp = await fetch(`/admin/posts/${slug}/edit`, {
        method: 'POST',
        headers: { 'X-CSRF-Token': csrf },
        body: content
      });
      if (resp.status === 303 || resp.ok) {
        location.href = '/admin';
        return;
      }
      alert('Update failed');
    } catch (err) {
      console.error(err);
      alert('Network error');
    }
  });
}
