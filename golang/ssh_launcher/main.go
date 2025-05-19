package main

import (
	"bufio"
	"embed"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mattn/go-colorable"
)

var stdout = colorable.NewColorableStdout()

// 用于存储临时目录的路径
var tempDir string

//go:embed putty.exe
var puttyFS embed.FS

// 定义 SSH 连接结构体
type Connection struct {
	Name     string `yaml:"name"`
	User     string `yaml:"user"`
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Password string `yaml:"password"`
}

// 配置结构体，包含 PuTTY 路径和连接列表
type Config struct {
	PuttyPath   string      `yaml:"putty_path"`
	Connections []Connection `yaml:"connections"`
}

// 加载配置文件（如果不存在则返回空配置）
func loadConfig(filename string) (*Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("无法读取配置文件: %v", err)
	}
	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}
	return &config, nil
}

// 保存配置到 YAML 文件
func saveConfig(filename string, config *Config) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}
	err = ioutil.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("写入配置文件失败: %v", err)
	}
	return nil
}

// 模糊搜索连接：根据关键词匹配名称、用户名或主机名
func fuzzySearchConnections(query string, connections []Connection) []Connection {
	query = strings.ToLower(query)
	var results []Connection
	for _, conn := range connections {
		if strings.Contains(strings.ToLower(conn.Name), query) ||
			strings.Contains(strings.ToLower(conn.User), query) ||
			strings.Contains(strings.ToLower(conn.Host), query) {
			results = append(results, conn)
		}
	}
	return results
}

// 打印主菜单选项
func printMenu(config *Config) {
	fmt.Fprintln(stdout, "\n\033[95m【SSH 连接】\033[0m")

	for i, conn := range config.Connections {
		fmt.Fprintf(stdout, "  \033[94m%d.\033[0m %s (\033[93m%s\033[0m@\033[92m%s\033[0m:%s)\n",
			i+1, conn.Name, conn.User, conn.Host, conn.Port)
	}

	fmt.Fprintln(stdout, "\n\033[95m【操作选项】\033[0m")
	fmt.Fprintln(stdout, "  \033[94m0.\033[0m 退出")
	fmt.Fprintln(stdout, "  \033[94m-1.\033[0m 添加新连接")
	fmt.Fprintln(stdout, "  \033[94m-2.\033[0m 设置 PuTTY 路径")
	fmt.Fprintf(stdout, "  \033[94m请输入编号、逗号分隔的多个编号，或搜索关键字：\033[0m ")
}

// 提取并运行嵌入的 PuTTY 可执行文件
func extractPutty() (string, error) {
	tmpDir, err := ioutil.TempDir("", "putty")
	if err != nil {
		return "", err
	}
	tempDir = tmpDir // 记录临时目录路径以便后续清理

	puttyContent, err := puttyFS.ReadFile("putty.exe")
	if err != nil {
		return "", err
	}

	puttyPath := filepath.Join(tmpDir, "putty.exe")
	err = ioutil.WriteFile(puttyPath, puttyContent, 0755)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", err
	}

	return puttyPath, nil
}

// 启动单个 PuTTY 实例
func connectTo(conn Connection, puttyPath string) {
	fmt.Fprintf(stdout, "正在启动连接：%s@%s:%s ...\n", conn.User, conn.Host, conn.Port)

	cmd := exec.Command(
		puttyPath,
		"-ssh",
		"-l", conn.User,
		"-pw", conn.Password,
		"-P", conn.Port,
		conn.Host,
	)

	err := cmd.Start()
	if err != nil {
		fmt.Fprintf(stdout, "启动 PuTTY 失败: %v\n", err)
	}
}

// 启动多个 PuTTY 实例
func connectMultiple(connections []Connection, puttyPath string) {
	for _, conn := range connections {
		connectTo(conn, puttyPath)
	}
}

// 解析用户输入的编号或逗号分隔的多个编号
func parseSelection(selection string, max int) ([]int, error) {
	selectedIndexes := make([]int, 0)
	parts := strings.Split(selection, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		index, err := strconv.Atoi(part)
		if err != nil || index < 1 || index > max {
			return nil, fmt.Errorf("无效的输入: %s", part)
		}
		selectedIndexes = append(selectedIndexes, index)
	}
	return selectedIndexes, nil
}

// 获取 PuTTY 的路径，如果配置文件中指定了路径则使用，否则尝试从嵌入的资源中提取
func getPuttyPath(config *Config) (string, error) {
	if config.PuttyPath != "" {
		if _, err := os.Stat(config.PuttyPath); err == nil {
			return config.PuttyPath, nil
		} else {
			fmt.Fprintln(stdout, "配置文件中的 PuTTY 路径无效，请检查路径是否正确。")
		}
	}

	puttyPath, err := extractPutty()
	if err != nil {
		return "", fmt.Errorf("无法准备嵌入的 PuTTY: %v", err)
	}
	return puttyPath, nil
}

