#!/bin/bash

# 项目名称（默认为当前目录名）
PROJECT_NAME=$(basename "$PWD")

# 默认输出目录
OUTPUT_DIR="./build"

# 创建输出目录（如果不存在）
mkdir -p "$OUTPUT_DIR"

# 定义要编译的目标平台列表
# 格式: GOOS/GOARCH
PLATFORMS=(
    "linux/arm64"
    "linux/amd64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)

# 遍历每个平台并执行编译
for PLATFORM in "${PLATFORMS[@]}"
do
    # 分割GOOS和GOARCH
    IFS='/' read -r GOOS GOARCH <<< "$PLATFORM"

    # 输出信息
    echo "Building for $GOOS/$GOARCH..."

    # 设置环境变量
    export GOOS=$GOOS
    export GOARCH=$GOARCH
    export CGO_ENABLED=0

    # 编译命令
    if [ "$GOOS" == "windows" ]; then
        go build -o "$OUTPUT_DIR/${PROJECT_NAME}_${GOOS}_${GOARCH}.exe"
    else
        go build -o "$OUTPUT_DIR/${PROJECT_NAME}_${GOOS}_${GOARCH}"
    fi

    # 检查编译结果
    if [ $? -ne 0 ]; then
        echo "Error occurred while building for $GOOS/$GOARCH."
        exit 1
    fi
done

echo "Build process completed successfully!"