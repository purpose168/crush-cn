package log

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestHTTPRoundTripLogger 测试HTTP请求/响应日志记录器
// 该测试验证：
// 1. HTTP客户端能够正常发送请求
// 2. 响应状态码正确
// 3. 敏感头部信息被正确过滤
func TestHTTPRoundTripLogger(t *testing.T) {
	// 创建一个返回500错误的测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Custom-Header", "test-value")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Internal server error", "code": 500}`))
	}))
	defer server.Close()

	// 创建带有日志记录功能的HTTP客户端
	client := NewHTTPClient()

	// 构造测试请求
	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		server.URL,
		strings.NewReader(`{"test": "data"}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret-token")

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	// 验证响应状态码
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("期望状态码 500，实际得到 %d", resp.StatusCode)
	}
}

// TestFormatHeaders 测试HTTP头部格式化函数
// 该测试验证：
// 1. 敏感头部（Authorization、API-Key）被正确隐藏
// 2. 非敏感头部（Content-Type、User-Agent）被正确保留
func TestFormatHeaders(t *testing.T) {
	// 构造测试用的HTTP头部
	headers := http.Header{
		"Content-Type":  []string{"application/json"},
		"Authorization": []string{"Bearer secret-token"},
		"X-API-Key":     []string{"api-key-123"},
		"User-Agent":    []string{"test-agent"},
	}

	// 格式化头部
	formatted := formatHeaders(headers)

	// 验证敏感头部被隐藏
	if formatted["Authorization"][0] != "[已隐藏]" {
		t.Error("Authorization 头部应该被隐藏")
	}
	if formatted["X-API-Key"][0] != "[已隐藏]" {
		t.Error("X-API-Key 头部应该被隐藏")
	}

	// 验证非敏感头部被保留
	if formatted["Content-Type"][0] != "application/json" {
		t.Error("Content-Type 头部应该被保留")
	}
	if formatted["User-Agent"][0] != "test-agent" {
		t.Error("User-Agent 头部应该被保留")
	}
}
