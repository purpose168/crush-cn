// Package copilot 提供 GitHub Copilot OAuth 认证功能
// 该包实现了 GitHub Copilot 的设备码认证流程，包括设备码获取、令牌轮询和令牌刷新等功能
package copilot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/purpose168/crush-cn/internal/oauth"
)

const (
	// clientID 是 GitHub Copilot OAuth 应用的客户端标识符
	clientID = "Iv1.b507a08c87ecfe98"

	// deviceCodeURL 是 GitHub 设备码请求端点
	deviceCodeURL = "https://github.com/login/device/code"
	// accessTokenURL 是 GitHub 访问令牌请求端点
	accessTokenURL = "https://github.com/login/oauth/access_token"
	// copilotTokenURL 是 GitHub Copilot 内部令牌请求端点
	copilotTokenURL = "https://api.github.com/copilot_internal/v2/token"
)

// ErrNotAvailable 表示 GitHub Copilot 服务不可用的错误
var ErrNotAvailable = errors.New("GitHub Copilot 服务不可用")

// DeviceCode 表示 GitHub 设备码认证流程中的设备码信息
// 包含设备码、用户码、验证 URI 等关键信息
type DeviceCode struct {
	// DeviceCode 是用于轮询访问令牌的设备码
	DeviceCode string `json:"device_code"`
	// UserCode 是用户需要在浏览器中输入的验证码
	UserCode string `json:"user_code"`
	// VerificationURI 是用户需要访问的验证页面地址
	VerificationURI string `json:"verification_uri"`
	// ExpiresIn 表示设备码的有效期（秒）
	ExpiresIn int `json:"expires_in"`
	// Interval 表示轮询访问令牌的建议间隔时间（秒）
	Interval int `json:"interval"`
}

// RequestDeviceCode 向 GitHub 发起设备码认证流程
// 该函数通过 POST 请求向 GitHub 的设备码端点发送客户端 ID 和权限范围，
// 获取设备码和用户验证信息，用于后续的 OAuth 认证流程
//
// 参数:
//   - ctx: 上下文，用于控制请求超时和取消
//
// 返回值:
//   - *DeviceCode: 设备码信息，包含设备码、用户码、验证 URI 等
//   - error: 错误信息，如果请求失败则返回错误
func RequestDeviceCode(ctx context.Context) (*DeviceCode, error) {
	// 构造表单数据，设置客户端 ID 和权限范围
	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("scope", "read:user")

	// 创建 POST 请求，携带表单数据
	req, err := http.NewRequestWithContext(ctx, "POST", deviceCodeURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	// 设置请求头，指定接受 JSON 格式的响应
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", userAgent)

	// 发送 HTTP 请求，设置 30 秒超时
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 检查响应状态码，如果不是 200 OK 则返回错误
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("设备码请求失败: %s - %s", resp.Status, string(body))
	}

	// 解析 JSON 响应，提取设备码信息
	var dc DeviceCode
	if err := json.NewDecoder(resp.Body).Decode(&dc); err != nil {
		return nil, err
	}
	return &dc, nil
}

// PollForToken 轮询 GitHub 以获取用户授权后的访问令牌
// 该函数按照指定的间隔时间持续轮询 GitHub，直到用户完成授权或超时
// 支持处理授权待定和减速等状态，动态调整轮询间隔
//
// 参数:
//   - ctx: 上下文，用于控制轮询超时和取消
//   - dc: 设备码信息，用于轮询访问令牌
//
// 返回值:
//   - *oauth.Token: OAuth 令牌信息，包含访问令牌和刷新令牌
//   - error: 错误信息，如果轮询失败或超时则返回错误
func PollForToken(ctx context.Context, dc *DeviceCode) (*oauth.Token, error) {
	// 设置轮询间隔，最小为 5 秒
	interval := max(dc.Interval, 5)
	// 计算授权截止时间
	deadline := time.Now().Add(time.Duration(dc.ExpiresIn) * time.Second)
	// 创建定时器，按间隔轮询
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	// 持续轮询直到截止时间
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			// 上下文被取消，返回错误
			return nil, ctx.Err()
		case <-ticker.C:
			// 定时器触发，继续轮询
		}

		// 尝试获取访问令牌
		token, err := tryGetToken(ctx, dc.DeviceCode)
		if err == errPending {
			// 授权待定，继续轮询
			continue
		}
		if err == errSlowDown {
			// 收到减速指令，增加轮询间隔 5 秒
			interval += 5
			ticker.Reset(time.Duration(interval) * time.Second)
			continue
		}
		if err != nil {
			// 其他错误，直接返回
			return nil, err
		}
		// 成功获取令牌，返回结果
		return token, nil
	}

	// 授权超时，返回错误
	return nil, fmt.Errorf("授权超时")
}

