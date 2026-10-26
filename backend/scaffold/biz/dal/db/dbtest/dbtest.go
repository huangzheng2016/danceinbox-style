// Package dbtest gives tests a fresh migrated database. It is imported only from
// _test.go files, so the SQLite driver never ends up in the production binary.
package dbtest

import (
	"path/filepath"
	"testing"

	"MODULE_NAME/biz/dal/db/model"
	"MODULE_NAME/biz/dal/db/query"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func New(t *testing.T) *query.Query {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	gdb, err := gorm.Open(sqlite.Open(path+"?_pragma=foreign_keys(1)"), &gorm.Config{
		TranslateError: true,
		Logger:         logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := gdb.AutoMigrate(model.All()...); err != nil {
		t.Fatal(err)
	}
	return query.Use(gdb)
}
