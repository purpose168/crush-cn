// Package hyper 提供处理 Hyper 设备流程认证的功能。
// 该包实现了 OAuth 2.0 设备授权流程（Device Authorization Flow），
// 允许用户在设备上完成身份验证，适用于输入受限的环境。
package hyper

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/purpose168/crush-cn/internal/agent/hyper"
	"github.com/purpose168/crush-cn/internal/event"
	"github.com/purpose168/crush-cn/internal/oauth"
)

// DeviceAuthResponse 包含设备授权端点的响应数据。
// 该结构体用于存储从 /device/auth 端点返回的设备授权信息，
// 包括设备码、用户码和验证 URL 等关键信息。
type DeviceAuthResponse struct {
	DeviceCode      string `json:"device_code"`       // 设备码，用于后续轮询获取令牌
	UserCode        string `json:"user_code"`         // 用户码，用户在验证页面输入的代码
	VerificationURL string `json:"verification_url"`  // 验证 URL，用户访问的验证页面地址
	ExpiresIn       int    `json:"expires_in"`        // 过期时间（秒），设备码的有效期
}

// TokenResponse 包含轮询端点的响应数据。
// 该结构体用于存储从 /device/token 端点返回的令牌信息，
// 包括刷新令牌、用户 ID 和组织信息等。
type TokenResponse struct {
	RefreshToken     string `json:"refresh_token,omitempty"`     // 刷新令牌，用于获取访问令牌
	UserID           string `json:"user_id"`                     // 用户 ID，唯一标识用户
	OrganizationID   string `json:"organization_id"`             // 组织 ID，用户所属组织的标识
	OrganizationName string `json:"organization_name"`           // 组织名称，用户所属组织的名称
	Error            string `json:"error,omitempty"`             // 错误代码，表示错误类型
	ErrorDescription string `json:"error_description,omitempty"` // 错误描述，详细说明错误原因
}

// InitiateDeviceAuth 调用 /device/auth 端点启动设备授权流程。
// 该函数向服务器发送设备授权请求，获取设备码和用户码，
// 用户需要在验证 URL 输入用户码以完成授权。
//
// 参数：
//   - ctx: 上下文，用于控制请求的生命周期和超时
//
// 返回值：
//   - *DeviceAuthResponse: 设备授权响应，包含设备码和验证信息
//   - error: 错误信息，如果请求失败则返回错误
func InitiateDeviceAuth(ctx context.Context) (*DeviceAuthResponse, error) {
	// 构建设备授权端点的完整 URL
	url := hyper.BaseURL() + "/device/auth"

	// 创建 HTTP POST 请求，包含设备名称的 JSON 数据
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, url,
		strings.NewReader(fmt.Sprintf(`{"device_name":%q}`, deviceName())),
	)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头，指定内容类型为 JSON
	req.Header.Set("Content-Type", "application/json")
	// 设置 User-Agent 标识客户端类型
	req.Header.Set("User-Agent", "crush")

	// 创建 HTTP 客户端，设置 30 秒超时
	client := &http.Client{Timeout: 30 * time.Second}
	// 执行 HTTP 请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("执行请求失败: %w", err)
	}
	// 确保在函数返回时关闭响应体
	defer resp.Body.Close()

	// 读取响应体，限制最大读取大小为 1MB 以防止内存耗尽
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查 HTTP 状态码，非 200 状态码表示请求失败
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("设备授权失败: 状态码 %d, 响应体 %q", resp.StatusCode, string(body))
	}

	// 解析 JSON 响应到 DeviceAuthResponse 结构体
	var authResp DeviceAuthResponse
	if err := json.Unmarshal(body, &authResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &authResp, nil
}

// deviceName 生成设备名称，用于标识发起授权请求的设备。
// 该函数尝试获取系统主机名，并将其包含在设备名称中，
// 如果无法获取主机名，则返回默认设备名称。
//
// 返回值：
//   - string: 设备名称，格式为 "Crush (主机名)" 或 "Crush"
func deviceName() string {
	// 尝试获取系统主机名
	if hostname, err := os.Hostname(); err == nil && hostname != "" {
		return "Crush (" + hostname + ")"
	}
	// 如果无法获取主机名，返回默认名称
	return "Crush"
}

