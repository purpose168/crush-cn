package dialog

import (
	"context"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/catwalk/pkg/catwalk"
	"github.com/purpose168/crush-cn/internal/config"
	"github.com/purpose168/crush-cn/internal/oauth/hyper"
	"github.com/purpose168/crush-cn/internal/ui/common"
)

func NewOAuthHyper(
	com *common.Common,
	isOnboarding bool,
	provider catwalk.Provider,
	model config.SelectedModel,
	modelType config.SelectedModelType,
) (*OAuth, tea.Cmd) {
	return newOAuth(com, isOnboarding, provider, model, modelType, &OAuthHyper{})
}

type OAuthHyper struct {
	cancelFunc func()
}

var _ OAuthProvider = (*OAuthHyper)(nil)

func (m *OAuthHyper) name() string {
	return "Hyper"
}

func (m *OAuthHyper) initiateAuth() tea.Msg {
	minimumWait := 750 * time.Millisecond
	startTime := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	authResp, err := hyper.InitiateDeviceAuth(ctx)

	ellapsed := time.Since(startTime)
	if ellapsed < minimumWait {
		time.Sleep(minimumWait - ellapsed)
	}

	if err != nil {
		return ActionOAuthErrored{fmt.Errorf("无法启动设备认证: %w", err)}
	}

	return ActionInitiateOAuth{
		DeviceCode:      authResp.DeviceCode,
		UserCode:        authResp.UserCode,
		ExpiresIn:       authResp.ExpiresIn,
		VerificationURL: authResp.VerificationURL,
	}
}

func (m *OAuthHyper) startPolling(deviceCode string, expiresIn int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		m.cancelFunc = cancel

		refreshToken, err := hyper.PollForToken(ctx, deviceCode, expiresIn)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return ActionOAuthErrored{err}
		}

		token, err := hyper.ExchangeToken(ctx, refreshToken)
		if err != nil {
			return ActionOAuthErrored{fmt.Errorf("令牌交换失败: %w", err)}
		}

		introspect, err := hyper.IntrospectToken(ctx, token.AccessToken)
		if err != nil {
			return ActionOAuthErrored{fmt.Errorf("令牌内省失败: %w", err)}
		}
		if !introspect.Active {
			return ActionOAuthErrored{fmt.Errorf("访问令牌未激活")}
		}

		return ActionCompleteOAuth{token}
	}
}

func (m *OAuthHyper) stopPolling() tea.Msg {
	if m.cancelFunc != nil {
		m.cancelFunc()
	}
	return nil
}
