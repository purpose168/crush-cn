package cmd

import (
	"cmp"
	"context"
	"fmt"
	"os"
	"os/signal"

	"charm.land/lipgloss/v2"
	"github.com/atotto/clipboard"
	"github.com/pkg/browser"
	hyperp "github.com/purpose168/crush-cn/internal/agent/hyper"
	"github.com/purpose168/crush-cn/internal/config"
	"github.com/purpose168/crush-cn/internal/oauth"
	"github.com/purpose168/crush-cn/internal/oauth/copilot"
	"github.com/purpose168/crush-cn/internal/oauth/hyper"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Aliases: []string{"auth"},
	Use:     "login [platform]",
	Short:   "登录 Crush 到平台",
	Long: `登录 Crush 到指定平台。
平台应作为参数提供。
可用平台有：hyper、copilot。`,
	Example: `
# 认证 Charm Hyper
crush login

# 认证 GitHub Copilot
crush login copilot
  `,
	ValidArgs: []cobra.Completion{
		"hyper",
		"copilot",
		"github",
		"github-copilot",
	},
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := setupAppWithProgressBar(cmd)
		if err != nil {
			return err
		}
		defer app.Shutdown()

		provider := "hyper"
		if len(args) > 0 {
			provider = args[0]
		}
		switch provider {
		case "hyper":
			return loginHyper(app.Config())
		case "copilot", "github", "github-copilot":
			return loginCopilot(app.Config())
		default:
			return fmt.Errorf("unknown platform: %s", args[0])
		}
	},
}

func loginHyper(cfg *config.Config) error {
	if !hyperp.Enabled() {
		return fmt.Errorf("hyper not enabled")
	}
	ctx := getLoginContext()

	resp, err := hyper.InitiateDeviceAuth(ctx)
	if err != nil {
		return err
	}

	if clipboard.WriteAll(resp.UserCode) == nil {
		fmt.Println("以下代码应该已经复制到剪贴板:")
	} else {
		fmt.Println("复制以下代码:")
	}

	fmt.Println()
	fmt.Println(lipgloss.NewStyle().Bold(true).Render(resp.UserCode))
	fmt.Println()
	fmt.Println("按回车键打开此 URL，然后将代码粘贴到那里:")
	fmt.Println()
	fmt.Println(lipgloss.NewStyle().Hyperlink(resp.VerificationURL, "id=hyper").Render(resp.VerificationURL))
	fmt.Println()
	waitEnter()
	if err := browser.OpenURL(resp.VerificationURL); err != nil {
		fmt.Println("无法打开 URL。您需要手动在浏览器中打开该 URL。")
	}

	fmt.Println("正在交换授权码...")
	refreshToken, err := hyper.PollForToken(ctx, resp.DeviceCode, resp.ExpiresIn)
	if err != nil {
		return err
	}

	fmt.Println("正在使用刷新令牌交换访问令牌...")
	token, err := hyper.ExchangeToken(ctx, refreshToken)
	if err != nil {
		return err
	}

	fmt.Println("正在验证访问令牌...")
	introspect, err := hyper.IntrospectToken(ctx, token.AccessToken)
	if err != nil {
		return fmt.Errorf("令牌内省失败: %w", err)
	}
	if !introspect.Active {
		return fmt.Errorf("访问令牌未激活")
	}

	if err := cmp.Or(
		cfg.SetConfigField("providers.hyper.api_key", token.AccessToken),
		cfg.SetConfigField("providers.hyper.oauth", token),
	); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("您现在已通过 Hyper 认证!")
	return nil
}

func loginCopilot(cfg *config.Config) error {
	ctx := getLoginContext()

	if cfg.HasConfigField("providers.copilot.oauth") {
		fmt.Println("您已经登录到 GitHub Copilot。")
		return nil
	}

	diskToken, hasDiskToken := copilot.RefreshTokenFromDisk()
	var token *oauth.Token

	switch {
	case hasDiskToken:
		fmt.Println("在磁盘上找到现有的 GitHub Copilot 令牌。使用它进行认证...")

		t, err := copilot.RefreshToken(ctx, diskToken)
		if err != nil {
			return fmt.Errorf("无法从磁盘刷新令牌: %w", err)
		}
		token = t
	default:
		fmt.Println("正在向 GitHub 请求设备代码...")
		dc, err := copilot.RequestDeviceCode(ctx)
		if err != nil {
			return err
		}

		fmt.Println()
		fmt.Println("打开以下 URL 并按照说明使用 GitHub Copilot 进行认证:")
		fmt.Println()
		fmt.Println(lipgloss.NewStyle().Hyperlink(dc.VerificationURI, "id=copilot").Render(dc.VerificationURI))
		fmt.Println()
		fmt.Println("代码:", lipgloss.NewStyle().Bold(true).Render(dc.UserCode))
		fmt.Println()
		fmt.Println("等待授权...")

		t, err := copilot.PollForToken(ctx, dc)
		if err == copilot.ErrNotAvailable {
			fmt.Println()
			fmt.Println("此账户无法使用 GitHub Copilot。如需注册，请访问以下页面:")
			fmt.Println()
			fmt.Println(lipgloss.NewStyle().Hyperlink(copilot.SignupURL, "id=copilot-signup").Render(copilot.SignupURL))
			fmt.Println()
			fmt.Println("如果符合条件，您可以申请免费访问。有关更多信息，请参阅:")
			fmt.Println()
			fmt.Println(lipgloss.NewStyle().Hyperlink(copilot.FreeURL, "id=copilot-free").Render(copilot.FreeURL))
		}
		if err != nil {
			return err
		}
		token = t
	}

	if err := cmp.Or(
		cfg.SetConfigField("providers.copilot.api_key", token.AccessToken),
		cfg.SetConfigField("providers.copilot.oauth", token),
	); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("您现在已通过 GitHub Copilot 认证!")
	return nil
}

func getLoginContext() context.Context {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	go func() {
		<-ctx.Done()
		cancel()
		os.Exit(1)
	}()
	return ctx
}

func waitEnter() {
	_, _ = fmt.Scanln()
}
