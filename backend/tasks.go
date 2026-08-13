package backend

import (
	"encoding/json"

	"github.com/hibiken/asynq"
)

// Khai báo các loại Task
const (
	TypeProcessArticle = "article:process"
)

// ArticlePayload là dữ liệu sẽ được đóng gói thành JSON và đẩy vào Redis
type ArticlePayload struct {
	Article Article `json:"article"`
}

// NewProcessArticleTask tạo một task mới chứa bài báo vừa lấy được
func NewProcessArticleTask(article Article) (*asynq.Task, error) {
	payload := ArticlePayload{Article: article}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	// Trả về task với loại là TypeProcessArticle và nội dung payload
	return asynq.NewTask(TypeProcessArticle, payloadBytes), nil
}
