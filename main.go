package main

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/pamperedcatto856/news_crawler/backend"
)

func main() {
	// 0. Khởi tạo Zap Logger và Bleve Index
	backend.InitLogger()
	backend.InitBleve()
	
	// Đảm bảo flush log trước khi thoát
	defer backend.Logger.Sync()
	if backend.SearchIndex != nil {
		defer backend.SearchIndex.Close()
	}

	// 1. Tải cấu hình biến môi trường
	err := godotenv.Load()
	if err != nil {
		backend.Logger.Warn("Không tìm thấy file .env, sẽ sử dụng biến môi trường mặc định.")
	}

	// 2. Kết nối Database gốc (Postgres)
	dsn := os.Getenv("DB_DSN")
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		backend.Logger.Fatal("Lỗi kết nối DB", zap.Error(err))
	}

	// 3. Khởi tạo kết nối Cache (Redis) cho Web API
	rdb := backend.InitRedis()

	// 3.5. Kích hoạt Asynq Worker Server (Tiến trình ngầm xử lý lưu DB)
	redisAddr := os.Getenv("REDIS_DSN")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	redisAddr = strings.TrimPrefix(redisAddr, "redis://")
	
	chDB := backend.InitClickHouse() // Vẫn cần khởi tạo ở đây để truyền vào Worker
	go backend.StartWorkerServer(redisAddr, db, chDB)

	// 4. Kích hoạt Background Crawler Worker (Chạy ngầm lấy dữ liệu từ mạng)
	// Crawler sẽ cào dữ liệu và ném vào Redis Queue thay vì tự lưu DB
	go backend.StartCrawler()

	// 5. Cấu hình Dispatcher (Web API & Static Files)
	mux := http.NewServeMux()

	// API Endpoints
	mux.HandleFunc("/api/articles", backend.GetArticlesHandler(db, rdb))
	mux.HandleFunc("/api/articles/", backend.GetArticleDetailHandler(db, rdb))
	mux.HandleFunc("/api/search", backend.SearchArticlesHandler(db))

	// Serve Frontend Static Files
	fileServer := http.FileServer(http.Dir("./frontend"))
	mux.Handle("/", fileServer)

	// Cấu hình Server với Timeouts nghiêm ngặt để chống treo kết nối
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,  // Thời gian tối đa để đọc request
		WriteTimeout: 10 * time.Second, // Thời gian tối đa để phản hồi
		IdleTimeout:  120 * time.Second,
	}

	backend.Logger.Info("🚀 Web Server đang chạy tại: http://localhost:8080")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		backend.Logger.Fatal("Lỗi khởi động Server", zap.Error(err))
	}
}
