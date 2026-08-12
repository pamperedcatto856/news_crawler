Kiến trúc tổng quan của hệ thống

Trước khi viết mã, bạn cần hiểu một con bot lấy tin tức hoạt động như thế nào qua 3 bước chính:

- Thu thập (Fetch/Scrape)
  Tải nội dung trang web hoặc đọc qua kênh cấp phát tin chuẩn.
- Xử lý (Parse/Clean)
  Lọc bỏ quảng cáo, định dạng lại văn bản (tiêu đề, nội dung, hình ảnh, nguồn).
- Lưu trữ & Phân phối (Store & Deliver)
  Lưu vào cơ sở dữ liệu và gửi đến người dùng qua Telegram, Web, hoặc App.

Các thành phần logic Backend chi tiết
Thành phần - Chức năng chính trong Backend - Thư viện/Công nghệ gợi ý (Go)

1. Fetcher & Parser
Gửi HTTP Request lấy dữ liệu từ nguồn (RSS/Web) và phân tích cú pháp XML/HTML.
net/http, [github.com/mmcdole/gofeed](cho RSS)
<!-- 1.5. Deduplication (Khử trùng lặp)
   Kiểm tra xem bài viết đã tồn tại trong cơ sở dữ liệu chưa dựa vào URL hoặc ID duy nhất để tránh lưu trùng.
   Lưu hash(url) hoặc khóa UNIQUE trong Database. -->
2. Storage (Cơ sở dữ liệu)
   Lưu trữ thông tin bài báo (Tiêu đề, Link, Nội dung tóm tắt, Thời gian đăng, Nguồn).
   PostgreSQL (dùng database/sql hoặc ORM như GORM).
3. Dispatcher (Phân phối)
Đẩy tin tức mới tới người dùng qua API.
<!-- 4. Scheduler
   Lên lịch tự động chạy tác vụ quét tin (ví dụ: cứ 30 phút quét 1 lần).
   [github.com/robfig/cron/v3] -->
