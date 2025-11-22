// posts_infinite.js - Infinite scroll for posts grid

(function() {
	'use strict';

	let currentPage = 1;
	let loading = false;
	let hasMore = true;

	const grid = document.getElementById('posts-grid');
	const sentinel = document.getElementById('infinite-sentinel');
	const loadingIndicator = document.getElementById('loading-indicator');
	const endMessage = document.getElementById('end-message');
	const announcer = document.getElementById('posts-announcer');

	if (!grid || !sentinel) {
		console.warn('Posts grid or sentinel not found');
		return;
	}

	// Create an intersection observer for the sentinel
	const observer = new IntersectionObserver((entries) => {
		entries.forEach(entry => {
			if (entry.isIntersecting && !loading && hasMore) {
				loadNextPage();
			}
		});
	}, {
		rootMargin: '100px' // Start loading a bit before sentinel is visible
	});

	observer.observe(sentinel);

	async function loadNextPage() {
		if (loading || !hasMore) return;

		loading = true;
		loadingIndicator.style.display = 'block';

		try {
			const response = await fetch(`/api/posts?page=${currentPage + 1}`);
			
			if (!response.ok) {
				throw new Error(`HTTP error! status: ${response.status}`);
			}

			const data = await response.json();

			if (data.posts && data.posts.length > 0) {
				appendPosts(data.posts);
				currentPage = data.page;
				hasMore = data.hasMore;

				// Announce new posts for screen readers
				if (announcer) {
					announcer.textContent = `Loaded ${data.posts.length} more posts`;
				}

				if (!hasMore) {
					observer.unobserve(sentinel);
					sentinel.style.display = 'none';
					endMessage.style.display = 'block';
				}
			} else {
				hasMore = false;
				observer.unobserve(sentinel);
				sentinel.style.display = 'none';
				endMessage.style.display = 'block';
			}
		} catch (error) {
			console.error('Error loading posts:', error);
			hasMore = false;
			loadingIndicator.style.display = 'none';
			showRetryButton();
		} finally {
			loading = false;
			loadingIndicator.style.display = 'none';
		}
	}

	function appendPosts(posts) {
		posts.forEach(post => {
			const card = createPostCard(post);
			grid.appendChild(card);
		});
	}

	function createPostCard(post) {
		const article = document.createElement('article');
		article.className = 'postcard';
		// Mark as grid context in case other scripts need to detect
		article.dataset.context = 'grid';
		article.dataset.slug = post.slug;

		const link = document.createElement('a');
		link.href = `/posts/${post.slug}`;
		link.className = 'postcard-link';

		const content = document.createElement('div');
		content.className = 'postcard-content';

		const title = document.createElement('h2');
		title.className = 'postcard-title';
		title.textContent = post.title;

		const description = document.createElement('p');
		description.className = 'postcard-description';
		description.textContent = post.lead || '';

		const meta = document.createElement('div');
		meta.className = 'postcard-meta';

		const time = document.createElement('time');
		const date = new Date(post.updated);
		time.setAttribute('datetime', post.updated);
		time.textContent = date.toISOString().split('T')[0];

		meta.appendChild(time);

		content.appendChild(title);
		content.appendChild(description);
		content.appendChild(meta);
		link.appendChild(content);
		article.appendChild(link);

		return article;
	}

	function showRetryButton() {
		const retryDiv = document.createElement('div');
		retryDiv.className = 'retry-container';
		retryDiv.style.textAlign = 'center';
		retryDiv.style.padding = '24px';

		const retryBtn = document.createElement('button');
		retryBtn.textContent = 'Retry loading posts';
		retryBtn.style.padding = '8px 16px';
		retryBtn.style.background = 'var(--color-accent)';
		retryBtn.style.color = 'var(--color-text-invert)';
		retryBtn.style.border = 'none';
		retryBtn.style.borderRadius = '4px';
		retryBtn.style.cursor = 'pointer';
		retryBtn.style.fontWeight = '600';

		retryBtn.addEventListener('click', () => {
			retryDiv.remove();
			hasMore = true;
			loadNextPage();
		});

		retryDiv.appendChild(retryBtn);
		sentinel.parentNode.insertBefore(retryDiv, sentinel);
	}
})();
