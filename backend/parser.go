package backend

import (
	"time"

	"github.com/mmcdole/gofeed"
)

// ParseRawRSS nhận vào sourceID và dữ liệu thô (raw data), trả về danh sách các Article
func ParseRawRSS(sourceID uint, rawData []byte) ([]Article, error) {
	fp := gofeed.NewParser()
	
	// Parse trực tiếp từ chuỗi dữ liệu thô thay vì tự gọi mạng
	feed, err := fp.ParseString(string(rawData))
	if err != nil {
		return nil, err
	}

	var articles []Article

	for _, item := range feed.Items {
		pubDate := time.Now()
		if item.PublishedParsed != nil {
			pubDate = *item.PublishedParsed
		}

		// Fallback để đảm bảo trường content không bị null
		content := item.Content
		if content == "" {
			content = item.Description
		}
		if content == "" {
			content = "Nội dung đang cập nhật..." // Yêu cầu của db là NOT NULL
		}

		var author string
		if len(item.Authors) > 0 {
			author = item.Authors[0].Name
		}

		var imageURL string
		if item.Image != nil {
			imageURL = item.Image.URL
		}

		article := Article{
			SourceID:    sourceID,
			Title:       item.Title,
			URL:         item.Link,
			Summary:     item.Description,
			Content:     content,
			ImageURL:    imageURL,
			Author:      author,
			PublishedAt: pubDate,
			CreatedAt:   time.Now(),
		}

		articles = append(articles, article)
	}

	return articles, nil
}
