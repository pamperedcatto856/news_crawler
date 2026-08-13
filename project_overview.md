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

Logic hoạt động của Fetcher

Fetcher đóng vai trò là "người đi lấy hàng", chỉ chịu trách nhiệm giao tiếp với môi trường bên ngoài mạng Internet để mang dữ liệu thô về mà không cần quan tâm bên trong dữ liệu đó chứa nội dung gì.

- Gửi yêu cầu mạng: Thực hiện các yêu cầu HTTP/HTTPS tới các đường dẫn RSS hoặc trang web mục tiêu.

- Quản lý cấu hình kết nối: Thiết lập thời gian chờ tối đa (Timeout) để tránh việc chương trình bị treo vô thời hạn khi mạng gặp sự cố.

- Cấu hình định danh (Headers): Đính kèm các thông tin như User-Agent để giả lập trình duyệt, giúp tránh việc bị các trang báo chặn truy cập (Anti-bot).

- Xử lý lỗi mạng: Phát hiện và ghi nhận các mã lỗi phản hồi từ máy chủ (ví dụ: lỗi 404 không tìm thấy trang, lỗi 500 lỗi máy chủ).

- Cơ chế thử lại (Retry): Tự động thực hiện lại yêu cầu kết nối một vài lần nếu gặp sự cố mất mạng tạm thời.

- Truyền dữ liệu thô: Trả về kết quả dưới dạng luồng dữ liệu thô (Raw Byte Stream hoặc chuỗi XML/HTML thô) để chuyển tiếp thẳng sang bộ phận xử lý tiếp theo.

Fetcher chịu trách nhiệm giao tiếp với Internet, vì vậy các giới hạn tập trung vào việc kiểm soát thời gian và tần suất:

- Giới hạn thời gian chờ (Request Timeout):

Bắt buộc đặt thời gian tối đa cho một yêu cầu mạng (ví dụ: từ 10 đến 15 giây). Nếu trang báo phản hồi chậm hoặc mất kết nối, hệ thống sẽ tự động hủy yêu cầu thay vì treo vô thời hạn, gây nghẽn tiến trình.

- Giới hạn tần suất gọi (Rate Limiting / Throttling):

Không gửi hàng loạt yêu cầu liên tục trong tích tắc. Phải thiết lập khoảng thời gian nghỉ cố định giữa các lần lấy tin (ví dụ: nghỉ 2 đến 3 giây giữa mỗi lần gọi RSS) để tránh làm sập máy chủ phía trang báo và không bị khóa địa chỉ IP (Anti-bot).

- Giới hạn dung lượng tải về (Payload Size Limit):

Thiết lập ngưỡng dung lượng tối đa cho phép của tệp phản hồi (ví dụ: tối đa 5MB cho một tệp RSS/XML). Điều này ngăn chặn việc nhận phải dữ liệu rác hoặc tệp quá lớn làm tràn bộ nhớ RAM.


Storage (DB)'s Limitations:

Để cơ sở dữ liệu không bị chậm sau thời gian dài hoạt động, cần áp dụng các nguyên tắc sau:

Giới hạn chống trùng lặp nghiêm ngặt (Unique Constraint):

Luôn dùng thuộc tính duy nhất trên đường dẫn (URL) để cơ sở dữ liệu tự động từ chối lưu các bản sao, giảm thiểu dung lượng thừa.

Chính sách giới hạn thời gian lưu trữ (Data Retention Policy):

Thiết lập quy tắc tự động xóa hoặc chuyển lưu trữ phụ (Archiving) đối với các bài viết cũ quá hạn (ví dụ: chỉ giữ lại tin tức trong vòng 30 hoặc 60 ngày gần nhất), vì tin tức cũ thường có ít giá trị đọc lại.