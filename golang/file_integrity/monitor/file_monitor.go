package monitor

import (
	"crypto/md5"
	"encoding/hex"
	"github.com/fsnotify/fsnotify"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"chenglongxie/file_integrity/models"
	"chenglongxie/file_integrity/config"
	"database/sql"
)

func CalculateMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func ReloadWatcherFromDB(watcher *fsnotify.Watcher, db *sql.DB) error {
	rows, err := db.Query("SELECT file_path FROM files WHERE is_deleted = FALSE")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			continue
		}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			log.Printf("文件不存在，跳过监控: %s", path)
			continue
		}
		err := watcher.Add(path)
		if err != nil {
			log.Printf("添加文件监控失败: %s - %v", path, err)
		} else {
			log.Printf("开始监控文件: %s", path)
		}
	}
	return nil
}

func MonitorFiles(watcher *fsnotify.Watcher, db *sql.DB) {
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				info, err := os.Stat(event.Name)
				if err != nil {
					log.Printf("获取文件信息失败: %v", err)
					continue
				}
				md5Sum, err := CalculateMD5(event.Name)
				if err != nil {
					log.Printf("计算 MD5 失败: %v", err)
					continue
				}
				record := models.FileRecord{
					HostIP:     config.Get().HostIP,
					FileName:   filepath.Base(event.Name),
					Filepath:   event.Name,
					LastUpdate: info.ModTime(),
					LatestMD5:  md5Sum,
					ScanTime:   time.Now(),
				}
				models.InsertOrUpdateFileRecord(db, record)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Println("监听错误:", err)
		}
	}
}