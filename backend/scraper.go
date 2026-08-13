package backend

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// ScrapeFullArticle tải và trích xuất toàn bộ nội dung của bài báo từ URL
func ScrapeFullArticle(articleURL string) (string, error) {
	client := GetHTTPClient()
	req, err := http.NewRequest("GET", articleURL, nil)
	if err != nil {
		return "", fmt.Errorf("lỗi tạo request: %v", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("lỗi tải trang: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("trang trả về mã lỗi: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("lỗi parse HTML: %v", err)
	}

	// Các selector phổ biến để tìm vùng nội dung chính
	selectors := []string{
		"article.fck_detail", // VnExpress (kiểu cũ)
		"article .fck_detail", // VnExpress (kiểu mới)
		".detail-content",     // Tuổi Trẻ, Dân Trí
		"article",             // Cấu trúc chuẩn HTML5
		".entry-content",      // Các trang dùng WordPress
	}

	var htmlContent string
	for _, sel := range selectors {
		selection := doc.Find(sel).First()
		if selection.Length() > 0 {
			// Duyệt qua các thẻ p và ảnh để giữ đúng thứ tự nội dung
			selection.Find("p, figure, div.fig-picture, img").Each(func(i int, s *goquery.Selection) {
				if s.Is("p") {
					// Bỏ qua nếu p nằm trong figure/div.fig-picture để tránh trùng lặp caption
					if s.ParentsFiltered("figure, div.fig-picture").Length() > 0 {
						return
					}
					text := strings.TrimSpace(s.Text())
					if len(text) > 30 {
						htmlContent += fmt.Sprintf("<p>%s</p>\n", text)
					}
				} else if s.Is("figure") || s.Is("div.fig-picture") {
					img := s.Find("img").First()
					if src, exists := img.Attr("src"); exists && src != "" {
						// Xử lý lazy loading của ảnh báo
						if dataSrc, has := img.Attr("data-src"); has && dataSrc != "" {
							src = dataSrc
						}
						caption := strings.TrimSpace(s.Text())
						htmlContent += fmt.Sprintf("<figure style=\"margin: 20px 0; text-align: center;\"><img src=\"%s\" style=\"max-width: 100%%; border-radius: 8px;\"><figcaption style=\"font-style: italic; color: #888; margin-top: 8px; font-size: 0.9rem;\">%s</figcaption></figure>\n", src, caption)
					}
				} else if s.Is("img") {
					// Ảnh đứng độc lập không thuộc figure
					if s.ParentsFiltered("figure, div.fig-picture").Length() == 0 {
						if src, exists := s.Attr("src"); exists && src != "" {
							if dataSrc, has := s.Attr("data-src"); has && dataSrc != "" {
								src = dataSrc
							}
							htmlContent += fmt.Sprintf("<div style=\"text-align: center; margin: 20px 0;\"><img src=\"%s\" style=\"max-width: 100%%; border-radius: 8px;\"></div>\n", src)
						}
					}
				}
			})
			if htmlContent != "" {
				return htmlContent, nil
			}
		}
	}

	// Fallback: nếu không khớp selector nào, tìm tất cả p và img có trong tài liệu
	doc.Find("p, img").Each(func(i int, s *goquery.Selection) {
		if s.Is("p") {
			text := strings.TrimSpace(s.Text())
			if len(text) > 50 {
				htmlContent += fmt.Sprintf("<p>%s</p>\n", text)
			}
		} else if s.Is("img") {
			if src, exists := s.Attr("src"); exists && src != "" {
				if dataSrc, has := s.Attr("data-src"); has && dataSrc != "" {
					src = dataSrc
				}
				htmlContent += fmt.Sprintf("<div style=\"text-align: center; margin: 20px 0;\"><img src=\"%s\" style=\"max-width: 100%%; border-radius: 8px;\"></div>\n", src)
			}
		}
	})

	if htmlContent != "" {
		return htmlContent, nil
	}

	return "", fmt.Errorf("không thể tìm thấy thẻ nội dung nào")
}

// GenerateSummary tạo bản tóm tắt nhanh dựa trên thuật toán trích xuất cơ bản (Extractive Summarization)
func GenerateSummary(htmlContent string) string {
	// Xóa các thẻ HTML để lấy raw text
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return "Không thể tạo tóm tắt."
	}
	
	text := strings.TrimSpace(doc.Text())
	if text == "" {
		return "Nội dung trống, không thể tóm tắt."
	}

	// Tách câu dựa trên dấu chấm
	sentences := strings.Split(text, ".")
	
	// Lấy 3-4 câu đầu tiên (thường chứa nội dung quan trọng nhất của tin tức)
	var summarySentences []string
	count := 0
	for _, s := range sentences {
		s = strings.TrimSpace(s)
		if len(s) > 20 { // Bỏ qua các câu quá ngắn
			summarySentences = append(summarySentences, s+".")
			count++
			if count >= 3 {
				break
			}
		}
	}

	if len(summarySentences) == 0 {
		return text // Trả về nội dung gốc nếu quá ngắn
	}

	return strings.Join(summarySentences, " ")
}
