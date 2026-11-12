package database

import (
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/laoin114514/gxuschedule-server/internal/config"
	"github.com/laoin114514/gxuschedule-server/internal/model"
)

// New 连接 MySQL 并自动建表/补列。
//
// 生产首次升级建议先执行 migrations/001_per_app_keys.sql（等价变更，便于备份与审阅）；
// 即使不执行，AutoMigrate 也会补齐新表与新列（新增列都带默认值，存量行可安全回填）。
func New(cfg *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=%s&parseTime=True&loc=Local",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBCharset)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("connect mysql(%s:%s): %w", cfg.DBHost, cfg.DBPort, err)
	}
	if err := db.AutoMigrate(
		&model.App{},
		&model.AppUploadKey{},
		&model.Release{},
		&model.AuditLog{},
	); err != nil {
		return nil, fmt.Errorf("auto migrate: %w", err)
	}
	return db, nil
}
