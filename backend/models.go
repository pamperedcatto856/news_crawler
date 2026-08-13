package backend

import "time"

type Source struct {
	ID        uint      `gorm:"primaryKey"`
	Name      string    `gorm:"not null;unique"`
	BaseURL   string    `gorm:"not null"`
	RSSURL    string    `gorm:"column:rss_url"`
	IsActive  bool      `gorm:"default:true"`
	CreatedAt time.Time
}

type Category struct {
	ID        uint      `gorm:"primaryKey"`
	SourceID  uint      `gorm:"not null"`
	Name      string    `gorm:"not null"`
	URL       string    
	CreatedAt time.Time
}

type Article struct {
	ID          uint      `gorm:"primaryKey"`
	SourceID    uint      `gorm:"not null"`
	CategoryID  *uint     
	Title       string    `gorm:"not null"`
	URL         string    `gorm:"uniqueIndex;not null"`
	Summary     string    
	Content     string    `gorm:"not null"`
	ImageURL    string    
	Author      string    
	PublishedAt time.Time `gorm:"not null"`
	CreatedAt   time.Time 
}
