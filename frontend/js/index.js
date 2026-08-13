function switchView(viewId, navElement) {
    // Ẩn tất cả section
    document.querySelectorAll('.view-section').forEach(section => {
        section.style.display = 'none';
    });
    // Hiển thị section được chọn
    document.getElementById(viewId).style.display = 'block';

    // Cập nhật trạng thái Active ở menu (chỉ khi click menu)
    if (navElement && navElement.classList.contains('nav-item')) {
        document.querySelectorAll('.nav-item').forEach(item => {
            item.classList.remove('active');
        });
        navElement.classList.add('active');
    }
}

async function openArticle(id) {
	// Ẩn bảng tin, hiện bài đọc
	document.querySelectorAll('.view-section').forEach(section => {
		section.style.display = 'none';
	});
	document.getElementById('view-article').style.display = 'block';

	// Reset content và hiện trạng thái loading
	document.getElementById('detail-title').innerText = 'Đang tải...';
	document.getElementById('detail-content').innerHTML = '<div style="text-align: center; padding: 40px;"><i class="fa-solid fa-spinner fa-spin fa-2x"></i></div>';
	document.getElementById('article-summary-box').style.display = 'none';

	try {
		const response = await fetch('/api/articles/' + id);
		if (!response.ok) throw new Error("Lỗi tải chi tiết");
		const article = await response.json();

		const badgeClass = getBadgeClass(article.source_name);
		const timeStr = timeAgo(article.published_at);

		// Render Meta
		const badge = document.getElementById('detail-source-badge');
		badge.className = 'source-badge ' + badgeClass;
		badge.innerText = article.source_name;
		document.getElementById('detail-time-stamp').innerText = 'Cập nhật: ' + timeStr;

		// Render Title
		document.getElementById('detail-title').innerText = article.title;

		// Render Summary
		if (article.summary && article.summary !== article.content && article.summary !== "Không thể tạo tóm tắt.") {
			document.getElementById('article-summary-box').style.display = 'block';
			document.getElementById('detail-summary-content').innerText = stripHtml(article.summary);
		}

		// Render Content
		document.getElementById('detail-content').innerHTML = article.content || '<p>Nội dung đang cập nhật...</p>';

		// Render Link
		document.getElementById('detail-source-link').href = article.url;

	} catch (error) {
		console.error("Lỗi:", error);
		document.getElementById('detail-content').innerHTML = '<div style="color: red; text-align: center;">Có lỗi xảy ra khi tải nội dung bài báo. Vui lòng thử lại.</div>';
	}
}

// Tiện ích: Loại bỏ các thẻ HTML để lấy văn bản thuần túy
function stripHtml(html) {
	if (!html) return "";
	const tmp = document.createElement("DIV");
	tmp.innerHTML = html;
	return tmp.textContent || tmp.innerText || "";
}

// Format date "10 phút trước", "2 giờ trước", etc.
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
	const lower = sourceName.toLowerCase();
	if (lower.includes('vnexpress')) return 'vnexpress';
	if (lower.includes('tuoitre')) return 'tuoitre';
	if (lower.includes('dantri')) return 'dantri';
	return 'default';
}

// Load Articles từ Dispatcher API
async function loadArticles() {
	const grid = document.querySelector('.news-grid');
	if (!grid) return;
	
	grid.innerHTML = '<div style="grid-column: 1 / -1; text-align: center; padding: 40px;"><i class="fa-solid fa-spinner fa-spin fa-2x"></i><p style="margin-top: 10px;">Đang tải tin tức...</p></div>';

	try {
		const response = await fetch('/api/articles');
		if (!response.ok) throw new Error("Lỗi tải tin");
		const data = await response.json();
		
		grid.innerHTML = '';
		
		if (!data || data.length === 0) {
			grid.innerHTML = '<div style="grid-column: 1 / -1; text-align: center; padding: 40px;">Không có bài báo nào.</div>';
			return;
		}

		data.forEach(article => {
			const badgeClass = getBadgeClass(article.source_name);
			const timeStr = timeAgo(article.published_at);
			
			const card = document.createElement('div');
			card.className = 'news-card-horizontal';
			card.onclick = () => openArticle(article.id);
			
			// Xử lý ảnh (nếu không có ảnh thì dùng ảnh placeholder)
			const thumb = article.image_url ? article.image_url : 'https://placehold.co/300x200/1e2128/94a3b8?text=No+Image';

			// Loại bỏ HTML rác (như img tag từ RSS) khỏi summary để không phá vỡ UI card
			const cleanSummary = stripHtml(article.summary);

			card.innerHTML = `
				<div class="news-thumb">
					<img src="${thumb}" alt="thumbnail">
				</div>
				<div class="news-content">
					<h3 class="news-title">${article.title}</h3>
					<p class="news-summary">${cleanSummary}</p>
					<div class="news-meta-bottom">
						<span class="source-text ${badgeClass}">${article.source_name}</span>
						<span class="dot-separator">•</span>
						<span class="time"><i class="fa-regular fa-clock"></i> ${timeStr}</span>
					</div>
				</div>
			`;
			grid.appendChild(card);
		});
	} catch (error) {
		console.error("Lỗi:", error);
		grid.innerHTML = '<div style="grid-column: 1 / -1; text-align: center; color: red; padding: 40px;">Có lỗi xảy ra khi lấy tin. Vui lòng thử lại.</div>';
	}
}

// Chạy load dữ liệu khi DOM sẵn sàng
document.addEventListener('DOMContentLoaded', () => {
	loadArticles();
});