// Package db owns the database handle. Business code reaches the database only
// through the generated query package returned by Query().
package db

import (
	"context"
	"os"
	"path/filepath"

	"MODULE_NAME/biz/dal/db/model"
	"MODULE_NAME/biz/dal/db/query"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	gdb *gorm.DB
	q   *query.Query
)

// Init opens the SQLite file at path (pure Go driver, no CGO).
func Init(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	cfg := &gorm.Config{TranslateError: true, Logger: logger.Default.LogMode(logger.Warn)}
	if os.Getenv("SQL_DEBUG") != "" {
		cfg.Logger = logger.Default.LogMode(logger.Info)
	}
	dsn := path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
	var err error
	if gdb, err = gorm.Open(sqlite.Open(dsn), cfg); err != nil {
		return err
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return err
	}
	// SQLite allows one writer; a single connection avoids SQLITE_BUSY under load.
	sqlDB.SetMaxOpenConns(1)
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
