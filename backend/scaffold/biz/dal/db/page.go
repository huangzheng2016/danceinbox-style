package db

import (
	"strings"

	"MODULE_NAME/biz/errs"

	"gorm.io/gen/field"
)

const (
	DefaultPageSize = 20
	MaxPageSize     = 50
)

// Page turns a request's 1-based page and size into FindByPage arguments.
// Missing or out-of-range values fall back to page 1 and the default size.
func Page(page, size int32) (offset, limit int) {
	if size <= 0 {
		size = DefaultPageSize
	}
	if size > MaxPageSize {
		size = MaxPageSize
	}
	if page < 1 {
		page = 1
	}
	return int(page-1) * int(size), int(size)
}

// Contains matches s literally anywhere in col: % and _ in user input are not
// wildcards. The explicit ESCAPE makes it behave the same on MySQL and SQLite.
func Contains(col field.String, s string) field.Expr {
	r := strings.NewReplacer("!", "!!", "%", "!%", "_", "!_")
	return field.NewUnsafeFieldRaw("? LIKE ? ESCAPE '!'", col, "%"+r.Replace(s)+"%")
}

// Order turns a list's sort parameter ("name" ascending, "-name" descending) into
// an ORDER BY expression. Only keys of allowed are accepted; "" returns nil so
// the caller applies its default order.
func Order(sort string, allowed map[string]field.OrderExpr) (field.Expr, error) {
	if sort == "" {
		return nil, nil
	}
	col, ok := allowed[strings.TrimPrefix(sort, "-")]
	if !ok {
		return nil, errs.Invalid("不支持按该字段排序")
	}
	if strings.HasPrefix(sort, "-") {
		return col.Desc(), nil
	}
	return col, nil
}
