package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	_ "modernc.org/sqlite"
	"gopkg.in/yaml.v2"
)

type Config struct {
	HostIP string `yaml:"host_ip"`
	Server struct {
		Port int `yaml:"port"`
	} `yaml:"server"`
	Database struct {
		Path string `yaml:"path"`
	} `yaml:"database"`
	CheckInterval int `yaml:"check_interval"` // 定时任务时间间隔（秒）
}

var cfg Config

func LoadConfig(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return err
	}
	return nil
}

func GetConfig() *Config {
	return &cfg
}

// 自定义时间格式
type CustomTime time.Time

func (ct CustomTime) MarshalJSON() ([]byte, error) {
	t := time.Time(ct)
	return json.Marshal(t.Format("2006-01-02 15:04:05"))
}

func (ct *CustomTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	t, err := time.Parse("2006-01-02 15:04:05", s)
	if err != nil {
		return err
	}
	*ct = CustomTime(t)
	return nil
}

type FileRecord struct {
	ID           int         `json:"id"`
	HostIP       string      `json:"host_ip"`
	FileName     string      `json:"file_name"`
	Filepath     string      `json:"file_path"`
	LastUpdate   CustomTime  `json:"last_update"`
	OriginalMD5  string      `json:"original_md5"`
	LatestMD5    string      `json:"latest_md5"`
	ScanTime     CustomTime  `json:"scan_time"`
	IsDeleted    bool        `json:"is_deleted"`
}

// 插入或更新文件记录
func InsertOrUpdateFileRecord(db *sql.DB, record FileRecord) error {
	var id int
	err := db.QueryRow("SELECT id FROM files WHERE file_path = ?", record.Filepath).Scan(&id)
	if err == sql.ErrNoRows {
		_, err = db.Exec(`INSERT INTO files(host_ip, file_name, file_path, last_update, original_md5, latest_md5, scan_time, is_deleted) VALUES(?, ?, ?, ?, ?, ?, ?, ?)`,
			record.HostIP, record.FileName, record.Filepath, time.Time(record.LastUpdate), record.OriginalMD5, record.LatestMD5, time.Time(record.ScanTime), record.IsDeleted)
	} else if err != nil {
		return err
	} else {
		_, err = db.Exec(`UPDATE files SET last_update=?, latest_md5=?, scan_time=?, is_deleted=? WHERE file_path=?`,
			time.Time(record.LastUpdate), record.LatestMD5, time.Time(record.ScanTime), record.IsDeleted, record.Filepath)
	}
	if err != nil {
		log.Printf("更新文件记录失败: %v", err)
	}
	return err
}

// 获取文件记录
func GetAllFiles(db *sql.DB) ([]FileRecord, error) {
	rows, err := db.Query("SELECT * FROM files")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []FileRecord
	for rows.Next() {
		var file FileRecord
		var lastUpdateTime, scanTime time.Time
		if err := rows.Scan(
			&file.ID,
			&file.HostIP,
			&file.FileName,
			&file.Filepath,
			&lastUpdateTime,
			&file.OriginalMD5,
			&file.LatestMD5,
			&scanTime,
			&file.IsDeleted,
		); err != nil {
			return nil, err
		}
		file.LastUpdate = CustomTime(lastUpdateTime)
		file.ScanTime = CustomTime(scanTime)
		files = append(files, file)
	}
	return files, nil
}

// 物理删除文件记录
func DeleteFileRecordByID(db *sql.DB, id int) error {
	_, err := db.Exec("DELETE FROM files WHERE id = ?", id)
	return err
}

// 计算文件的MD5值
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

// 初始化数据库
func InitDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	createTableSQL := `
	CREATE TABLE IF NOT EXISTS files (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		host_ip TEXT NOT NULL,
		file_name TEXT NOT NULL,
		file_path TEXT NOT NULL UNIQUE,
		last_update DATETIME NOT NULL,
		original_md5 TEXT NOT NULL,
		latest_md5 TEXT NOT NULL,
		scan_time DATETIME NOT NULL,
		is_deleted BOOLEAN NOT NULL DEFAULT FALSE
	);
	`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		return nil, err
	}

	return db, nil
}

