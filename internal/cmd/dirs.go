package cmd

import (
	"os"
	"path/filepath"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/charmbracelet/x/term"
	"github.com/purpose168/crush-cn/internal/config"
	"github.com/spf13/cobra"
)

var dirsCmd = &cobra.Command{
	Use:   "dirs",
	Short: "打印 Crush 使用的目录",
	Long: `打印 Crush 存储配置和数据文件的目录。
这包括全局配置目录和数据目录。`,
	Example: `
# 打印所有目录
crush dirs

# 仅打印配置目录
crush dirs config

# 仅打印数据目录
crush dirs data
  `,
	Run: func(cmd *cobra.Command, args []string) {
		if term.IsTerminal(os.Stdout.Fd()) {
			// 我们在 TTY 中：美化输出。
			t := table.New().
				Border(lipgloss.RoundedBorder()).
				StyleFunc(func(row, col int) lipgloss.Style {
					return lipgloss.NewStyle().Padding(0, 2)
				}).
				Row("Config", filepath.Dir(config.GlobalConfig())).
				Row("Data", filepath.Dir(config.GlobalConfigData()))
			lipgloss.Println(t)
			return
		}
		// 不在 TTY 中。
		cmd.Println(filepath.Dir(config.GlobalConfig()))
		cmd.Println(filepath.Dir(config.GlobalConfigData()))
	},
}

var configDirCmd = &cobra.Command{
	Use:   "config",
	Short: "打印 Crush 使用的配置目录",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println(filepath.Dir(config.GlobalConfig()))
	},
}

var dataDirCmd = &cobra.Command{
	Use:   "data",
	Short: "打印 Crush 使用的数据目录",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println(filepath.Dir(config.GlobalConfigData()))
	},
}

func init() {
	dirsCmd.AddCommand(configDirCmd, dataDirCmd)
}
