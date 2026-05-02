package main

import (
	"log"
	"os"
	"time"

	"devtoolbox/internal/app"
)

func main() {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		log.Fatalf("load timezone Asia/Shanghai: %v", err)
	}
	time.Local = loc

	// 统一默认数据目录，确保本地启动和 Docker 共用同一套数据
	if os.Getenv("DATA_DIR") == "" {
		os.Setenv("DATA_DIR", "./data")
	}
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}
	if err := app.New().Run(addr); err != nil {
		log.Fatal(err)
	}
}
