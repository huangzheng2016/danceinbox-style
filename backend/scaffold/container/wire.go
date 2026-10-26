//go:build wireinject

package container

import (
	"MODULE_NAME/biz/dal/db"
	"MODULE_NAME/biz/extension/cache"
	"MODULE_NAME/biz/service/attachment"
	"MODULE_NAME/biz/service/example"
	"MODULE_NAME/biz/service/run"
	"MODULE_NAME/biz/service/user"

	"github.com/google/wire"
)

// A new service: add its constructor here and a field to Container, then
// run go generate ./container.
func build() (*Container, error) {
	wire.Build(
		db.Query,
		cache.New,
		user.NewService,
		example.NewService,
		run.NewService,
		run.NewDemoAgent,
		attachment.NewService,
		wire.Struct(new(Container), "*"),
	)
	return nil, nil
}
