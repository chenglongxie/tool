# File Integrity

## 项目创建过程

1. 初始化项目

```bash
mkdir file_integrity
cd file_integrity
go mod init chenglongxie/file_integrity
```

2. 添加依赖包
```bash
go get github.com/fsnotify/fsnotify
go get github.com/gorilla/mux
go get gopkg.in/yaml.v2
go get modernc.org/sqlite

```