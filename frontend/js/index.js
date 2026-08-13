// App State
let allArticles = [];
let currentSort = 'newest';
let currentSearchQuery = '';

function switchView(viewId, navElement) {
    document.querySelectorAll('.view-section').forEach(section => {
        section.style.display = 'none';
    });
    document.getElementById(viewId).style.display = 'block';

    if (navElement && navElement.classList.contains('nav-item')) {
        document.querySelectorAll('.nav-item').forEach(item => {
            item.classList.remove('active');
        });
        navElement.classList.add('active');
    }
}

async function openArticle(id) {
	document.querySelectorAll('.view-section').forEach(section => {
		section.style.display = 'none';
	});
	document.getElementById('view-article').style.display = 'block';

	document.getElementById('detail-title').innerText = 'Đang tải...';
	document.getElementById('detail-content').innerHTML = '<div style="text-align: center; padding: 40px;"><i class="fa-solid fa-spinner fa-spin fa-2x"></i></div>';
	document.getElementById('article-summary-box').style.display = 'none';

	try {
		const response = await fetch('/api/articles/' + id);
		if (!response.ok) throw new Error("Lỗi tải chi tiết");
		const article = await response.json();

		const badgeClass = getBadgeClass(article.source_name);
		const timeStr = timeAgo(article.published_at);

		const badge = document.getElementById('detail-source-badge');
		badge.className = 'source-badge ' + badgeClass;
		badge.innerText = article.source_name || "Unknown";
		document.getElementById('detail-time-stamp').innerText = 'Cập nhật: ' + timeStr;
		document.getElementById('detail-title').innerText = article.title;

		if (article.summary && article.summary !== article.content && article.summary !== "Không thể tạo tóm tắt.") {
			document.getElementById('article-summary-box').style.display = 'block';
			document.getElementById('detail-summary-content').innerText = stripHtml(article.summary);
		}

		document.getElementById('detail-content').innerHTML = article.content || '<p>Nội dung đang cập nhật...</p>';
		document.getElementById('detail-source-link').href = article.url;

	} catch (error) {
		console.error("Lỗi:", error);
		document.getElementById('detail-content').innerHTML = '<div style="color: red; text-align: center;">Có lỗi xảy ra khi tải nội dung bài báo. Vui lòng thử lại.</div>';
	}
}

function stripHtml(html) {
	if (!html) return "";
	const tmp = document.createElement("DIV");
	tmp.innerHTML = html;
	return tmp.textContent || tmp.innerText || "";
}

function timeAgo(dateString) {
	const date = new Date(dateString);
	const now = new Date();
	const diff = (now.getTime() - date.getTime()) / 1000;
	
	if (diff < 60) return "Vừa xong";
	if (diff < 3600) return Math.floor(diff / 60) + " phút trước";
	if (diff < 86400) return Math.floor(diff / 3600) + " giờ trước";
	return Math.floor(diff / 86400) + " ngày trước";
}

function getBadgeClass(sourceName) {
	if (!sourceName) return 'default';
	const lower = sourceName.toLowerCase();
	if (lower.includes('vnexpress')) return 'vnexpress';
	if (lower.includes('tuoitre')) return 'tuoitre';
	if (lower.includes('dantri')) return 'dantri';
	return 'default';
}

