package db

import (
	"database/sql"
	_ "modernc.org/sqlite"
	"log"
)

func InitDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath+"?_busy_timeout=5000")
	if err != nil {
		return nil, err
	}

	createTableSQL := `
	CREATE TABLE IF NOT EXISTS files (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		host_ip TEXT NOT NULL,
		file_name TEXT NOT NULL,
		file_path TEXT NOT NULL UNIQUE,
		last_update DATETIME,
		original_md5 TEXT,
		latest_md5 TEXT,
		scan_time DATETIME,
		is_deleted BOOLEAN DEFAULT FALSE
	);
	`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		return nil, err
	}
	log.Println("数据库初始化成功")
	return db, nil
}