// 清理函数，用于在程序结束时删除临时目录
func cleanup() {
	if tempDir != "" {
		os.RemoveAll(tempDir)
	}
}

func main() {
	defer cleanup() // 程序结束时调用清理函数

	// 加载配置
	config, err := loadConfig("ssh.yaml")
	if err != nil {
		log.Fatal(err)
	}

	reader := bufio.NewReader(os.Stdin)

	// 获取 PuTTY 路径
	puttyPath, err := getPuttyPath(config)
	if err != nil {
		log.Fatal(err)
	}

	// 主循环，持续显示菜单并处理用户输入
	for {
		printMenu(config)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		switch input {
		case "0":
			fmt.Fprintln(stdout, "再见！感谢使用本工具。")
			return
		case "-1":
			addConnection(config, reader)
			saveConfig("ssh.yaml", config)
		case "-2":
			setPuttyPath(config, reader)
			saveConfig("ssh.yaml", config)
		default:
			// 尝试解析为编号或逗号分隔的多个编号
			selectedIndexes, err := parseSelection(input, len(config.Connections))
			if err == nil {
				var selectedConnections []Connection
				for _, index := range selectedIndexes {
					selectedConnections = append(selectedConnections, config.Connections[index-1])
				}
				connectMultiple(selectedConnections, puttyPath)
				continue
			}

			// 否则视为模糊搜索
			results := fuzzySearchConnections(input, config.Connections)
			if len(results) == 0 {
				fmt.Fprintln(stdout, "未找到匹配的连接，请重新输入。")
				continue
			}

			// 显示模糊搜索结果并让用户选择
			fmt.Fprintln(stdout, "\n\033[95m搜索结果如下：\033[0m")
			for i, conn := range results {
				fmt.Fprintf(stdout, "  \033[94m%d.\033[0m %s (\033[93m%s\033[0m@\033[92m%s\033[0m:%s)\n",
					i+1, conn.Name, conn.User, conn.Host, conn.Port)
			}
			fmt.Fprintf(stdout, "请选择要连接的序号（多个用逗号分隔）：")

			selectedInput, _ := reader.ReadString('\n')
			selectedIndexes, err = parseSelection(strings.TrimSpace(selectedInput), len(results))
			if err != nil {
				fmt.Fprintln(stdout, "输入无效，请重新选择。")
				continue
			}

			var selectedConnections []Connection
			for _, selectedIndex := range selectedIndexes {
				selectedConnections = append(selectedConnections, results[selectedIndex-1])
			}
			connectMultiple(selectedConnections, puttyPath)
		}
	}
}

// 添加新连接
func addConnection(config *Config, reader *bufio.Reader) {
	fmt.Fprintln(stdout, "\n正在添加新连接...")
	fmt.Fprint(stdout, "  名称: ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)

	fmt.Fprint(stdout, "  用户名: ")
	user, _ := reader.ReadString('\n')
	user = strings.TrimSpace(user)

	fmt.Fprint(stdout, "  主机地址: ")
	host, _ := reader.ReadString('\n')
	host = strings.TrimSpace(host)

	fmt.Fprint(stdout, "  端口 (默认22): ")
	port, _ := reader.ReadString('\n')
	port = strings.TrimSpace(port)
	if port == "" {
		port = "22"
	}

	fmt.Fprint(stdout, "  密码: ")
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)

	newConn := Connection{
		Name:     name,
		User:     user,
		Host:     host,
		Port:     port,
		Password: password,
	}
	config.Connections = append(config.Connections, newConn)
	fmt.Fprintln(stdout, "连接已添加。\n")
}

// 设置 PuTTY 路径
func setPuttyPath(config *Config, reader *bufio.Reader) {
	fmt.Fprint(stdout, "请输入 PuTTY 安装路径（例如 C:\\Program Files\\PuTTY\\putty.exe）: ")
	puttyPath, _ := reader.ReadString('\n')
	puttyPath = strings.TrimSpace(puttyPath)

	if _, err := os.Stat(puttyPath); os.IsNotExist(err) {
		fmt.Fprintln(stdout, "错误：指定的路径下没有找到 PuTTY，请确认路径是否正确。")
	} else {
		config.PuttyPath = puttyPath
		fmt.Fprintln(stdout, "PuTTY 路径已设置。\n")
	}
}