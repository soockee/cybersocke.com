// admin_sort.js - Client-side table sorting for admin posts table

(function() {
	'use strict';

	const table = document.getElementById('admin-posts-table');
	if (!table) return;

	const headers = table.querySelectorAll('th[data-sort-key]');
	let currentSortKey = null;
	let currentSortDir = 'asc';

	headers.forEach(header => {
		header.addEventListener('click', () => {
			const sortKey = header.dataset.sortKey;
			
			// Toggle direction if clicking same header
			if (sortKey === currentSortKey) {
				currentSortDir = currentSortDir === 'asc' ? 'desc' : 'asc';
			} else {
				currentSortKey = sortKey;
				currentSortDir = 'asc';
			}

			// Update aria-sort attributes
			headers.forEach(h => h.setAttribute('aria-sort', 'none'));
			header.setAttribute('aria-sort', currentSortDir === 'asc' ? 'ascending' : 'descending');

			sortTable(sortKey, currentSortDir);
		});
	});

	function sortTable(key, direction) {
		const tbody = table.querySelector('tbody');
		const rows = Array.from(tbody.querySelectorAll('tr'));

		rows.sort((rowA, rowB) => {
			let a, b;

			switch (key) {
				case 'title':
					a = rowA.querySelector('td:nth-child(1) a').textContent.toLowerCase();
					b = rowB.querySelector('td:nth-child(1) a').textContent.toLowerCase();
					return direction === 'asc' ? a.localeCompare(b) : b.localeCompare(a);

				case 'slug':
					a = rowA.dataset.slug.toLowerCase();
					b = rowB.dataset.slug.toLowerCase();
					return direction === 'asc' ? a.localeCompare(b) : b.localeCompare(a);

				case 'updated':
					a = new Date(rowA.dataset.updated);
					b = new Date(rowB.dataset.updated);
					return direction === 'asc' ? a - b : b - a;

				case 'published':
					a = rowA.dataset.published === 'true' ? 1 : 0;
					b = rowB.dataset.published === 'true' ? 1 : 0;
					return direction === 'asc' ? a - b : b - a;

				case 'tags':
					a = parseInt(rowA.dataset.tagsCount, 10) || 0;
					b = parseInt(rowB.dataset.tagsCount, 10) || 0;
					return direction === 'asc' ? a - b : b - a;

				case 'leadLength':
					a = parseInt(rowA.dataset.leadLength, 10) || 0;
					b = parseInt(rowB.dataset.leadLength, 10) || 0;
					return direction === 'asc' ? a - b : b - a;

				default:
					return 0;
			}
		});

		// Re-append rows in sorted order
		rows.forEach(row => tbody.appendChild(row));
	}
})();
