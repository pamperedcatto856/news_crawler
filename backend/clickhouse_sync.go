package backend

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/clickhouse"
	"gorm.io/gorm"
)

// InitClickHouse kết nối tới cụm ClickHouse thông qua chproxy
// Cấu hình DSN mẫu cho chproxy: clickhouse://default:password@localhost:8123/default?dial_timeout=10s&read_timeout=20s
func InitClickHouse() *gorm.DB {
	dsn := os.Getenv("CLICKHOUSE_DSN")
	if dsn == "" {
		log.Println("Bỏ qua kết nối ClickHouse do chưa cấu hình CLICKHOUSE_DSN.")
		return nil
	}

	db, err := gorm.Open(clickhouse.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Printf("Lỗi kết nối ClickHouse (thông qua chproxy): %v\n", err)
		return nil
	}

	// Tự động tạo bảng (chú ý: ClickHouse Engine mặc định của GORM thường là MergeTree)
	// Đối với môi trường Production, nên tạo bảng thủ công với cấu trúc Engine và Partition tối ưu.
	err = db.AutoMigrate(&Article{})
	if err != nil {
		log.Printf("Cảnh báo khi AutoMigrate ClickHouse: %v\n", err)
	} else {
		fmt.Println("Đã khởi tạo bảng Article trên ClickHouse thành công.")
	}

	return db
}

// SyncArticlesToClickHouse là một tác vụ chạy nền (background job) để đồng bộ dữ liệu
// từ PostgreSQL sang ClickHouse nhằm giảm tải cho DB chính.
func SyncArticlesToClickHouse(chDB *gorm.DB, articles []Article) {
	if chDB == nil || len(articles) == 0 {
		return
	}

	// ClickHouse tối ưu cho việc chèn hàng loạt (Bulk Insert)
	// Tránh việc chèn từng dòng một.
	result := chDB.CreateInBatches(articles, 100)
	if result.Error != nil {
		log.Printf("Lỗi đồng bộ dữ liệu sang ClickHouse: %v\n", result.Error)
	} else {
		fmt.Printf("Đã đồng bộ thành công %d bài viết sang kho dữ liệu phân tích ClickHouse.\n", result.RowsAffected)
	}
}
