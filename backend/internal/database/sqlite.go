package database

import (
	"fmt"
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// connectSQLite 创建 SQLite 数据库连接，并在需要时创建数据库文件的父目录。
// 参数 path 是 SQLite 文件路径或 :memory:；返回数据库连接和目录创建或连接错误。
func connectSQLite(path string) (*gorm.DB, error) {
	if path != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, fmt.Errorf("create SQLite database directory: %w", err)
		}
	}
	return gorm.Open(sqlite.Open(path), &gorm.Config{})
}
