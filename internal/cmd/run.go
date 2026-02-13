package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"

	"charm.land/log/v2"
	"github.com/purpose168/crush-cn/internal/event"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run [prompt...]",
	Short: "运行单个非交互式提示",
	Long: `在非交互模式下运行单个提示并退出。
提示可以作为参数提供或从标准输入管道传输。`,
	Example: `
# 运行简单提示
crush run Explain the use of context in Go

# 从标准输入管道输入
curl https://charm.land | crush run "Summarize this website"

# 从文件读取
crush run "What is this code doing?" <<< prrr.go

# 在安静模式下运行（隐藏 spinner）
crush run --quiet "Generate a README for this project"

# 在详细模式下运行
crush run --verbose "Generate a README for this project"
  `,
	RunE: func(cmd *cobra.Command, args []string) error {
		quiet, _ := cmd.Flags().GetBool("quiet")
		verbose, _ := cmd.Flags().GetBool("verbose")
		largeModel, _ := cmd.Flags().GetString("model")
		smallModel, _ := cmd.Flags().GetString("small-model")

		// 在 SIGINT 或 SIGTERM 信号时取消。
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
		defer cancel()

		app, err := setupApp(cmd)
		if err != nil {
			return err
		}
		defer app.Shutdown()

		if !app.Config().IsConfigured() {
			return fmt.Errorf("未配置任何提供商 - 请运行 'crush' 以交互方式设置提供商")
		}

		if verbose {
			slog.SetDefault(slog.New(log.New(os.Stderr)))
		}

		prompt := strings.Join(args, " ")

		prompt, err = MaybePrependStdin(prompt)
		if err != nil {
			slog.Error("从标准输入读取失败", "error", err)
			return err
		}

		if prompt == "" {
			return fmt.Errorf("未提供提示")
		}

		event.SetNonInteractive(true)
		event.AppInitialized()

		return app.RunNonInteractive(ctx, os.Stdout, prompt, largeModel, smallModel, quiet || verbose)
	},
	PostRun: func(cmd *cobra.Command, args []string) {
		event.AppExited()
	},
}

func init() {
	runCmd.Flags().BoolP("quiet", "q", false, "隐藏 spinner")
	runCmd.Flags().BoolP("verbose", "v", false, "显示日志")
	runCmd.Flags().StringP("model", "m", "", "要使用的模型。接受 'model' 或 'provider/model' 以区分不同提供商中同名的模型")
	runCmd.Flags().String("small-model", "", "要使用的小模型。如果未提供，将使用提供商的默认小模型")
}
