package cmd

import (
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"

	"charm.land/catwalk/pkg/catwalk"
	"charm.land/lipgloss/v2/tree"
	"github.com/mattn/go-isatty"
	"github.com/purpose168/crush-cn/internal/config"
	"github.com/spf13/cobra"
)

// modelsCmd 定义了 'models' 命令，用于列出所有可用的模型
var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "列出所有已配置提供商的可用模型",
	Long:  `列出所有已配置提供商的可用模型。显示提供商名称和模型 ID。`,
	Example: `# 列出所有可用模型
crush models

# 搜索模型
crush models gpt5`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 解析当前工作目录
		cwd, err := ResolveCwd(cmd)
		if err != nil {
			return err
		}

		// 获取数据目录和调试标志
		dataDir, _ := cmd.Flags().GetString("data-dir")
		debug, _ := cmd.Flags().GetBool("debug")

		// 初始化配置
		cfg, err := config.Init(cwd, dataDir, debug)
		if err != nil {
			return err
		}

		// 检查是否已配置提供商
		if !cfg.IsConfigured() {
			return fmt.Errorf("未配置提供商 - 请运行 'crush' 以交互方式设置提供商")
		}

		// 处理搜索术语
		term := strings.ToLower(strings.Join(args, " "))
		// 过滤函数，用于筛选匹配的提供商和模型
		filter := func(p config.ProviderConfig, m catwalk.Model) bool {
			for _, s := range []string{p.ID, p.Name, m.ID, m.Name} {
				if term == "" || strings.Contains(strings.ToLower(s), term) {
					return true
				}
			}
			return false
		}

		// 存储提供商 ID 和对应的模型列表
		var providerIDs []string
		providerModels := make(map[string][]string)

		// 遍历所有提供商
		for providerID, provider := range cfg.Providers.Seq2() {
			// 跳过禁用的提供商
			if provider.Disable {
				continue
			}
			var found bool
			// 遍历提供商的所有模型
			for _, model := range provider.Models {
				// 应用过滤条件
				if !filter(provider, model) {
					continue
				}
				// 添加匹配的模型
				providerModels[providerID] = append(providerModels[providerID], model.ID)
				found = true
			}
			// 如果没有找到匹配的模型，跳过该提供商
			if !found {
				continue
			}
			// 对模型列表进行排序
			slices.Sort(providerModels[providerID])
			// 添加提供商 ID
			providerIDs = append(providerIDs, providerID)
		}
		// 对提供商 ID 进行排序
		sort.Strings(providerIDs)

		// 检查是否找到任何提供商
		if len(providerIDs) == 0 && len(args) == 0 {
			return fmt.Errorf("未找到启用的提供商")
		}
		if len(providerIDs) == 0 {
			return fmt.Errorf("未找到匹配 %q 的启用提供商", term)
		}

		// 非终端输出模式
		if !isatty.IsTerminal(os.Stdout.Fd()) {
			for _, providerID := range providerIDs {
				for _, modelID := range providerModels[providerID] {
					fmt.Println(providerID + "/" + modelID)
				}
			}
			return nil
		}

		// 终端输出模式，使用树形结构
		t := tree.New()
		for _, providerID := range providerIDs {
			providerNode := tree.Root(providerID)
			for _, modelID := range providerModels[providerID] {
				providerNode.Child(modelID)
			}
			t.Child(providerNode)
		}

		// 打印树形结构
		cmd.Println(t)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(modelsCmd)
}
