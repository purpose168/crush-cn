package cmd

import (
	"encoding/json"
	"os"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/charmbracelet/x/term"
	"github.com/purpose168/crush-cn/internal/projects"
	"github.com/spf13/cobra"
)

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "列出项目目录",
	Long:  "列出已知存在 Crush 项目数据的目录",
	Example: `
# 以表格形式列出所有项目
crush projects

# 以 JSON 格式输出项目数据
crush projects --json
  `,
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOutput, _ := cmd.Flags().GetBool("json")

		projectList, err := projects.List()
		if err != nil {
			return err
		}

		if jsonOutput {
			output := struct {
				Projects []projects.Project `json:"projects"`
			}{Projects: projectList}

			data, err := json.Marshal(output)
			if err != nil {
				return err
			}
			cmd.Println(string(data))
			return nil
		}

		if len(projectList) == 0 {
			cmd.Println("尚未跟踪任何项目。")
			return nil
		}

		if term.IsTerminal(os.Stdout.Fd()) {
			// 我们在 TTY 中：美化输出
			t := table.New().
				Border(lipgloss.RoundedBorder()).
				StyleFunc(func(row, col int) lipgloss.Style {
					return lipgloss.NewStyle().Padding(0, 2)
				}).
				Headers("路径", "数据目录", "最后访问时间")

			for _, p := range projectList {
				t.Row(p.Path, p.DataDir, p.LastAccessed.Local().Format("2006-01-02 15:04"))
			}
			lipgloss.Println(t)
			return nil
		}

		// 非 TTY 环境：普通输出
		for _, p := range projectList {
			cmd.Printf("%s\t%s\t%s\n", p.Path, p.DataDir, p.LastAccessed.Format("2006-01-02T15:04:05Z07:00"))
		}
		return nil
	},
}

func init() {
	projectsCmd.Flags().Bool("json", false, "以 JSON 格式输出")
}