// PollForToken 轮询 /device/token 端点直到授权完成。
// 该函数按照指定的轮询间隔持续检查授权状态，
// 直到用户完成授权或授权码过期。它处理各种错误状态，
// 包括授权待处理（authorization_pending）等情况。
//
// 参数：
//   - ctx: 上下文，用于控制轮询的生命周期
//   - deviceCode: 设备码，用于标识授权请求
//   - expiresIn: 过期时间（秒），设备码的有效期
//
// 返回值：
//   - string: 刷新令牌，用于后续获取访问令牌
//   - error: 错误信息，如果轮询失败或超时则返回错误
func PollForToken(ctx context.Context, deviceCode string, expiresIn int) (string, error) {
	// 创建带有超时的上下文，超时时间为设备码的有效期
	ctx, cancel := context.WithTimeout(ctx, time.Duration(expiresIn)*time.Second)
	defer cancel()

	// 设置轮询间隔为 5 秒
	d := 5 * time.Second
	// 创建定时器，定期检查授权状态
	ticker := time.NewTicker(d)
	defer ticker.Stop()

	// 开始轮询循环
	for {
		select {
		case <-ctx.Done():
			// 上下文被取消或超时，返回错误
			return "", ctx.Err()
		case <-ticker.C:
			// 定时器触发，执行一次轮询
			result, err := pollOnce(ctx, deviceCode)
			if err != nil {
				return "", err
			}
			// 如果获取到刷新令牌，表示授权成功
			if result.RefreshToken != "" {
				// 设置用户别名用于事件跟踪
				event.Alias(result.UserID)
				return result.RefreshToken, nil
			}
			// 根据错误类型处理不同情况
			switch result.Error {
			case "authorization_pending":
				// 授权待处理，继续轮询
				continue
			default:
				// 其他错误，返回错误描述
				return "", errors.New(result.ErrorDescription)
			}
		}
	}
}

// pollOnce 执行一次令牌轮询请求。
// 该函数向服务器发送一次 GET 请求，检查授权状态并获取令牌信息。
//
// 参数：
//   - ctx: 上下文，用于控制请求的生命周期
//   - deviceCode: 设备码，用于标识授权请求
//
// 返回值：
//   - TokenResponse: 令牌响应，包含令牌信息或错误信息
//   - error: 错误信息，如果请求失败则返回错误
func pollOnce(ctx context.Context, deviceCode string) (TokenResponse, error) {
	var result TokenResponse
	// 构建令牌查询端点的完整 URL
	url := fmt.Sprintf("%s/device/auth/%s", hyper.BaseURL(), deviceCode)
	// 创建 HTTP GET 请求
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return result, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头，指定内容类型为 JSON
	req.Header.Set("Content-Type", "application/json")
	// 设置 User-Agent 标识客户端类型
	req.Header.Set("User-Agent", "crush")

	// 创建 HTTP 客户端，设置 30 秒超时
	client := &http.Client{Timeout: 30 * time.Second}
	// 执行 HTTP 请求
	resp, err := client.Do(req)
	if err != nil {
		return result, fmt.Errorf("执行请求失败: %w", err)
	}
	// 确保在函数返回时关闭响应体
	defer resp.Body.Close()

	// 读取响应体，限制最大读取大小为 1MB 以防止内存耗尽
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return result, fmt.Errorf("读取响应失败: %w", err)
	}

	// 解析 JSON 响应到 TokenResponse 结构体
	if err := json.Unmarshal(body, &result); err != nil {
		return result, fmt.Errorf("解析响应失败: %w: %s", err, string(body))
	}

	// 检查 HTTP 状态码，非 200 状态码表示请求失败
	if resp.StatusCode != http.StatusOK {
		return result, fmt.Errorf("令牌请求失败: 状态码 %d 响应体 %q", resp.StatusCode, string(body))
	}

	return result, nil
}

