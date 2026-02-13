// Package copilot 提供GitHub Copilot OAuth认证相关功能
// 本文件实现了从磁盘读取GitHub Copilot OAuth令牌的功能
package copilot

import (
	"encoding/json" // JSON编码/解码包
	"os"            // 操作系统功能包，提供文件操作等
	"path/filepath" // 文件路径操作包
	"runtime"       // 运行时包，提供操作系统相关信息
)

// RefreshTokenFromDisk 从磁盘读取GitHub Copilot的OAuth刷新令牌
// 返回值:
//   - string: OAuth令牌字符串，如果读取失败则为空字符串
//   - bool: 是否成功读取令牌，true表示成功，false表示失败
//
// 该函数会从GitHub Copilot的配置文件中读取已保存的OAuth令牌，
// 配置文件位置因操作系统而异（Windows使用LOCALAPPDATA环境变量，
// Linux/macOS使用HOME环境变量）
func RefreshTokenFromDisk() (string, bool) {
	// 读取令牌文件内容
	data, err := os.ReadFile(tokenFilePath())
	if err != nil {
		// 文件读取失败，返回空字符串和false表示失败
		return "", false
	}

	// 定义JSON内容的数据结构
	// 配置文件是一个映射，键为应用标识符，值为包含用户信息和令牌的结构体
	var content map[string]struct {
		User        string `json:"user"`        // GitHub用户名
		OAuthToken  string `json:"oauth_token"` // OAuth访问令牌
		GitHubAppID string `json:"githubAppId"` // GitHub应用ID
	}

	// 将JSON数据解析到content结构体中
	if err := json.Unmarshal(data, &content); err != nil {
		// JSON解析失败，返回空字符串和false表示失败
		return "", false
	}

	// 查找GitHub Copilot应用的配置项
	// "github.com:Iv1.b507a08c87ecfe98" 是GitHub Copilot应用的客户端ID
	if app, ok := content["github.com:Iv1.b507a08c87ecfe98"]; ok {
		// 找到配置项，返回OAuth令牌和true表示成功
		return app.OAuthToken, true
	}

	// 未找到GitHub Copilot的配置项，返回空字符串和false表示失败
	return "", false
}

// tokenFilePath 返回GitHub Copilot配置文件的完整路径
// 返回值:
//   - string: 配置文件的完整路径
//
// 根据不同的操作系统返回不同的配置文件路径：
//   - Windows: %LOCALAPPDATA%\github-copilot\apps.json
//   - Linux/macOS: ~/.config/github-copilot/apps.json
func tokenFilePath() string {
	// 根据操作系统类型选择配置文件路径
	switch runtime.GOOS {
	case "windows":
		// Windows系统使用LOCALAPPDATA环境变量
		return filepath.Join(os.Getenv("LOCALAPPDATA"), "github-copilot/apps.json")
	default:
		// Linux和macOS系统使用HOME环境变量
		return filepath.Join(os.Getenv("HOME"), ".config/github-copilot/apps.json")
	}
}
