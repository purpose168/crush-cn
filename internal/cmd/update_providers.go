package cmd

import (
	"fmt"
	"log/slog"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/exp/charmtone"
	"github.com/purpose168/crush-cn/internal/config"
	"github.com/spf13/cobra"
)

var updateProvidersSource string

var updateProvidersCmd = &cobra.Command{
	Use:   "update-providers [path-or-url]",
	Short: "更新提供者",
	Long:  `从指定的本地路径或远程URL更新提供者信息。`,
	Example: `
# 远程更新Catwalk提供者（默认）
crush update-providers

# 从自定义URL更新Catwalk提供者
crush update-providers https://example.com/providers.json

# 从本地文件更新Catwalk提供者
crush update-providers /path/to/local-providers.json

# 从嵌入式版本更新Catwalk提供者
crush update-providers embedded

# 更新Hyper提供者信息
crush update-providers --source=hyper

# 从自定义URL更新Hyper
crush update-providers --source=hyper https://hyper.example.com
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 注意(@andreynering)：我们希望在此处跳过向stdout输出日志。
		slog.SetDefault(slog.New(slog.DiscardHandler))

		var pathOrURL string
		if len(args) > 0 {
			pathOrURL = args[0]
		}

		var err error
		switch updateProvidersSource {
		case "catwalk":
			err = config.UpdateProviders(pathOrURL)
		case "hyper":
			err = config.UpdateHyper(pathOrURL)
		default:
			return fmt.Errorf("无效的源 %q，必须是 'catwalk' 或 'hyper'", updateProvidersSource)
		}

		if err != nil {
			return err
		}

		// 注意(@andreynering)：这种样式大致是从Fang的错误消息复制而来，适用于成功消息。
		headerStyle := lipgloss.NewStyle().
			Foreground(charmtone.Butter).
			Background(charmtone.Guac).
			Bold(true).
			Padding(0, 1).
			Margin(1).
			MarginLeft(2).
			SetString("SUCCESS")
		textStyle := lipgloss.NewStyle().
			MarginLeft(2).
			SetString(fmt.Sprintf("%s 提供者更新成功。", updateProvidersSource))

		fmt.Printf("%s\n%s\n\n", headerStyle.Render(), textStyle.Render())
		return nil
	},
}

func init() {
	updateProvidersCmd.Flags().StringVar(&updateProvidersSource, "source", "catwalk", "要更新的提供者源（catwalk 或 hyper）")
}
