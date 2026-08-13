package backend

import (
	"os"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// StartCrawler là hàm chính khởi chạy crawler ngầm
func StartCrawler() {
	// Mở connection riêng cho Crawler để lấy danh sách nguồn
	dsn := os.Getenv("DB_DSN")
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		Logger.Fatal("Crawler: Lỗi kết nối DB", zap.Error(err))
		return
	}
	
	// Khởi tạo Asynq Client kết nối tới Redis
	redisAddr := os.Getenv("REDIS_DSN")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	redisAddr = strings.TrimPrefix(redisAddr, "redis://")
	
	clientQueue := asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr})
	defer clientQueue.Close()

	// Khởi tạo Colly Collector
	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36"),
		colly.Async(true), // Cho phép cào bất đồng bộ tốc độ cao
	)

	// Đặt Rate Limit để mô phỏng người dùng, tránh bị khóa IP
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*", // Áp dụng cho mọi domain
		Parallelism: 2,   // Tối đa 2 luồng song song cùng lúc
		RandomDelay: 2 * time.Second,
	})

	for {
		var sources []Source
		// Chỉ lấy các nguồn đang active và có link RSS
		if err := db.Where("is_active = ? AND rss_url IS NOT NULL", true).Find(&sources).Error; err != nil {
			Logger.Error("Crawler: Lỗi lấy danh sách nguồn", zap.Error(err))
			time.Sleep(30 * time.Second)
			continue
		}

		for _, source := range sources {
			Logger.Info("Bắt đầu cào nguồn", zap.String("Tên", source.Name), zap.String("URL", source.RSSURL))
			
			// Cấu hình sự kiện khi nhận được phản hồi RSS từ Colly
			c.OnResponse(func(r *colly.Response) {
				articles, err := ParseRawRSS(source.ID, r.Body)
				if err != nil {
					Logger.Error("Lỗi parse RSS", zap.String("Nguồn", source.Name), zap.Error(err))
					return
				}

				countQueued := 0
				for _, article := range articles {
					task, err := NewProcessArticleTask(article)
					if err != nil {
						Logger.Error("Lỗi tạo task", zap.String("Title", article.Title), zap.Error(err))
						continue
					}

					info, err := clientQueue.Enqueue(task, asynq.Retention(24*time.Hour))
					if err != nil {
						Logger.Error("Lỗi đẩy task vào hàng đợi", zap.Error(err))
					} else {
						countQueued++
						_ = info
					}
				}

				if countQueued > 0 {
					Logger.Info("Cào thành công", zap.String("Nguồn", source.Name), zap.Int("Số bài viết xếp hàng", countQueued))
				} else {
					Logger.Info("Không có bài viết nào được xử lý", zap.String("Nguồn", source.Name))
				}
			})

			c.OnError(func(r *colly.Response, err error) {
				Logger.Error("Colly gặp lỗi khi tải trang", zap.String("URL", r.Request.URL.String()), zap.Error(err))
			})

			// Thực thi truy cập URL
			c.Visit(source.RSSURL)
		}
		
		// Colly.Wait() để đảm bảo tất cả các requests bất đồng bộ trong vòng này hoàn tất trước khi nghỉ 5 phút
		c.Wait()

		Logger.Info("Quá trình fetch hoàn tất. Nghỉ 5 phút trước vòng lặp tiếp theo...")
		time.Sleep(5 * time.Minute)
	}
}
