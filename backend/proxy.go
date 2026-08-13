package backend

import (
	"net/http"
	"net/url"
	"os"
	"time"
)

// GetHTTPClient trả về một http.Client có cấu hình Proxy (nếu có) và Timeout (10 giây)
func GetHTTPClient() *http.Client {
	proxyStr := os.Getenv("PROXY_URL")

	// Khởi tạo Transport mặc định của Go
	transport := http.DefaultTransport.(*http.Transport).Clone()

	// Cài đặt Proxy nếu biến môi trường PROXY_URL được thiết lập
	if proxyStr != "" {
		proxyURL, err := url.Parse(proxyStr)
		if err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}

	return &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second, // Timeout 10 giây
	}
}
