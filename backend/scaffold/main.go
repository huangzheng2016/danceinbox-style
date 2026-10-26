package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"MODULE_NAME/biz/dal/db"
	"MODULE_NAME/biz/handler"
	"MODULE_NAME/biz/mw"
	"MODULE_NAME/config"
	"MODULE_NAME/container"

	"github.com/cloudwego/hertz/pkg/app/server"
)

// set by build.sh: -ldflags "-X main.version=$VERSION"
var version = "dev"

const usage = `usage:
  app [-c local]                              serve
  PASSWORD=... app [-c local] adduser NAME ROLE   create a user (ROLE: USER, MODERATOR, ADMIN)`

func main() {
	name := flag.String("c", "local", "config name: reads config/<name>.json")
	flag.Usage = func() { fmt.Fprintln(os.Stderr, usage) }
	flag.Parse()

	if err := config.Load(*name); err != nil {
		log.Fatal(err)
	}
	if err := db.Init(config.C.DB.DSN); err != nil {
		log.Fatalf("db: %v", err)
	}
	if err := container.Init(); err != nil {
		log.Fatalf("container: %v", err)
	}

	switch flag.Arg(0) {
	case "":
		serve()
	case "adduser":
		u, err := container.Default.User.Create(context.Background(), flag.Arg(1), os.Getenv("PASSWORD"), flag.Arg(2))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("created user %s (id %d, %s)\n", u.Username, u.ID, u.Role)
	default:
		flag.Usage()
		os.Exit(2)
	}
}

func serve() {
	handler.Version = version
	h := server.Default(
		server.WithHostPorts(config.C.Server.Addr),
		// uploads are the largest bodies; leave room for the multipart framing
		server.WithMaxRequestBodySize((config.C.Upload.MaxMB+1)<<20),
		server.WithExitWaitTime(5*time.Second),
	)
	h.Use(mw.AccessLog())
	register(h)
	h.Spin()
}
