// Dùng để truyền nối với cache và user interface
package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// InitRedis khởi tạo kết nối đến Redis Cache Server
func InitRedis() *redis.Client {
	redisDSN := os.Getenv("REDIS_DSN")
	if redisDSN == "" {
		log.Println("Cảnh báo: Chưa cấu hình REDIS_DSN, tính năng Cache sẽ bị vô hiệu hóa.")
		return nil
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisDSN,
		Password: "", // Không dùng pass trong môi trường dev
		DB:       0,  // Dùng DB mặc định
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("Lỗi kết nối Redis: %v. Cache bị vô hiệu hóa.", err)
		return nil
	}

	log.Println("Đã kết nối Redis Cache thành công.")
	return rdb
}

// ArticleResponse cấu trúc trả về cho Frontend, mix thêm Tên nguồn báo
type ArticleResponse struct {
	ID          uint      `json:"id"`
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Summary     string    `json:"summary"`
	ImageURL    string    `json:"image_url"`
	SourceName  string    `json:"source_name"`
	Category    string    `json:"category"`
	PublishedAt time.Time `json:"published_at"`
}

// GetArticlesHandler trả về danh sách các bài viết mới nhất (có hỗ trợ Redis Cache)
func GetArticlesHandler(db *gorm.DB, rdb *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set headers
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		ctx := r.Context()
		cacheKey := "api:articles:newest:50"

		// 1. Cố gắng đọc từ Redis Cache trước (Cache Hit)
		if rdb != nil {
			cachedData, err := rdb.Get(ctx, cacheKey).Result()
			if err == nil {
				// Cache Hit! Trả về nguyên khối JSON string
				w.Write([]byte(cachedData))
				return
			}
		}

		// 2. Nếu không có trong Cache (Cache Miss), truy vấn Database
		var articles []Article
		if err := db.Order("published_at DESC").Limit(50).Find(&articles).Error; err != nil {
			http.Error(w, `{"error": "Lỗi truy xuất cơ sở dữ liệu"}`, http.StatusInternalServerError)
			return
		}

		// Lấy map Source ID -> Name để hiển thị cho đẹp
		var sources []Source
		db.Find(&sources)
		sourceMap := make(map[uint]string)
		for _, s := range sources {
			sourceMap[s.ID] = s.Name
		}

		// Chuyển đổi định dạng phù hợp với giao diện
		var response []ArticleResponse
		for _, a := range articles {
			sName := "Unknown"
			if name, ok := sourceMap[a.SourceID]; ok {
				sName = name
			}
			
			response = append(response, ArticleResponse{
				ID:          a.ID,
				Title:       a.Title,
				URL:         a.URL,
				Summary:     a.Summary,
				ImageURL:    a.ImageURL,
				SourceName:  sName,
				Category:    "News", // Ở đây mặc định lấy category chung, có thể mở rộng sau
				PublishedAt: a.PublishedAt,
			})
		}

		// Encode kết quả thành JSON
		jsonBytes, err := json.Marshal(response)
		if err != nil {
			http.Error(w, `{"error": "Lỗi chuyển đổi dữ liệu"}`, http.StatusInternalServerError)
			return
		}

		// 3. Lưu vào Redis Cache để dùng cho các request tiếp theo (TTL: 5 phút)
		if rdb != nil {
			// Fire & forget trong background để không block API response
			go func() {
				bgCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				rdb.Set(bgCtx, cacheKey, jsonBytes, 5*time.Minute)
			}()
		}

		// 4. Trả kết quả cho Frontend
		w.Write(jsonBytes)
	}
}

// ArticleDetailResponse cấu trúc trả về chi tiết 1 bài viết
type ArticleDetailResponse struct {
	ID          uint      `json:"id"`
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Summary     string    `json:"summary"`
	Content     string    `json:"content"`
	ImageURL    string    `json:"image_url"`
	SourceName  string    `json:"source_name"`
	Author      string    `json:"author"`
	PublishedAt time.Time `json:"published_at"`
}

// GetArticleDetailHandler trả về chi tiết 1 bài báo, thực hiện On-Demand scraping nếu nội dung chưa đầy đủ
func GetArticleDetailHandler(db *gorm.DB, rdb *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// Trích xuất ID từ URL path (vd: /api/articles/123)
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 4 {
			http.Error(w, `{"error": "ID không hợp lệ"}`, http.StatusBadRequest)
			return
		}
		
		idStr := parts[3]
		id, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil {
			http.Error(w, `{"error": "ID phải là số"}`, http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		cacheKey := fmt.Sprintf("api:article_detail:%d", id)

		// 1. Cố gắng đọc từ Redis Cache trước (Cache Hit)
		if rdb != nil {
			cachedData, err := rdb.Get(ctx, cacheKey).Result()
			if err == nil {
				w.Write([]byte(cachedData))
				return
			}
		}

		// 2. Lấy dữ liệu bài báo từ CSDL
		var article Article
		if err := db.First(&article, id).Error; err != nil {
			http.Error(w, `{"error": "Không tìm thấy bài báo"}`, http.StatusNotFound)
			return
		}

		// Kiểm tra xem bài báo đã có nội dung dài (scraped content) chưa
		// Giả định: nếu content chứa các thẻ HTML như <p> hoặc dài hơn 500 ký tự, thì đã được cào
		needsScraping := true
		if strings.Contains(article.Content, "<p>") || len(article.Content) > 500 {
			needsScraping = false
		}

		if needsScraping {
			log.Printf("Bắt đầu cào on-demand cho bài báo ID %d: %s", id, article.URL)
			scrapedHTML, err := ScrapeFullArticle(article.URL)
			if err == nil && scrapedHTML != "" {
				article.Content = scrapedHTML
				// Tạo AI summary (Dummy extractive summary) từ nội dung vừa cào
				article.Summary = GenerateSummary(scrapedHTML)
				
				// Lưu cập nhật vào CSDL để lần sau không cần cào lại
				db.Save(&article)
			} else {
				log.Printf("Lỗi cào bài báo ID %d: %v", id, err)
				// Fallback: nếu lỗi, giữ nguyên content gốc (từ RSS)
			}
		}

		// Lấy tên nguồn
		var source Source
		sourceName := "Unknown"
		if err := db.First(&source, article.SourceID).Error; err == nil {
			sourceName = source.Name
		}

		response := ArticleDetailResponse{
			ID:          article.ID,
			Title:       article.Title,
			URL:         article.URL,
			Summary:     article.Summary,
			Content:     article.Content,
			ImageURL:    article.ImageURL,
			SourceName:  sourceName,
			Author:      article.Author,
			PublishedAt: article.PublishedAt,
		}

		jsonBytes, err := json.Marshal(response)
		if err != nil {
			http.Error(w, `{"error": "Lỗi chuyển đổi dữ liệu"}`, http.StatusInternalServerError)
			return
		}

		// 3. Lưu Cache (Cache 1 tiếng vì bài cũ ít khi thay đổi nội dung)
		if rdb != nil {
			go func() {
				bgCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				rdb.Set(bgCtx, cacheKey, jsonBytes, 1*time.Hour)
			}()
		}

		w.Write(jsonBytes)
	}
}