package main

import (
	"fmt"
	"log"
	"net"

	"gorm.io/gorm"

	"personal_assistant_server/internal/config"
	"personal_assistant_server/internal/database"
	utils "personal_assistant_server/internal/goutils"
	"personal_assistant_server/internal/router"
)

func main() {
	/*
		应用初始化或运行失败时，先使用 Zap 记录完整错误，再主动刷新日志缓冲区，
		最后通过标准日志终止进程，避免 log.Fatalf 直接退出导致 Zap 日志未写入完成。
	*/
	db, err := bootstrap()
	if err != nil {
		utils.SugaredLogger.Errorw("personal-assistant api stopped", "error", err)
		utils.SyncZap()
		log.Fatalf("personal-assistant api stopped: %v", err)
		return
	}
	defer utils.SyncZap()

	/*
		启动 HTTP 服务；服务启动或运行失败时，记录完整错误并刷新 Zap 日志，
		随后通过标准日志终止进程，确保退出前日志已经写入完成。
	*/
	if err := run(db); err != nil {
		utils.SugaredLogger.Errorw("personal-assistant api stopped", "error", err)
		utils.SyncZap()
		log.Fatalf("personal-assistant api stopped: %v", err)
	}
}

// run 创建并启动 HTTP 服务。
func run(db *gorm.DB) error {
	cfg := config.Get()
	engine := router.New(db, cfg)
	address := net.JoinHostPort(cfg.Host, cfg.Port)
	utils.SugaredLogger.Infow(
		"HTTP 服务启动",
		"address", "http://"+address,
		"mode", cfg.Mode,
	)
	if err := engine.Run(address); err != nil {
		return fmt.Errorf("run HTTP server on %s: %w", address, err)
	}

	return nil
}

// bootstrap 初始化应用运行所需的配置、日志和数据库。
//
// 函数依次加载全局配置、记录当前数据存储信息、建立数据库连接，
// 并执行数据库表迁移和初始数据写入；任一阶段失败时立即终止后续流程。
//
// 参数：
//   - 无。
//
// 返回值：
//   - *gorm.DB：初始化完成的数据库连接；初始化失败时返回 nil。
//   - error：配置初始化、数据库连接、表迁移或初始数据写入失败时返回错误；
//     全部初始化阶段执行成功时返回 nil。
func bootstrap() (*gorm.DB, error) {
	if err := config.Initialize(); err != nil {
		return nil, fmt.Errorf("应用初始化失败：%w", err)
	}
	cfg := config.Get()
	connectionName, connectionConfig, err := cfg.ActiveDatabaseConnection()
	if err != nil {
		return nil, fmt.Errorf("选择当前数据库连接失败：%w", err)
	}
	utils.SugaredLogger.Infow(
		"数据存储配置加载完成",
		"连接名称", connectionName,
		"数据库驱动", connectionConfig.Driver,
		"文件路径", connectionConfig.Path,
		"服务器地址", connectionConfig.Host,
		"服务端口", connectionConfig.Port,
		"数据库名称", connectionConfig.Name,
	)

	db, err := database.Connect(connectionConfig)
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败：%w", err)
	}
	if err := database.MigrateAndSeed(db); err != nil {
		return nil, fmt.Errorf("初始化数据库失败：%w", err)
	}

	return db, nil
}
