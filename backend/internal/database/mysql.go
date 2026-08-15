package database

import (
	"net"
	"strconv"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	gormMySQL "gorm.io/driver/mysql"
	"gorm.io/gorm"

	"personal_assistant_server/internal/config"
)

// connectMySQL 根据配置创建 MySQL 数据库连接。
// 参数 cfg 包含服务器、账户及数据库信息；返回数据库连接和连接错误。
func connectMySQL(cfg config.DatabaseConnectionConfig) (*gorm.DB, error) {
	return gorm.Open(gormMySQL.Open(mysqlDSN(cfg)), &gorm.Config{})
}

// mysqlDSN 根据 MySQL 配置生成驱动连接字符串。
// 参数 cfg 包含服务器、账户及数据库信息；返回支持特殊字符转义的 MySQL DSN。
func mysqlDSN(cfg config.DatabaseConnectionConfig) string {
	dsnConfig := mysqlDriver.Config{
		User:      cfg.Username,
		Passwd:    cfg.Password,
		Net:       "tcp",
		Addr:      net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)),
		DBName:    cfg.Name,
		Params:    map[string]string{"charset": cfg.Charset},
		ParseTime: true,
		Loc:       time.Local,
	}
	return dsnConfig.FormatDSN()
}
