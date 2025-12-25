#!/bin/bash

# Phase 2 功能测试脚本

echo "🧪 Phase 2 功能测试"
echo "===================="
echo ""

# 检查二进制文件
if [ ! -f "./finta" ]; then
    echo "❌ 未找到 finta 二进制文件，正在构建..."
    go build -o finta cmd/finta/main.go
    if [ $? -ne 0 ]; then
        echo "❌ 构建失败"
        exit 1
    fi
    echo "✅ 构建成功"
fi

echo "1️⃣ 测试帮助信息"
echo "-------------------"
./finta chat --help | grep -E "(streaming|parallel)"
echo ""

echo "2️⃣ 测试工具列表"
echo "-------------------"
echo "期望看到 5 个工具: read, bash, write, glob, grep"
echo ""

echo "3️⃣ 可用的测试命令（需要 OPENAI_API_KEY）"
echo "-------------------"
echo ""
echo "基础模式:"
echo "  ./finta chat --model=deepseek-chat 'List all .go files in internal/'"
echo ""
echo "流式模式:"
echo "  ./finta chat --streaming --model=deepseek-chat 'Explain the project structure'"
echo ""
echo "顺序执行模式:"
echo "  ./finta chat --parallel=false --model=deepseek-chat 'Find and read go.mod'"
echo ""
echo "详细模式 + 流式:"
echo "  ./finta chat --verbose --streaming --model=deepseek-chat 'Count lines in all Go files'"
echo ""

echo "✅ Phase 2 测试脚本完成"
echo ""
echo "💡 提示: 设置 OPENAI_API_KEY 环境变量后可运行实际测试"
