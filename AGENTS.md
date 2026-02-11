# Crush 开发指南

## 构建/测试/代码检查命令

- **构建**：`go build .` 或 `go run .`
- **测试**：`task test` 或 `go test ./...`（运行单个测试：`go test ./internal/llm/prompt -run TestGetContextFromPaths`）
- **更新 Golden 文件**：`go test ./... -update`（当测试输出变化时重新生成 .golden 文件）
  - 更新特定包：`go test ./internal/tui/components/core -update`（在这种情况下，我们正在更新 "core"）
- **代码检查**：`task lint:fix`
- **格式化**：`task fmt`（执行 gofumpt -w .）
- **开发模式**：`task dev`（启用性能分析运行）

## 代码风格指南

- **导入**：使用 goimports 格式化，按标准库、外部包、内部包分组
- **格式化**：使用 gofumpt（比 gofmt 更严格），在 golangci-lint 中启用
- **命名**：标准 Go 约定 - 导出项使用 PascalCase，非导出项使用 camelCase
- **类型**：优先使用显式类型，使用类型别名提高清晰度（例如，`type AgentName string`）
- **错误处理**：显式返回错误，使用 `fmt.Errorf` 包装错误
- **上下文**：操作的第一个参数始终传递 context.Context
- **接口**：在消费包中定义接口，保持接口小而专注
- **结构体**：使用结构体嵌入进行组合，将相关字段分组
- **常量**：使用带类型的常量和 iota 表示枚举，在 const 块中分组
- **测试**：使用 testify 的 `require` 包，使用 `t.Parallel()` 进行并行测试，
  使用 `t.SetEnv()` 设置环境变量。当需要临时目录时，始终使用 `t.Tempdir()`。
  此目录不需要手动删除。
- **JSON 标签**：JSON 字段名使用 snake_case
- **文件权限**：文件权限使用八进制表示法（0o755, 0o644）
- **日志消息**：日志消息必须以大写字母开头（例如，"Failed to save session" 而不是 "failed to save session"）
  - 这由 `task lint:log` 强制执行，作为 `task lint` 的一部分运行
- **注释**：注释必须以句号结尾，除非注释位于行尾。

## 使用模拟提供者进行测试

当编写涉及提供者配置的测试时，使用模拟提供者来避免 API 调用：

```go
func TestYourFunction(t *testing.T) {
    // 启用测试用的模拟提供者
    originalUseMock := config.UseMockProviders
    config.UseMockProviders = true
    defer func() {
        config.UseMockProviders = originalUseMock
        config.ResetProviders()
    }()

    // 重置提供者以确保获取新的模拟数据
    config.ResetProviders()

    // 测试代码 - 提供者现在将返回模拟数据
    providers := config.Providers()
    // ... 测试逻辑
}
```

## 格式化

- 始终格式化你编写的任何 Go 代码。
  - 首先，尝试 `gofumpt -w .`。
  - 如果 `gofumpt` 不可用，使用 `goimports`。
  - 如果 `goimports` 不可用，使用 `gofmt`。
  - 你也可以使用 `task fmt` 在整个项目上运行 `gofumpt -w .`，
    只要 `gofumpt` 在 `PATH` 中。

## 注释

- 独立行的注释应以大写字母开头并以句号结尾。注释应在 78 列处换行。

## 提交

- 始终使用语义化提交（`fix:`, `feat:`, `chore:`, `refactor:`, `docs:`, `sec:` 等）。
- 尝试将提交消息保持在一行，不包括你的归因。仅在确实需要额外上下文时使用多行提交。

## 处理 TUI（用户界面）
在开始处理 TUI 相关工作之前，请先阅读 internal/ui/AGENTS.md 文件
