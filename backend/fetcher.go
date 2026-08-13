package main

import (
	"fmt"
	"log"
	"time"

	"github.com/mmcdole/gofeed"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Article định nghĩa cấu trúc bảng dữ liệu dưới dạng đối tượng Go (Model)
type Article struct {
	ID          uint   `gorm:"primaryKey"`
	Title       string `gorm:"not null"`
	URL         string `gorm:"uniqueIndex;not null"`
	PublishedAt time.Time
	CreatedAt   time.Time
}

func main() {
	// Bước 1: Kết nối cơ sở dữ liệu SQLite thông qua GORM
	db, err := gorm.Open(sqlite.Open("news.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("Không thể kết nối cơ sở dữ liệu: %v", err)
	}

	// Bước 2: Tự động tạo bảng 'articles' dựa trên cấu trúc struct Article
	err = db.AutoMigrate(&Article{})
	if err != nil {
		log.Fatalf("Không thể tự động tạo bảng dữ liệu: %v", err)
	}
	fmt.Println("Đã khởi tạo cơ sở dữ liệu thành công với GORM.")

	// Bước 3: Tải dữ liệu RSS từ VnExpress
	fp := gofeed.NewParser()
	rssURL := "https://vnexpress.net/rss/tin-moi-nhat.rss"

	feed, err := fp.ParseURL(rssURL)
	if err != nil {
		log.Fatalf("Lỗi khi tải dữ liệu từ RSS: %v", err)
	}

	fmt.Printf("Bắt đầu xử lý nguồn: %s (Tổng số bài viết trên feed: %d)\n", feed.Title, len(feed.Items))

	countNewArticles := 0

	// Bước 4: Duyệt qua từng bài viết và lưu vào CSDL
	for _, item := range feed.Items {
		pubDate := time.Now()
		if item.PublishedParsed != nil {
			pubDate = *item.PublishedParsed
		}

		article := Article{
			Title:       item.Title,
			URL:         item.Link,
			PublishedAt: pubDate,
			CreatedAt:   time.Now(),
		}

		// Sử dụng mệnh đề OnConflict để tự động bỏ qua nếu URL đã tồn tại (chống trùng lặp)
		result := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&article)
		if result.Error != nil {
			log.Printf("Lỗi ghi bài viết '%s': %v", item.Title, result.Error)
			continue
		}

		// Kiểm tra xem có bản ghi mới nào thực sự được thêm vào hay không
		if result.RowsAffected > 0 {
			countNewArticles++
		}
	}

	fmt.Printf("Quá trình fetch hoàn tất. Đã lưu thành công %d bài viết mới vào cơ sở dữ liệu.\n", countNewArticles)
}
