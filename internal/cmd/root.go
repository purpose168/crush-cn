package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/fang"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/exp/charmtone"
	"github.com/charmbracelet/x/term"
	"github.com/purpose168/crush-cn/internal/app"
	"github.com/purpose168/crush-cn/internal/config"
	"github.com/purpose168/crush-cn/internal/db"
	"github.com/purpose168/crush-cn/internal/event"
	"github.com/purpose168/crush-cn/internal/projects"
	"github.com/purpose168/crush-cn/internal/ui/common"
	ui "github.com/purpose168/crush-cn/internal/ui/model"
	"github.com/purpose168/crush-cn/internal/version"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.PersistentFlags().StringP("cwd", "c", "", "当前工作目录")
	rootCmd.PersistentFlags().StringP("data-dir", "D", "", "自定义 crush 数据目录")
	rootCmd.PersistentFlags().BoolP("debug", "d", false, "调试")
	rootCmd.Flags().BoolP("help", "h", false, "帮助")
	rootCmd.Flags().BoolP("yolo", "y", false, "自动接受所有权限（危险模式）")

	rootCmd.AddCommand(
		runCmd,
		dirsCmd,
		projectsCmd,
		updateProvidersCmd,
		logsCmd,
		schemaCmd,
		loginCmd,
		statsCmd,
	)
}

var rootCmd = &cobra.Command{
	Use:   "crush",
	Short: "软件开发的 AI 助手",
	Long:  "软件开发和类似任务的 AI 助手，可直接访问终端",
	Example: `
# 在交互模式下运行
crush

# 启用调试日志运行
crush -d

# 在特定目录中启用调试日志运行
crush -d -c /path/to/project

# 使用自定义数据目录运行
crush -D /path/to/custom/.crush

# 打印版本
crush -v

# 运行单个非交互式提示
crush run "解释 Go 中 context 的使用"

# 在危险模式下运行（自动接受所有权限）
crush -y
  `,
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := setupAppWithProgressBar(cmd)
		if err != nil {
			return err
		}
		defer app.Shutdown()

		event.AppInitialized()

		// Set up the TUI.
		var env uv.Environ = os.Environ()

		com := common.DefaultCommon(app)
		model := ui.New(com)

		program := tea.NewProgram(
			model,
			tea.WithEnvironment(env),
			tea.WithContext(cmd.Context()),
			tea.WithFilter(ui.MouseEventFilter), // Filter mouse events based on focus state
		)
		go app.Subscribe(program)

		if _, err := program.Run(); err != nil {
			event.Error(err)
			slog.Error("TUI 运行错误", "error", err)
			return errors.New("Crush 崩溃了。如果启用了指标，我们已经收到了通知。如果您想报告它，请复制上面的堆栈跟踪并在 https://github.com/purpose168/crush-cn/issues/new?template=bug.yml 打开一个问题") //nolint:staticcheck
		}
		return nil
	},
}

var heartbit = lipgloss.NewStyle().Foreground(charmtone.Dolly).SetString(`
    ▄▄▄▄▄▄▄▄    ▄▄▄▄▄▄▄▄
  ███████████  ███████████
████████████████████████████
████████████████████████████
██████████▀██████▀██████████
██████████ ██████ ██████████
▀▀██████▄████▄▄████▄██████▀▀
  ████████████████████████
    ████████████████████
       ▀▀██████████▀▀
           ▀▀▀▀▀▀
`)

// copied from cobra:
const defaultVersionTemplate = `{{with .DisplayName}}{{printf "%s " .}}{{end}}{{printf "version %s" .Version}}
`

func Execute() {
	// 注意：非常 hacky 的实现：我们创建一个使用 STDOUT 的 colorprofile 写入器，然后让它
	// 转发到一个 bytes.Buffer，将彩色的 heartbit 写入其中，最后
	// 在版本模板中前置它。
	// 不幸的是，cobra 没有给我们提供一种设置函数来处理
	// 打印版本的方法，而且 PreRunE 在版本已经被处理后运行，所以那也不起作用。
	// 这是我能找到的唯一相对有效的方法。
	if term.IsTerminal(os.Stdout.Fd()) {
		var b bytes.Buffer
		w := colorprofile.NewWriter(os.Stdout, os.Environ())
		w.Forward = &b
		_, _ = w.WriteString(heartbit.String())
		rootCmd.SetVersionTemplate(b.String() + "\n" + defaultVersionTemplate)
	}
	if err := fang.Execute(
		context.Background(),
		rootCmd,
		fang.WithVersion(version.Version),
		fang.WithNotifySignal(os.Interrupt),
	); err != nil {
		os.Exit(1)
	}
}

