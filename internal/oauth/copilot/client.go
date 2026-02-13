// Package copilot 提供 GitHub Copilot 集成功能。
// 本包实现了与 GitHub Copilot API 的通信客户端，包括自定义的 HTTP 传输层，
// 用于根据请求内容自动设置 X-Initiator 头部信息。
package copilot

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"

	"github.com/purpose168/crush-cn/internal/log"
)

// assistantRolePattern 用于匹配 JSON 中的助手角色标识
// 该正则表达式匹配格式："role": "assistant"，允许键值之间存在空白字符变化
var assistantRolePattern = regexp.MustCompile(`"role"\s*:\s*"assistant"`)

// NewClient 创建一个新的 HTTP 客户端，该客户端使用自定义传输层，
// 能够根据请求体中的消息历史自动添加 X-Initiator 头部。
//
// 参数：
//   - isSubAgent: 布尔值，指示当前是否为子代理模式
//   - debug: 布尔值，指示是否启用调试模式
//
// 返回：
//   - *http.Client: 配置了自定义传输层的 HTTP 客户端
//
// 功能说明：
//   - 自定义传输层会检查请求体中是否包含助手消息
//   - 根据消息历史自动设置 X-Initiator 头部为 "user" 或 "agent"
//   - 在子代理模式下，始终将 X-Initiator 设置为 "agent"
func NewClient(isSubAgent, debug bool) *http.Client {
	return &http.Client{
		Transport: &initiatorTransport{debug: debug, isSubAgent: isSubAgent},
	}
}

// initiatorTransport 自定义 HTTP 传输层结构体
// 实现了 http.RoundTripper 接口，用于拦截和修改 HTTP 请求
type initiatorTransport struct {
	debug      bool // 是否启用调试模式
	isSubAgent bool // 是否为子代理模式
}

// RoundTrip 实现 http.RoundTripper 接口，处理 HTTP 请求的往返过程
// 该方法会检查请求体中的消息历史，并据此设置 X-Initiator 头部
//
// 参数：
//   - req: HTTP 请求对象
//
// 返回：
//   - *http.Response: HTTP 响应对象
//   - error: 错误信息
//
// 工作流程：
//  1. 检查请求是否为空
//  2. 检查请求体是否存在
//  3. 读取并分析请求体内容
//  4. 根据消息历史设置 X-Initiator 头部
//  5. 发送请求并返回响应
func (t *initiatorTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// 定义常量：头部名称和可能的值
	const (
		xInitiatorHeader = "X-Initiator" // 自定义头部名称，用于标识请求发起者
		userInitiator    = "user"        // 用户发起者标识
		agentInitiator   = "agent"       // 代理发起者标识
	)

	// 检查请求是否为空
	if req == nil {
		return nil, fmt.Errorf("HTTP 请求为空")
	}

	// 检查请求体是否为空（http.NoBody 表示没有请求体）
	if req.Body == http.NoBody {
		// 没有请求体可供检查，默认设置为用户发起
		req.Header.Set(xInitiatorHeader, userInitiator)
		slog.Debug("将 X-Initiator 头部设置为 user（无请求体）")
		return t.roundTrip(req)
	}

	// 克隆请求以避免修改原始请求对象
	req = req.Clone(req.Context())

	// 读取原始请求体内容到字节数组，以便后续检查
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, fmt.Errorf("读取请求体失败: %w", err)
	}
	defer req.Body.Close()

	// 使用保留的字节数据恢复原始请求体
	req.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	// 使用正则表达式检查是否存在助手消息
	// 这种方法可以处理 JSON 中的空白字符变化，同时避免完整的 JSON 解析开销
	initiator := userInitiator
	if assistantRolePattern.Match(bodyBytes) || t.isSubAgent {
		// 如果找到助手消息或处于子代理模式，设置发起者为 agent
		slog.Debug("将 X-Initiator 头部设置为 agent（在历史记录中找到助手消息）")
		initiator = agentInitiator
	} else {
		// 未找到助手消息，设置发起者为 user
		slog.Debug("将 X-Initiator 头部设置为 user（未找到助手消息）")
	}
	req.Header.Set(xInitiatorHeader, initiator)

	return t.roundTrip(req)
}

// roundTrip 执行实际的 HTTP 请求往返
// 根据调试模式选择不同的传输层实现
//
// 参数：
//   - req: HTTP 请求对象
//
// 返回：
//   - *http.Response: HTTP 响应对象
//   - error: 错误信息
func (t *initiatorTransport) roundTrip(req *http.Request) (*http.Response, error) {
	if t.debug {
		// 调试模式：使用自定义 HTTP 客户端的传输层（可能包含日志记录等功能）
		return log.NewHTTPClient().Transport.RoundTrip(req)
	}
	// 正常模式：使用默认传输层
	return http.DefaultTransport.RoundTrip(req)
}
