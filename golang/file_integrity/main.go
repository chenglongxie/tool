package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/mux"

	"chenglongxie/file_integrity/config"
	"chenglongxie/file_integrity/db"
	"chenglongxie/file_integrity/monitor"
	"chenglongxie/file_integrity/api"
)

func main() {
	if err := config.LoadConfig("file_integrity.yaml"); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	dbConn, err := db.InitDB(config.Get().Database.Path)
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	defer dbConn.Close()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("无法创建文件监听器: %v", err)
	}
	defer watcher.Close()

	go monitor.MonitorFiles(watcher, dbConn)

	if err := monitor.ReloadWatcherFromDB(watcher, dbConn); err != nil {
		log.Printf("首次加载文件监控失败: %v", err)
	}

	r := mux.NewRouter()
	api.SetupRoutes(r, dbConn, watcher)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", config.Get().Server.Port),
		Handler:      r,
		ReadTimeout:  5 * time.Minute,
		WriteTimeout: 5 * time.Minute,
	}

	log.Printf("服务启动于 http://%s:%d", config.Get().HostIP,config.Get().Server.Port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("HTTP 服务启动失败: %v", err)
	}
}