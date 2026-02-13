package config

import (
	"cmp"
	"context"
	"log/slog"
	"testing"

	"charm.land/catwalk/pkg/catwalk"
	"github.com/purpose168/crush-cn/internal/oauth"
	"github.com/purpose168/crush-cn/internal/oauth/copilot"
)

// ImportCopilot 从磁盘导入现有的 GitHub Copilot 令牌
// 该方法会检查配置中是否已存在 Copilot API 密钥或 OAuth 配置，
// 如果不存在，则尝试从磁盘读取并刷新令牌
// 返回值：*oauth.Token - 刷新后的令牌，bool - 是否成功导入
func (c *Config) ImportCopilot() (*oauth.Token, bool) {
	// 如果正在运行测试，则跳过导入
	if testing.Testing() {
		return nil, false
	}

	// 检查配置中是否已存在 Copilot API 密钥或 OAuth 配置
	// 如果已存在，则无需导入
	if c.HasConfigField("providers.copilot.api_key") || c.HasConfigField("providers.copilot.oauth") {
		return nil, false
	}

	// 从磁盘读取现有的刷新令牌
	diskToken, hasDiskToken := copilot.RefreshTokenFromDisk()
	if !hasDiskToken {
		return nil, false
	}

	// 在磁盘上找到现有的 GitHub Copilot 令牌，开始认证
	slog.Info("Found existing GitHub Copilot token on disk. Authenticating...")
	// 使用刷新令牌获取新的访问令牌
	token, err := copilot.RefreshToken(context.TODO(), diskToken)
	if err != nil {
		// 无法导入 GitHub Copilot 令牌
		slog.Error("Unable to import GitHub Copilot token", "error", err)
		return nil, false
	}

	// 将令牌设置为推理提供者的 API 密钥
	if err := c.SetProviderAPIKey(string(catwalk.InferenceProviderCopilot), token); err != nil {
		return token, false
	}

	// 将令牌保存到配置文件中
	// 尝试设置 API 密钥字段或 OAuth 字段（优先使用 API 密钥）
	if err := cmp.Or(
		c.SetConfigField("providers.copilot.api_key", token.AccessToken),
		c.SetConfigField("providers.copilot.oauth", token),
	); err != nil {
		// 无法将 GitHub Copilot 令牌保存到磁盘
		slog.Error("Unable to save GitHub Copilot token to disk", "error", err)
	}

	// GitHub Copilot 成功导入
	slog.Info("GitHub Copilot successfully imported")
	return token, true
}
