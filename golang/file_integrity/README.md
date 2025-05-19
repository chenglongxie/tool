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

## 说明

### 功能总结

1. **文件监控**：
   - 定期检查数据库中的文件记录。
   - 如果文件不存在，则更新记录的 `IsDeleted` 状态为 `true`。

2. **MD5计算**：
   - 在文件发生变化时，重新计算文件的MD5哈希值，并更新记录。

3. **数据库操作**：
   - 将文件信息存储在SQLite数据库中。
   - 支持插入、更新和物理删除操作。

4. **API接口**：
   - `/files` (POST): 添加文件到监控列表。`HostIP` 采用配置文件中的配置，不接收接口传递。如果文件不存在则返回错误。
   - `/files/{id}` (DELETE): 通过ID物理删除文件记录。
   - `/files` (GET): 列出所有文件及其状态。

5. **配置管理**：
   - 通过 YAML 文件加载配置，包括主机IP、服务器端口、数据库路径和定时任务时间间隔。

### 配置文件示例 (`file_integrity.yaml`)
```yaml
host_ip: "127.0.0.1"
server:
  port: 8080
database:
  path: "./file_integrity.db"
check_interval: 30 # 定时任务时间间隔（秒）
```

### 运行环境要求
- 安装了Go语言环境。
- 安装了 `github.com/gorilla/mux` 和 `modernc.org/sqlite` 包。
- 确保有权限访问和修改指定的文件路径。
- 创建并配置 `file_integrity.yaml` 文件。

### 安装依赖包
```sh
go get github.com/gorilla/mux
go get modernc.org/sqlite
go get gopkg.in/yaml.v2
```

### 运行程序
```sh
go run main.go
```

这将启动文件监控系统，并开始监控指定目录下的文件变化。

### 使用示例

#### 1. 添加文件到监控列表
假设你要监控的文件路径为 `/path/to/your/file.txt`，可以使用以下命令添加该文件：
```sh
curl -X POST -H "Content-Type: application/json" -d '{"file_path": "/path/to/your/file.txt"}' http://localhost:8080/files
```
成功后，你会收到类似如下的响应：
```json
{
  "id": 1,
  "host_ip": "127.0.0.1",
  "file_name": "file.txt",
  "file_path": "/path/to/your/file.txt",
  "last_update": "2023-10-01T12:34:56Z",
  "original_md5": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
  "latest_md5": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
  "scan_time": "2023-10-01T12:34:56Z",
  "is_deleted": false
}
```

#### 2. 物理删除文件记录
假设你想删除ID为1的文件记录，可以使用以下命令：
```sh
curl -X DELETE http://localhost:8080/files/1
```
成功后，你会收到一个空的响应，状态码为200 OK。

#### 3. 列出所有文件
如果你想查看所有被监控的文件及其状态，可以使用以下命令：
```sh
curl http://localhost:8080/files
```
成功后，你会收到类似如下的响应：
```json
[
  {
    "id": 1,
    "host_ip": "127.0.0.1",
    "file_name": "file.txt",
    "file_path": "/path/to/your/file.txt",
    "last_update": "2023-10-01T12:34:56Z",
    "original_md5": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
    "latest_md5": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
    "scan_time": "2023-10-01T12:34:56Z",
    "is_deleted": false
  }
]
```

确保 `file_integrity.yaml` 文件存在于项目根目录下，并且配置正确。这样你就可以顺利运行文件监控系统并进行相应的操作了。