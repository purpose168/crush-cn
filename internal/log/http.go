package log

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// NewHTTPClient 创建一个带有请求/响应日志记录功能的HTTP客户端
// 当调试模式开启时，会自动记录所有HTTP请求和响应的详细信息
// 返回值: 配置了日志记录的HTTP客户端实例
func NewHTTPClient() *http.Client {
	return &http.Client{
		Transport: &HTTPRoundTripLogger{
			Transport: http.DefaultTransport,
		},
	}
}

// HTTPRoundTripLogger 是一个实现了 http.RoundTripper 接口的传输层
// 用于拦截并记录HTTP请求和响应的详细信息
type HTTPRoundTripLogger struct {
	Transport http.RoundTripper  // 底层传输层，用于实际执行HTTP请求
}

// RoundTrip 实现了 http.RoundTripper 接口，在请求前后添加日志记录
// 参数:
//   - req: 要发送的HTTP请求
// 返回值:
//   - *http.Response: HTTP响应
//   - error: 请求过程中发生的错误
func (h *HTTPRoundTripLogger) RoundTrip(req *http.Request) (*http.Response, error) {
	var err error
	var save io.ReadCloser
	// 复制请求体以便日志记录
	save, req.Body, err = drainBody(req.Body)
	if err != nil {
		slog.Error(
			"HTTP请求失败",
			"method", req.Method,
			"url", req.URL,
			"error", err,
		)
		return nil, err
	}

	// 如果启用了调试级别，记录请求详情
	if slog.Default().Enabled(req.Context(), slog.LevelDebug) {
		slog.Debug(
			"HTTP请求",
			"method", req.Method,
			"url", req.URL,
			"body", bodyToString(save),
		)
	}

	// 执行请求并计算耗时
	start := time.Now()
	resp, err := h.Transport.RoundTrip(req)
	duration := time.Since(start)
	if err != nil {
		slog.Error(
			"HTTP请求失败",
			"method", req.Method,
			"url", req.URL,
			"duration_ms", duration.Milliseconds(),
			"error", err,
		)
		return resp, err
	}

	// 复制响应体以便日志记录
	save, resp.Body, err = drainBody(resp.Body)
	// 如果启用了调试级别，记录响应详情
	if slog.Default().Enabled(req.Context(), slog.LevelDebug) {
		slog.Debug(
			"HTTP响应",
			"status_code", resp.StatusCode,
			"status", resp.Status,
			"headers", formatHeaders(resp.Header),
			"body", bodyToString(save),
			"content_length", resp.ContentLength,
			"duration_ms", duration.Milliseconds(),
			"error", err,
		)
	}
	return resp, err
}

// bodyToString 将HTTP请求/响应体转换为字符串
// 如果内容是JSON格式，会进行格式化输出
// 参数:
//   - body: HTTP请求或响应的body
// 返回值: 格式化后的字符串表示
func bodyToString(body io.ReadCloser) string {
	if body == nil {
		return ""
	}
	src, err := io.ReadAll(body)
	if err != nil {
		slog.Error("读取body失败", "error", err)
		return ""
	}
	var b bytes.Buffer
	// 尝试格式化JSON，如果不是JSON则直接返回原始内容
	if json.Indent(&b, bytes.TrimSpace(src), "", "  ") != nil {
		// 不是JSON格式，直接返回原始字符串
		return string(src)
	}
	return b.String()
}

// formatHeaders 格式化HTTP头部用于日志记录，过滤掉敏感信息
// 对于包含认证信息的头部（如Authorization、API-Key、Token等），会用[REDACTED]替换实际值
// 参数:
//   - headers: 原始HTTP头部
// 返回值: 过滤后的头部映射
func formatHeaders(headers http.Header) map[string][]string {
	filtered := make(map[string][]string)
	for key, values := range headers {
		lowerKey := strings.ToLower(key)
		// 过滤敏感头部信息，防止泄露认证凭据
		if strings.Contains(lowerKey, "authorization") ||
			strings.Contains(lowerKey, "api-key") ||
			strings.Contains(lowerKey, "token") ||
			strings.Contains(lowerKey, "secret") {
			filtered[key] = []string{"[已隐藏]"}
		} else {
			filtered[key] = values
		}
	}
	return filtered
}

// drainBody 复制HTTP body以便多次读取
// 由于HTTP body只能读取一次，此函数创建两个副本供不同用途使用
// 参数:
//   - b: 原始HTTP body
// 返回值:
//   - r1: body的第一个副本
//   - r2: body的第二个副本
//   - err: 读取过程中发生的错误
func drainBody(b io.ReadCloser) (r1, r2 io.ReadCloser, err error) {
	if b == nil || b == http.NoBody {
		return http.NoBody, http.NoBody, nil
	}
	var buf bytes.Buffer
	if _, err = buf.ReadFrom(b); err != nil {
		return nil, b, err
	}
	if err = b.Close(); err != nil {
		return nil, b, err
	}
	return io.NopCloser(&buf), io.NopCloser(bytes.NewReader(buf.Bytes())), nil
}
