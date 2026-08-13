package backend

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/blevesearch/bleve/v2"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const bleveIndexDir = "news_crawler.bleve"

// SearchIndex là biến toàn cục chứa bleve index
var SearchIndex bleve.Index

// ArticleIndexData là dữ liệu rút gọn dùng để lập chỉ mục tìm kiếm
type ArticleIndexData struct {
	ID          string
	Title       string
	Summary     string
	Content     string
	Author      string
	PublishedAt time.Time
}

// InitBleve khởi tạo engine tìm kiếm cục bộ Bleve
func InitBleve() {
	// Kiểm tra xem file index đã tồn tại chưa
	if _, err := os.Stat(bleveIndexDir); os.IsNotExist(err) {
		// Chưa có thì tạo mới
		mapping := bleve.NewIndexMapping()
		var err error
		SearchIndex, err = bleve.New(bleveIndexDir, mapping)
		if err != nil {
			Logger.Fatal("Lỗi tạo mới Bleve index", zap.Error(err))
		}
		Logger.Info("Đã tạo mới Bleve Index tại", zap.String("dir", bleveIndexDir))
	} else {
		// Đã có thì mở lại
		SearchIndex, err = bleve.Open(bleveIndexDir)
		if err != nil {
			Logger.Fatal("Lỗi mở Bleve index", zap.Error(err))
		}
		Logger.Info("Đã kết nối tới Bleve Index hiện tại.")
	}
}

// IndexArticle thêm một bài báo vào cơ sở dữ liệu tìm kiếm của Bleve
func IndexArticle(article Article) {
	if SearchIndex == nil {
		Logger.Error("SearchIndex chưa được khởi tạo!")
		return
	}

	docID := strconv.Itoa(int(article.ID))
	
	// Lập chỉ mục với các trường quan trọng (bỏ qua ImageURL, URL vì không cần tìm mờ)
	data := ArticleIndexData{
		ID:          docID,
		Title:       article.Title,
		Summary:     article.Summary,
		Content:     article.Content,
		Author:      article.Author,
		PublishedAt: article.PublishedAt,
	}

	err := SearchIndex.Index(docID, data)
	if err != nil {
		Logger.Error("Lỗi lập chỉ mục bài báo", zap.String("Title", article.Title), zap.Error(err))
	}
}

// SearchResultItem đại diện cho một kết quả tìm kiếm trả về
type SearchResultItem struct {
	ID    uint    `json:"id"`
	Score float64 `json:"score"`
}

// SearchArticles thực hiện tìm kiếm full-text search trả về ID và Điểm số (Score)
func SearchArticles(queryText string, limit int) ([]SearchResultItem, error) {
	if SearchIndex == nil {
		return nil, fmt.Errorf("Bleve Index chưa khởi tạo")
	}

	// Tạo một truy vấn Fuzzy (Tìm mờ - chấp nhận sai chính tả nhẹ) hoặc truy vấn chuỗi
	query := bleve.NewMatchQuery(queryText)
	query.Fuzziness = 1 // Chấp nhận sai lệch 1 ký tự

	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.Size = limit // Giới hạn số kết quả trả về

	searchResult, err := SearchIndex.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	var results []SearchResultItem
	for _, hit := range searchResult.Hits {
		// Bleve lưu ID dưới dạng chuỗi, ta chuyển lại thành int
		idInt, err := strconv.Atoi(hit.ID)
		if err == nil {
			results = append(results, SearchResultItem{
				ID:    uint(idInt),
				Score: hit.Score,
			})
		}
	}

	return results, nil
}
// SearchArticlesHandler xử lý tìm kiếm full-text search thông qua Bleve
func SearchArticlesHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		queryParam := r.URL.Query().Get("q")
		if queryParam == "" {
			http.Error(w, `{"error": "Vui lòng truyền tham số q (từ khóa tìm kiếm)"}`, http.StatusBadRequest)
			return
		}

		// 1. Tìm kiếm ID từ Bleve
		results, err := SearchArticles(queryParam, 50)
		if err != nil {
			Logger.Error("Lỗi tìm kiếm qua Bleve", zap.Error(err))
			http.Error(w, `{"error": "Lỗi truy xuất công cụ tìm kiếm"}`, http.StatusInternalServerError)
			return
		}

		if len(results) == 0 {
			w.Write([]byte(`[]`)) // Trả về mảng rỗng nếu không có kết quả
			return
		}

		// 2. Thu thập danh sách ID
		var ids []uint
		for _, res := range results {
			ids = append(ids, res.ID)
		}

		// 3. Truy vấn PostgreSQL để lấy thông tin đầy đủ cho các ID này
		var articles []Article
		if err := db.Where("id IN ?", ids).Find(&articles).Error; err != nil {
			Logger.Error("Lỗi lấy dữ liệu từ Postgres sau khi tìm kiếm", zap.Error(err))
			http.Error(w, `{"error": "Lỗi truy xuất cơ sở dữ liệu"}`, http.StatusInternalServerError)
			return
		}

		// Lấy danh sách Nguồn để ánh xạ tên nguồn
		var sources []Source
		db.Find(&sources)
		sourceMap := make(map[uint]string)
		for _, s := range sources {
			sourceMap[s.ID] = s.Name
		}

		// Format lại kết quả trả về
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
				Category:    "News",
				PublishedAt: a.PublishedAt,
			})
		}

		// Trả kết quả JSON
		jsonBytes, err := json.Marshal(response)
		if err != nil {
			http.Error(w, `{"error": "Lỗi chuyển đổi dữ liệu"}`, http.StatusInternalServerError)
			return
		}

		w.Write(jsonBytes)
	}
}
