package oauth

import (
	"time"
)

// Token 表示一个 OAuth2 令牌。
// 该结构体包含访问令牌、刷新令牌以及过期时间等相关信息，
// 用于在 OAuth2 认证流程中管理和验证令牌的有效性。
type Token struct {
	AccessToken  string `json:"access_token"`  // 访问令牌，用于访问受保护资源的凭证
	RefreshToken string `json:"refresh_token"` // 刷新令牌，用于获取新的访问令牌
	ExpiresIn    int    `json:"expires_in"`    // 令牌有效期（秒），表示从颁发时起的有效时长
	ExpiresAt    int64  `json:"expires_at"`    // 令牌过期时间戳（Unix 时间戳），表示令牌的具体过期时刻
}

// SetExpiresAt 根据当前时间和 ExpiresIn 计算并设置 ExpiresAt 字段。
// 该方法将 ExpiresIn（有效期秒数）转换为具体的过期时间戳，
// 通过当前时间加上有效期来计算令牌的绝对过期时间。
//
// 计算公式：ExpiresAt = 当前时间 + ExpiresIn（秒）
//
// 使用示例：
//
//	token := &Token{ExpiresIn: 3600} // 令牌有效期为 1 小时
//	token.SetExpiresAt()              // 设置过期时间戳
func (t *Token) SetExpiresAt() {
	t.ExpiresAt = time.Now().Add(time.Duration(t.ExpiresIn) * time.Second).Unix()
}

// IsExpired 检查令牌是否已过期或即将过期（在其生命周期的 10% 内）。
// 该方法实现了提前过期机制，当令牌剩余有效期不足原有效期的 10% 时，
// 即认为令牌已过期，以避免在令牌即将过期时使用导致请求失败。
//
// 判断逻辑：
//   - 如果当前时间 >= (过期时间 - 有效期/10)，则认为令牌已过期
//   - 这样可以在令牌即将过期前提前刷新，确保服务的连续性
//
// 返回值：
//   - true: 令牌已过期或即将过期，需要刷新
//   - false: 令牌仍然有效，可以继续使用
//
// 使用示例：
//
//	if token.IsExpired() {
//	    // 需要刷新令牌
//	    refreshToken(token)
//	}
func (t *Token) IsExpired() bool {
	return time.Now().Unix() >= (t.ExpiresAt - int64(t.ExpiresIn)/10)
}

// SetExpiresIn 根据 ExpiresAt 字段计算并设置 ExpiresIn 字段。
// 该方法将绝对过期时间戳转换为相对有效期秒数，
// 通过计算从当前时间到过期时间的剩余秒数来确定令牌的剩余有效期。
//
// 计算公式：ExpiresIn = 过期时间 - 当前时间（秒）
//
// 使用场景：
//   - 当从存储中加载令牌时，需要根据过期时间戳重新计算剩余有效期
//   - 在令牌序列化或传输前，更新有效期字段
//
// 使用示例：
//
//	token := &Token{ExpiresAt: time.Now().Add(1800 * time.Second).Unix()}
//	token.SetExpiresIn() // 计算剩余有效期（约 1800 秒）
func (t *Token) SetExpiresIn() {
	t.ExpiresIn = int(time.Until(time.Unix(t.ExpiresAt, 0)).Seconds())
}
