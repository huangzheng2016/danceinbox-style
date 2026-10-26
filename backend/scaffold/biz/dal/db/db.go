// Package db owns the database handle. Business code reaches the database only
// through the generated query package returned by Query().
package db

import (
	"context"
	"os"
	"time"

	"MODULE_NAME/biz/dal/db/model"
	"MODULE_NAME/biz/dal/db/query"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	gdb *gorm.DB
	q   *query.Query
)

func Init(dsn string) error {
	cfg := &gorm.Config{TranslateError: true, Logger: logger.Default.LogMode(logger.Warn)}
	if os.Getenv("SQL_DEBUG") != "" {
		cfg.Logger = logger.Default.LogMode(logger.Info)
	}
	var err error
	if gdb, err = gorm.Open(mysql.Open(dsn), cfg); err != nil {
		return err
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return err
	}
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(time.Hour)
	if err := gdb.AutoMigrate(model.All()...); err != nil {
		return err
	}
	q = query.Use(gdb)
	return nil
}

// Query is the Wire provider services depend on.
func Query() *query.Query { return q }

func Ping(ctx context.Context) error {
	sqlDB, err := gdb.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}
