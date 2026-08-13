package backend

import (
	"log"

	"go.uber.org/zap"
)

// Logger là biến toàn cục dùng chung cho toàn bộ backend
var Logger *zap.Logger

// InitLogger khởi tạo Zap Logger
func InitLogger() {
	var err error
	// Sử dụng NewProduction cho môi trường thực tế, ghi log dưới dạng JSON siêu nhanh
	// Hoặc dùng NewDevelopment để dễ đọc trong console (nhưng ta sẽ dùng Production cho chuẩn)
	config := zap.NewProductionConfig()
	config.OutputPaths = []string{"stdout"} // Ghi thẳng ra console để dễ nhìn
	
	Logger, err = config.Build()
	if err != nil {
		log.Fatalf("Không thể khởi tạo Zap Logger: %v", err)
	}

	Logger.Info("Khởi tạo hệ thống Log Zap thành công!")
}
