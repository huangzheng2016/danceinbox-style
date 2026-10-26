// Package container holds the process-wide services. Handlers and middleware
// reach them through Default; tests build services directly instead.
package container

// go generate skips files excluded by build tags, so wire.go's directive lives here.
//
//go:generate go tool wire

import (
	"MODULE_NAME/biz/extension/cache"
	"MODULE_NAME/biz/service/attachment"
	"MODULE_NAME/biz/service/example"
	"MODULE_NAME/biz/service/run"
	"MODULE_NAME/biz/service/user"
)

type Container struct {
	Cache      cache.Cache
	User       *user.Service
	Example    *example.Service
	Run        *run.Service
	Attachment *attachment.Service
}

var Default *Container

// Init runs after db.Init.
func Init() error {
	c, err := build()
	if err != nil {
		return err
	}
	Default = c
	return nil
}
