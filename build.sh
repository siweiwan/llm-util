#!/bin/bash
# 从 .env 文件加载凭据（本地开发用）
if [ -f .env ]; then
  export $(grep -v '^#' .env | grep -v '^$' | xargs)
fi

# 动态程序名：从 go.mod 取模块名，可通过 BIN_NAME 环境变量覆盖
BIN_NAME=${BIN_NAME:-$(grep '^module ' go.mod | awk '{print $2}')}
OUTPUT="${BIN_NAME}.exe"

# 使用 ldflags 将 AK/SK 嵌入二进制（源码中不出现明文凭据）
LD_FLAGS="-s -w"
if [ -n "$ALIBABA_CLOUD_ACCESS_KEY_ID" ]; then
  LD_FLAGS="$LD_FLAGS -X 'llm-util/file/qwen.AccessKeyId=$ALIBABA_CLOUD_ACCESS_KEY_ID'"
fi
if [ -n "$ALIBABA_CLOUD_ACCESS_KEY_SECRET" ]; then
  LD_FLAGS="$LD_FLAGS -X 'llm-util/file/qwen.AccessKeySecret=$ALIBABA_CLOUD_ACCESS_KEY_SECRET'"
fi

GOOS=windows GOARCH=amd64 go build -ldflags "$LD_FLAGS" -o "$OUTPUT" main.go
echo "Built: $OUTPUT"
