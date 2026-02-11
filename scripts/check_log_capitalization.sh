#!/bin/bash
# 检查 Go 代码中日志消息是否以大写字母开头的脚本

# 使用 grep 递归搜索所有 .go 文件中的 slog 日志调用
# 查找以小写字母开头的日志消息
if grep -rE 'slog\.(Error|Info|Warn|Debug|Fatal|Print|Println|Printf)\(["][a-z]' --include="*.go" . 2>/dev/null; then
  # 如果找到小写开头的日志消息，输出错误信息
  echo "❌ Log messages must start with a capital letter. Found lowercase logs above."
  exit 1  # 退出码 1 表示检查失败
fi

# 如果没有找到小写开头的日志消息，脚本会正常退出（退出码 0）