// ExchangeToken 使用刷新令牌交换访问令牌。
// 该函数向服务器发送刷新令牌，获取可用于 API 访问的访问令牌。
//
// 参数：
//   - ctx: 上下文，用于控制请求的生命周期
//   - refreshToken: 刷新令牌，用于获取新的访问令牌
//
// 返回值：
//   - *oauth.Token: OAuth 令牌，包含访问令牌和相关信息
//   - error: 错误信息，如果交换失败则返回错误
func ExchangeToken(ctx context.Context, refreshToken string) (*oauth.Token, error) {
	// 构建请求体，包含刷新令牌
	reqBody := map[string]string{
		"refresh_token": refreshToken,
	}

	// 将请求体序列化为 JSON
	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 构建令牌交换端点的完整 URL
	url := hyper.BaseURL() + "/token/exchange"
	// 创建 HTTP POST 请求
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头，指定内容类型为 JSON
	req.Header.Set("Content-Type", "application/json")
	// 设置 User-Agent 标识客户端类型
	req.Header.Set("User-Agent", "crush")

	// 创建 HTTP 客户端，设置 30 秒超时
	client := &http.Client{Timeout: 30 * time.Second}
	// 执行 HTTP 请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("执行请求失败: %w", err)
	}
	// 确保在函数返回时关闭响应体
	defer resp.Body.Close()

	// 读取响应体，限制最大读取大小为 1MB 以防止内存耗尽
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查 HTTP 状态码，非 200 状态码表示请求失败
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("令牌交换失败: 状态码 %d 响应体 %q", resp.StatusCode, string(body))
	}

	// 解析 JSON 响应到 oauth.Token 结构体
	var token oauth.Token
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 设置令牌的过期时间
	token.SetExpiresAt()
	return &token, nil
}

// IntrospectTokenResponse 包含令牌自省端点的响应数据。
// 该结构体用于存储从 /token/introspect 端点返回的令牌验证信息，
// 包括令牌是否有效、用户标识和组织信息等。
type IntrospectTokenResponse struct {
	Active bool   `json:"active"`          // 令牌是否有效，true 表示令牌有效
	Sub    string `json:"sub,omitempty"`   // 主体标识，通常是用户 ID
	OrgID  string `json:"org_id,omitempty"` // 组织 ID，令牌所属组织
	Exp    int64  `json:"exp,omitempty"`   // 过期时间戳（Unix 时间戳）
	Iat    int64  `json:"iat,omitempty"`   // 签发时间戳（Unix 时间戳）
	Iss    string `json:"iss,omitempty"`   // 签发者，令牌的签发方
	Jti    string `json:"jti,omitempty"`   // JWT ID，令牌的唯一标识符
}

// IntrospectToken 使用自省端点验证访问令牌。
// 该函数实现了 OAuth 2.0 令牌自省（RFC 7662）标准，
// 用于检查令牌的有效性和获取令牌的元数据信息。
//
// 参数：
//   - ctx: 上下文，用于控制请求的生命周期
//   - accessToken: 访问令牌，需要验证的令牌
//
// 返回值：
//   - *IntrospectTokenResponse: 令牌自省响应，包含令牌验证结果
//   - error: 错误信息，如果验证失败则返回错误
func IntrospectToken(ctx context.Context, accessToken string) (*IntrospectTokenResponse, error) {
	// 构建请求体，包含需要验证的访问令牌
	reqBody := map[string]string{
		"token": accessToken,
	}

	// 将请求体序列化为 JSON
	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 构建令牌自省端点的完整 URL
	url := hyper.BaseURL() + "/token/introspect"
	// 创建 HTTP POST 请求
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头，指定内容类型为 JSON
	req.Header.Set("Content-Type", "application/json")
	// 设置 User-Agent 标识客户端类型
	req.Header.Set("User-Agent", "crush")

	// 创建 HTTP 客户端，设置 30 秒超时
	client := &http.Client{Timeout: 30 * time.Second}
	// 执行 HTTP 请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("执行请求失败: %w", err)
	}
	// 确保在函数返回时关闭响应体
	defer resp.Body.Close()

	// 读取响应体，限制最大读取大小为 1MB 以防止内存耗尽
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查 HTTP 状态码，非 200 状态码表示请求失败
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("令牌自省失败: 状态码 %d 响应体 %q", resp.StatusCode, string(body))
	}

	// 解析 JSON 响应到 IntrospectTokenResponse 结构体
	var result IntrospectTokenResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &result, nil
}