var (
	// errPending 表示授权待定状态，用户尚未完成授权
	errPending = fmt.Errorf("授权待定")
	// errSlowDown 表示需要减速，增加轮询间隔
	errSlowDown = fmt.Errorf("减速")
)

// tryGetToken 尝试从 GitHub 获取访问令牌
// 该函数向 GitHub 的访问令牌端点发送请求，携带设备码信息，
// 根据响应状态判断授权是否完成，或是否需要调整轮询策略
//
// 参数:
//   - ctx: 上下文，用于控制请求超时和取消
//   - deviceCode: 设备码，用于标识授权请求
//
// 返回值:
//   - *oauth.Token: OAuth 令牌信息，如果授权成功
//   - error: 错误信息，包括授权待定、减速或其他错误
func tryGetToken(ctx context.Context, deviceCode string) (*oauth.Token, error) {
	// 构造表单数据，设置客户端 ID、设备码和授权类型
	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("device_code", deviceCode)
	data.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")

	// 创建 POST 请求，携带表单数据
	req, err := http.NewRequestWithContext(ctx, "POST", accessTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	// 设置请求头，指定接受 JSON 格式的响应
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", userAgent)

	// 发送 HTTP 请求，设置 30 秒超时
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 解析 JSON 响应，提取访问令牌和错误信息
	var result struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	// 根据错误字段判断授权状态
	switch result.Error {
	case "":
		// 无错误，检查是否获取到访问令牌
		if result.AccessToken == "" {
			return nil, errPending
		}
		// 成功获取访问令牌，继续获取 Copilot 令牌
		return getCopilotToken(ctx, result.AccessToken)
	case "authorization_pending":
		// 授权待定，用户尚未完成授权
		return nil, errPending
	case "slow_down":
		// 需要减速，增加轮询间隔
		return nil, errSlowDown
	default:
		// 其他错误，返回错误信息
		return nil, fmt.Errorf("授权失败: %s", result.Error)
	}
}

// getCopilotToken 使用 GitHub 访问令牌获取 Copilot 令牌
// 该函数向 GitHub Copilot 内部 API 发送请求，获取用于 Copilot 服务的专用令牌
//
// 参数:
//   - ctx: 上下文，用于控制请求超时和取消
//   - githubToken: GitHub 访问令牌，用于身份验证
//
// 返回值:
//   - *oauth.Token: OAuth 令牌信息，包含 Copilot 访问令牌和过期时间
//   - error: 错误信息，如果请求失败或 Copilot 服务不可用
func getCopilotToken(ctx context.Context, githubToken string) (*oauth.Token, error) {
	// 创建 GET 请求，访问 Copilot 令牌端点
	req, err := http.NewRequestWithContext(ctx, "GET", copilotTokenURL, nil)
	if err != nil {
		return nil, err
	}
	// 设置请求头，指定接受 JSON 格式的响应和授权令牌
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", githubToken))
	// 添加额外的请求头
	for k, v := range Headers() {
		req.Header.Set(k, v)
	}

	// 发送 HTTP 请求，设置 30 秒超时
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 检查响应状态码
	if resp.StatusCode == http.StatusForbidden {
		// 403 禁止访问，表示 Copilot 服务不可用
		return nil, ErrNotAvailable
	}
	if resp.StatusCode != http.StatusOK {
		// 其他非 200 状态码，返回错误
		return nil, fmt.Errorf("Copilot 令牌请求失败: %s - %s", resp.Status, string(body))
	}

	// 解析 JSON 响应，提取 Copilot 令牌和过期时间
	var result struct {
		Token     string `json:"token"`
		ExpiresAt int64  `json:"expires_at"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	// 构造 OAuth 令牌对象
	copilotToken := &oauth.Token{
		AccessToken:  result.Token,      // Copilot 访问令牌
		RefreshToken: githubToken,       // GitHub 访问令牌，用于刷新
		ExpiresAt:    result.ExpiresAt,  // 令牌过期时间戳
	}
	// 计算并设置过期时长
	copilotToken.SetExpiresIn()

	return copilotToken, nil
}

// RefreshToken 使用 GitHub 令牌刷新 Copilot 令牌
// 该函数通过 GitHub 访问令牌重新获取 Copilot 专用令牌，
// 用于在令牌过期后更新访问凭证
//
// 参数:
//   - ctx: 上下文，用于控制请求超时和取消
//   - githubToken: GitHub 访问令牌，用于身份验证
//
// 返回值:
//   - *oauth.Token: OAuth 令牌信息，包含新的 Copilot 访问令牌
//   - error: 错误信息，如果刷新失败
func RefreshToken(ctx context.Context, githubToken string) (*oauth.Token, error) {
	return getCopilotToken(ctx, githubToken)
}