function renderArticles(articles) {
	const grid = document.querySelector('.news-grid');
	if (!grid) return;
	grid.innerHTML = '';
	
	if (!articles || articles.length === 0) {
		grid.innerHTML = '<div style="grid-column: 1 / -1; text-align: center; padding: 40px; color: var(--text-muted);">Không tìm thấy bài báo nào.</div>';
		return;
	}

	// Áp dụng Sort
	let sortedArticles = [...articles];
	if (currentSort === 'newest') {
		sortedArticles.sort((a, b) => new Date(b.published_at) - new Date(a.published_at));
	} else if (currentSort === 'oldest') {
		sortedArticles.sort((a, b) => new Date(a.published_at) - new Date(b.published_at));
	}

	sortedArticles.forEach(article => {
		const badgeClass = getBadgeClass(article.source_name);
		const timeStr = timeAgo(article.published_at);
		
		const card = document.createElement('div');
		card.className = 'news-card-horizontal';
		card.onclick = () => openArticle(article.id);
		
		const thumb = article.image_url ? article.image_url : 'https://placehold.co/300x200/1e2128/94a3b8?text=No+Image';
		const cleanSummary = stripHtml(article.summary);

		card.innerHTML = `
			<div class="news-thumb">
				<img src="${thumb}" alt="thumbnail">
			</div>
			<div class="news-content">
				<h3 class="news-title">${article.title}</h3>
				<p class="news-summary">${cleanSummary}</p>
				<div class="news-meta-bottom">
					<span class="source-text ${badgeClass}">${article.source_name || 'Unknown'}</span>
					<span class="dot-separator">•</span>
					<span class="time"><i class="fa-regular fa-clock"></i> ${timeStr}</span>
				</div>
			</div>
		`;
		grid.appendChild(card);
	});
}

async function loadArticles() {
	const grid = document.querySelector('.news-grid');
	if (!grid) return;
	
	grid.innerHTML = '<div style="grid-column: 1 / -1; text-align: center; padding: 40px;"><i class="fa-solid fa-spinner fa-spin fa-2x"></i><p style="margin-top: 10px;">Đang tải tin tức...</p></div>';

	try {
		const response = await fetch('/api/articles');
		if (!response.ok) throw new Error("Lỗi tải tin");
		const data = await response.json();
		allArticles = data || [];
		renderArticles(allArticles);
	} catch (error) {
		console.error("Lỗi:", error);
		grid.innerHTML = '<div style="grid-column: 1 / -1; text-align: center; color: #f87171; padding: 40px;">Có lỗi xảy ra khi lấy tin. Vui lòng thử lại.</div>';
	}
}

async function performSearch(query) {
	currentSearchQuery = query;
	if (!query.trim()) {
		renderArticles(allArticles);
		return;
	}

	const grid = document.querySelector('.news-grid');
	grid.innerHTML = '<div style="grid-column: 1 / -1; text-align: center; padding: 40px;"><i class="fa-solid fa-spinner fa-spin fa-2x"></i><p style="margin-top: 10px;">Đang tìm kiếm...</p></div>';

	try {
		const response = await fetch(`/api/search?q=${encodeURIComponent(query)}`);
		if (!response.ok) throw new Error("Lỗi tìm kiếm");
		const data = await response.json();
		renderArticles(data || []);
	} catch (error) {
		console.error("Lỗi:", error);
		grid.innerHTML = '<div style="grid-column: 1 / -1; text-align: center; color: #f87171; padding: 40px;">Có lỗi xảy ra khi tìm kiếm.</div>';
	}
}

// Bắt sự kiện DOM Ready
document.addEventListener('DOMContentLoaded', () => {
	loadArticles();

	// Search logic
	const searchInput = document.querySelector('.search-bar input');
	const searchIcon = document.querySelector('.search-bar i');

	const executeSearch = () => {
		const query = searchInput.value;
		performSearch(query);
	};

	if (searchInput) {
		searchInput.addEventListener('keypress', (e) => {
			if (e.key === 'Enter') {
				executeSearch();
			}
		});
	}

	if (searchIcon) {
		searchIcon.style.cursor = 'pointer';
		searchIcon.addEventListener('click', executeSearch);
	}

	// Sort logic
	const sortSelect = document.getElementById('sort-by');
	if (sortSelect) {
		sortSelect.addEventListener('change', (e) => {
			currentSort = e.target.value;
			if (currentSearchQuery.trim()) {
				// Nếu đang search, việc sort lại mảng đã render sẽ tự động chạy trong renderArticles
				// nhưng ở đây ta gọi lại search thì hơi thừa, nên chỉ render lại (tuy nhiên ta đang ko lưu kết quả search).
				// Do đó, tối ưu là gọi lại search API, hoặc lưu mảng currentArticles
				executeSearch(); 
			} else {
				renderArticles(allArticles);
			}
		});
	}
});