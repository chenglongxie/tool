package api

import (
	"chenglongxie/file_integrity/models"
	"database/sql"
	"encoding/json"
	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func SetupRoutes(router *mux.Router, db *sql.DB, watcher *fsnotify.Watcher) {
	router.HandleFunc("/files", func(w http.ResponseWriter, r *http.Request) {
		AddFileToWatch(w, r, db, watcher)
	}).Methods("POST")

	router.HandleFunc("/files/{filepath}", func(w http.ResponseWriter, r *http.Request) {
		RemoveFileFromWatch(w, r, db, watcher)
	}).Methods("DELETE")

	router.HandleFunc("/files", func(w http.ResponseWriter, r *http.Request) {
		ListAllFiles(w, r, db)
	}).Methods("GET")
}

func AddFileToWatch(w http.ResponseWriter, r *http.Request, db *sql.DB, watcher *fsnotify.Watcher) {
	var file models.FileRecord
	if err := json.NewDecoder(r.Body).Decode(&file); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	hostIP := file.HostIP
	if hostIP == "" {
		hostIP = "127.0.0.1"
	}

	fileName := ""
	if file.Filepath != "" {
		fileName = filepath.Base(file.Filepath)
	}

	info, err := os.Stat(file.Filepath)
	if os.IsNotExist(err) {
		http.Error(w, "文件不存在", http.StatusBadRequest)
		return
	}

	md5Sum, err := models.CalculateMD5(file.Filepath)
	if err != nil {
		http.Error(w, "无法计算 MD5", http.StatusInternalServerError)
		return
	}

	file.FileName = fileName
	file.LastUpdate = info.ModTime()
	file.OriginalMD5 = md5Sum
	file.LatestMD5 = md5Sum
	file.ScanTime = time.Now()
	file.HostIP = hostIP

	if err := models.InsertOrUpdateFileRecord(db, file); err != nil {
		http.Error(w, "插入或更新数据库失败", http.StatusInternalServerError)
		return
	}

	if err := watcher.Add(file.Filepath); err != nil {
		log.Printf("添加文件到监控失败: %v", err)
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(file)
}

func RemoveFileFromWatch(w http.ResponseWriter, r *http.Request, db *sql.DB, watcher *fsnotify.Watcher) {
	params := mux.Vars(r)
	filePath := params["filepath"]

	_, err := db.Exec("UPDATE files SET is_deleted = TRUE WHERE file_path = ?", filePath)
	if err != nil {
		http.Error(w, "删除文件失败", http.StatusInternalServerError)
		return
	}

	if err := watcher.Remove(filePath); err != nil {
		log.Printf("停止监听失败: %v", err)
	}
	log.Printf("停止监控文件: %s", filePath)

	w.WriteHeader(http.StatusOK)
}

func ListAllFiles(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	files, err := models.GetAllActiveFiles(db)
	if err != nil {
		http.Error(w, "获取文件列表失败", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(files)
}