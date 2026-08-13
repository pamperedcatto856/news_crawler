package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/pamperedcatto856/news_crawler/backend"
)

func main() {
	// 1. Load cấu hình môi trường
	if err := godotenv.Load(".env"); err != nil {
		log.Println("Cảnh báo: Không thể tải file .env")
	}

	// 2. Khởi tạo kết nối Storage (Postgres)
	dsn := os.Getenv("DB_DSN")
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Lỗi kết nối DB: %v", err)
	}

	// 3. Khởi tạo kết nối Cache (Redis)
	rdb := backend.InitRedis()

	// 4. Kích hoạt Background Crawler Worker (Chạy ngầm)
	// Crawler sẽ tự mở connection pool riêng bên trong nó để không ảnh hưởng luồng web
	go backend.StartCrawler()

	// 5. Cấu hình Dispatcher (Web API & Static Files)
	mux := http.NewServeMux()

	// API Endpoints
	mux.HandleFunc("/api/articles", backend.GetArticlesHandler(db, rdb))
	mux.HandleFunc("/api/articles/", backend.GetArticleDetailHandler(db, rdb))

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

	fmt.Println("🚀 Web Server đang chạy tại: http://localhost:8080")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Lỗi khởi động Server: %v", err)
	}
}
