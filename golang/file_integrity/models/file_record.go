package models

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"os"
	"time"

	_ "modernc.org/sqlite" // SQLite数据库驱动
	"io"                            // 添加io包用于文件读取
	"log"
)

type FileRecord struct {
	ID           int       `json:"id"`
	HostIP       string    `json:"host_ip"`
	FileName     string    `json:"file_name"`
	Filepath     string    `json:"file_path"`
	LastUpdate   time.Time `json:"last_update"`
	OriginalMD5  string    `json:"original_md5"`
	LatestMD5    string    `json:"latest_md5"`
	ScanTime     time.Time `json:"scan_time"`
	IsDeleted    bool      `json:"is_deleted"`
}

func InsertOrUpdateFileRecord(db *sql.DB, record FileRecord) error {
	var id int
	err := db.QueryRow("SELECT id FROM files WHERE file_path = ?", record.Filepath).Scan(&id)
	if err == sql.ErrNoRows {
		_, err = db.Exec(`INSERT INTO files(host_ip, file_name, file_path, last_update, original_md5, latest_md5, scan_time) VALUES(?, ?, ?, ?, ?, ?, ?)`,
			record.HostIP, record.FileName, record.Filepath, record.LastUpdate, record.OriginalMD5, record.LatestMD5, record.ScanTime)
	} else if err != nil {
		return err
	} else {
		_, err = db.Exec(`UPDATE files SET last_update=?, latest_md5=?, scan_time=?, is_deleted=? WHERE file_path=?`,
			record.LastUpdate, record.LatestMD5, record.ScanTime, false, record.Filepath)
	}
	if err != nil {
		log.Printf("更新文件记录失败: %v", err)
	}
	return err
}

func GetAllActiveFiles(db *sql.DB) ([]FileRecord, error) {
	rows, err := db.Query("SELECT * FROM files WHERE is_deleted = FALSE")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []FileRecord
	for rows.Next() {
		var file FileRecord
		if err := rows.Scan(
			&file.ID,
			&file.HostIP,
			&file.FileName,
			&file.Filepath,
			&file.LastUpdate,
			&file.OriginalMD5,
			&file.LatestMD5,
			&file.ScanTime,
			&file.IsDeleted,
		); err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, nil
}

func CalculateMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil { // 使用io.Copy进行文件内容读取
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}