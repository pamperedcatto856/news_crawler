package backend

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ArticleProcessor struct chứa các dependencies cần thiết cho Worker (như kết nối CSDL)
type ArticleProcessor struct {
	postgresDB   *gorm.DB
	clickhouseDB *gorm.DB
}

// NewArticleProcessor khởi tạo bộ xử lý
func NewArticleProcessor(pgDB *gorm.DB, chDB *gorm.DB) *ArticleProcessor {
	return &ArticleProcessor{
		postgresDB:   pgDB,
		clickhouseDB: chDB,
	}
}

// ProcessTask xử lý logic lưu bài viết
func (processor *ArticleProcessor) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload ArticlePayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry) // SkipRetry vì lỗi decode không thể sửa bằng cách thử lại
	}

	article := payload.Article

	// 1. Lưu vào Postgres (Bắt buộc phải lưu chống trùng lặp)
	result := processor.postgresDB.Clauses(clause.OnConflict{DoNothing: true}).Create(&article)
	if result.Error != nil {
		return fmt.Errorf("không thể lưu bài báo vào Postgres: %v", result.Error)
	}

	// Nếu bài báo đã tồn tại (result.RowsAffected == 0), ta bỏ qua, không cần đẩy sang ClickHouse
	if result.RowsAffected == 0 {
		return nil
	}

	Logger.Info("Đã lưu bài báo vào Postgres thành công", zap.String("Title", article.Title))

	// 2. Lập chỉ mục bài báo cho tính năng tìm kiếm mờ bằng Bleve
	IndexArticle(article)

	// 3. Đồng bộ sang ClickHouse
	if processor.clickhouseDB != nil {
		SyncArticlesToClickHouse(processor.clickhouseDB, []Article{article})
	}

	return nil
}

// StartWorkerServer thiết lập và chạy Asynq Server
func StartWorkerServer(redisAddr string, pgDB *gorm.DB, chDB *gorm.DB) {
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{
			Concurrency: 10,
			Queues: map[string]int{
				"default": 10,
			},
		},
	)

	mux := asynq.NewServeMux()
	processor := NewArticleProcessor(pgDB, chDB)
	mux.HandleFunc(TypeProcessArticle, processor.ProcessTask)

	Logger.Info("Bắt đầu chạy hệ thống Worker Asynq...")
	if err := srv.Run(mux); err != nil {
		Logger.Fatal("Lỗi khởi động Asynq Worker Server", zap.Error(err))
	}
}
