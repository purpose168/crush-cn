#!/bin/bash
# 为所有 issue 和 PR 运行标签自动分类工作流的脚本

# 获取所有状态的 issue 列表（最多 1000 个）
# 只获取 issue 编号，并格式化为纯数字列表
ISSUES=$(gh issue list --state=all --limit=1000 --json "number" -t '{{range .}}{{printf "%.0f\n" .number}}{{end}}')

# 获取所有状态的 PR 列表（最多 1000 个）
# 只获取 PR 编号，并格式化为纯数字列表
PRS=$(gh pr list --state=all --limit=1000 --json "number" -t '{{range .}}{{printf "%.0f\n" .number}}{{end}}')

# 遍历所有 issue，为每个 issue 触发标签分类工作流
for issue in $ISSUES; do
  echo "Dispatching labeler.yml for $issue"  # 输出正在处理的 issue 编号
  gh workflow run labeler.yml -f issue-number="$issue"  # 触发 labeler.yml 工作流并传递 issue 编号

done

# 遍历所有 PR，为每个 PR 触发标签分类工作流
for pr in $PRS; do
  echo "Dispatching labeler.yml for $pr"  # 输出正在处理的 PR 编号
  gh workflow run labeler.yml -f issue-number="$pr"  # 触发 labeler.yml 工作流并传递 PR 编号

done
