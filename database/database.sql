CREATE TABLE sources (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    base_url VARCHAR(255) NOT NULL,
    rss_url VARCHAR(500), 
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE categories (
    id SERIAL PRIMARY KEY,
    source_id INT NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    url VARCHAR(500), -- Link của danh mục trên trang gốc (phục vụ cho crawler)
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (source_id, name) -- Đảm bảo 1 nguồn báo không có 2 danh mục trùng tên
);

CREATE TABLE articles (
    id SERIAL PRIMARY KEY,
    source_id INT NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    category_id INT REFERENCES categories(id) ON DELETE CASCADE,
    title VARCHAR(500) NOT NULL,
    url TEXT NOT NULL UNIQUE, 
    summary TEXT,        
    content TEXT NOT NULL,  
    image_url TEXT,  
    author VARCHAR(255), 
    published_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tối ưu khi lọc bài viết theo nguồn hoặc danh mục
CREATE INDEX idx_articles_source_id ON articles(source_id);
CREATE INDEX idx_articles_category_id ON articles(category_id);

-- Tối ưu khi sắp xếp bài viết mới nhất (ORDER BY published_at DESC)
CREATE INDEX idx_articles_published_at ON articles(published_at DESC);

INSERT INTO sources (name, base_url, rss_url) VALUES 
('VnExpress Events', 'https://vnexpress.net/thoi-su', 'https://vnexpress.net/rss/thoi-su.rss'),
('VnExpress Business', 'https://vnexpress.net/kinh-doanh', 'https://vnexpress.net/rss/kinh-doanh.rss'),
('VnExpress World', 'https://vnexpress.net/the-gioi', 'https://vnexpress.net/rss/the-gioi.rss'),
('VnExpress Law', 'https://vnexpress.net/phap-luat', 'https://vnexpress.net/rss/phap-luat.rss'),
('VnExpress Real Estate', 'https://vnexpress.net/bat-dong-san', 'https://vnexpress.net/rss/bat-dong-san.rss'),
('VnExpress Sports', 'https://vnexpress.net/the-thao', 'https://vnexpress.net/rss/the-thao.rss');