// 定时任务：检查文件状态并更新数据库
func CheckFilesPeriodically(db *sql.DB) {
	interval := time.Duration(GetConfig().CheckInterval) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		files, err := GetAllFiles(db)
		if err != nil {
			log.Printf("获取文件列表失败: %v", err)
			continue
		}

		for _, file := range files {
			info, err := os.Stat(file.Filepath)
			if os.IsNotExist(err) {
				// 文件不存在，更新记录的 IsDeleted 状态为 true
				file.IsDeleted = true
				if err := InsertOrUpdateFileRecord(db, file); err != nil {
					log.Printf("更新文件记录失败: %v", err)
				}
				log.Printf("文件不存在，更新记录的 IsDeleted 状态为 true: %s", file.Filepath)
				continue
			}

			if !info.ModTime().Equal(time.Time(file.LastUpdate)) {
				// 文件存在且最后更新时间不一致，重新计算MD5值
				md5Sum, err := CalculateMD5(file.Filepath)
				if err != nil {
					log.Printf("计算 MD5 失败: %v", err)
					continue
				}
				file.LastUpdate = CustomTime(info.ModTime())
				file.LatestMD5 = md5Sum
				file.ScanTime = CustomTime(time.Now())
				if err := InsertOrUpdateFileRecord(db, file); err != nil {
					log.Printf("更新文件记录失败: %v", err)
				}
				log.Printf("文件更新，新MD5: %s", file.Filepath)
			}
		}
	}
}

// API handlers
func SetupRoutes(router *mux.Router, db *sql.DB) {
	router.HandleFunc("/files", func(w http.ResponseWriter, r *http.Request) {
		AddFileToWatch(w, r, db)
	}).Methods("POST")

	router.HandleFunc("/files/{id}", func(w http.ResponseWriter, r *http.Request) {
		DeleteFileRecordHandler(w, r, db)
	}).Methods("DELETE")

	router.HandleFunc("/files", func(w http.ResponseWriter, r *http.Request) {
		ListAllFiles(w, r, db)
	}).Methods("GET")
}

func AddFileToWatch(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var file FileRecord
	if err := json.NewDecoder(r.Body).Decode(&file); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	filePath := file.Filepath
	if filePath == "" {
		http.Error(w, "文件路径不能为空", http.StatusBadRequest)
		return
	}

	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		http.Error(w, "文件不存在", http.StatusBadRequest)
		return
	}

	hostIP := GetConfig().HostIP
	fileName := filepath.Base(filePath)

	md5Sum, err := CalculateMD5(filePath)
	if err != nil {
		http.Error(w, "无法计算 MD5", http.StatusInternalServerError)
		return
	}

	file.FileName = fileName
	file.LastUpdate = CustomTime(info.ModTime())
	file.OriginalMD5 = md5Sum
	file.LatestMD5 = md5Sum
	file.ScanTime = CustomTime(time.Now())
	file.HostIP = hostIP
	file.IsDeleted = false

	if err := InsertOrUpdateFileRecord(db, file); err != nil {
		http.Error(w, "插入或更新数据库失败", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string][]FileRecord{"data": {file}})
}

func DeleteFileRecordHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	params := mux.Vars(r)
	idStr := params["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "无效的ID", http.StatusBadRequest)
		return
	}

	if err := DeleteFileRecordByID(db, id); err != nil {
		http.Error(w, "删除文件记录失败", http.StatusInternalServerError)
		return
	}

	log.Printf("物理删除文件记录: ID=%d", id)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "文件记录删除成功"})
}

func ListAllFiles(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	files, err := GetAllFiles(db)
	if err != nil {
		http.Error(w, "获取文件列表失败", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string][]FileRecord{"data": files})
}

func init() {
	if err := LoadConfig("./file_integrity.yaml"); err != nil {
		log.Fatalf("无法加载配置文件: %v", err)
	}
}

func main() {
	dbPath := GetConfig().Database.Path
	db, err := InitDB(dbPath)
	if err != nil {
		log.Fatalf("无法初始化数据库: %v", err)
	}
	defer db.Close()

	r := mux.NewRouter()
	SetupRoutes(r, db)

	go CheckFilesPeriodically(db)

	port := fmt.Sprintf(":%d", GetConfig().Server.Port)
	fmt.Printf("文件监控系统已启动，监听端口: %s...\n", port)
	http.ListenAndServe(port, r)
}