// supportsProgressBar 尝试通过查看环境变量来确定当前终端是否支持进度条。
func supportsProgressBar() bool {
	if !term.IsTerminal(os.Stderr.Fd()) {
		return false
	}
	termProg := os.Getenv("TERM_PROGRAM")
	_, isWindowsTerminal := os.LookupEnv("WT_SESSION")

	return isWindowsTerminal || strings.Contains(strings.ToLower(termProg), "ghostty")
}

func setupAppWithProgressBar(cmd *cobra.Command) (*app.App, error) {
	app, err := setupApp(cmd)
	if err != nil {
		return nil, err
	}

	// 检查配置中是否启用了进度条（如果为 nil 则默认为 true）
	progressEnabled := app.Config().Options.Progress == nil || *app.Config().Options.Progress
	if progressEnabled && supportsProgressBar() {
		_, _ = fmt.Fprintf(os.Stderr, ansi.SetIndeterminateProgressBar)
		defer func() { _, _ = fmt.Fprintf(os.Stderr, ansi.ResetProgressBar) }()
	}

	return app, nil
}

// setupApp 处理交互式和非交互式模式的通用设置逻辑。
// 返回应用实例、配置、清理函数和任何错误。
func setupApp(cmd *cobra.Command) (*app.App, error) {
	debug, _ := cmd.Flags().GetBool("debug")
	yolo, _ := cmd.Flags().GetBool("yolo")
	dataDir, _ := cmd.Flags().GetString("data-dir")
	ctx := cmd.Context()

	cwd, err := ResolveCwd(cmd)
	if err != nil {
		return nil, err
	}

	cfg, err := config.Init(cwd, dataDir, debug)
	if err != nil {
		return nil, err
	}

	if cfg.Permissions == nil {
		cfg.Permissions = &config.Permissions{}
	}
	cfg.Permissions.SkipRequests = yolo

	if err := createDotCrushDir(cfg.Options.DataDirectory); err != nil {
		return nil, err
	}

	// 在集中式项目列表中注册此项目。
	if err := projects.Register(cwd, cfg.Options.DataDirectory); err != nil {
		slog.Warn("注册项目失败", "error", err)
		// 非致命错误：即使注册失败也继续执行
	}

	// 连接到数据库；这也会运行迁移。
	conn, err := db.Connect(ctx, cfg.Options.DataDirectory)
	if err != nil {
		return nil, err
	}

	appInstance, err := app.New(ctx, conn, cfg)
	if err != nil {
		slog.Error("创建应用实例失败", "error", err)
		return nil, err
	}

	if shouldEnableMetrics(cfg) {
		event.Init()
	}

	return appInstance, nil
}

func shouldEnableMetrics(cfg *config.Config) bool {
	if v, _ := strconv.ParseBool(os.Getenv("CRUSH_DISABLE_METRICS")); v {
		return false
	}
	if v, _ := strconv.ParseBool(os.Getenv("DO_NOT_TRACK")); v {
		return false
	}
	if cfg.Options.DisableMetrics {
		return false
	}
	return true
}

func MaybePrependStdin(prompt string) (string, error) {
	if term.IsTerminal(os.Stdin.Fd()) {
		return prompt, nil
	}
	fi, err := os.Stdin.Stat()
	if err != nil {
		return prompt, err
	}
	// 检查标准输入是否为命名管道（|）或常规文件（<）。
	if fi.Mode()&os.ModeNamedPipe == 0 && !fi.Mode().IsRegular() {
		return prompt, nil
	}
	bts, err := io.ReadAll(os.Stdin)
	if err != nil {
		return prompt, err
	}
	return string(bts) + "\n\n" + prompt, nil
}

func ResolveCwd(cmd *cobra.Command) (string, error) {
	cwd, _ := cmd.Flags().GetString("cwd")
	if cwd != "" {
		err := os.Chdir(cwd)
		if err != nil {
			return "", fmt.Errorf("failed to change directory: %v", err)
		}
		return cwd, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %v", err)
	}
	return cwd, nil
}

func createDotCrushDir(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create data directory: %q %w", dir, err)
	}

	gitIgnorePath := filepath.Join(dir, ".gitignore")
	if _, err := os.Stat(gitIgnorePath); os.IsNotExist(err) {
		if err := os.WriteFile(gitIgnorePath, []byte("*\n"), 0o644); err != nil {
			return fmt.Errorf("failed to create .gitignore file: %q %w", gitIgnorePath, err)
		}
	}

	return nil
}
