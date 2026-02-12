package tools

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	"charm.land/fantasy"
	"github.com/purpose168/crush-cn/internal/shell"
)

const (
	JobOutputToolName = "job_output"
)

//go:embed job_output.md
var jobOutputDescription []byte

type JobOutputParams struct {
	ShellID string `json:"shell_id" description:"要从中检索输出的后台shell的ID"`
}

type JobOutputResponseMetadata struct {
	ShellID          string `json:"shell_id"`
	Command          string `json:"command"`
	Description      string `json:"description"`
	Done             bool   `json:"done"`
	WorkingDirectory string `json:"working_directory"`
}

// NewJobOutputTool 创建一个新的作业输出工具实例
func NewJobOutputTool() fantasy.AgentTool {
	return fantasy.NewAgentTool(
		JobOutputToolName,
		string(jobOutputDescription),
		func(ctx context.Context, params JobOutputParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if params.ShellID == "" {
				return fantasy.NewTextErrorResponse("缺少shell_id"), nil
			}

			bgManager := shell.GetBackgroundShellManager()
			bgShell, ok := bgManager.Get(params.ShellID)
			if !ok {
				return fantasy.NewTextErrorResponse(fmt.Sprintf("未找到后台shell: %s", params.ShellID)), nil
			}

			stdout, stderr, done, err := bgShell.GetOutput()

			var outputParts []string
			if stdout != "" {
				outputParts = append(outputParts, stdout)
			}
			if stderr != "" {
				outputParts = append(outputParts, stderr)
			}

			status := "运行中"
			if done {
				status = "已完成"
				if err != nil {
					exitCode := shell.ExitCode(err)
					if exitCode != 0 {
						outputParts = append(outputParts, fmt.Sprintf("退出码 %d", exitCode))
					}
				}
			}

			output := strings.Join(outputParts, "\n")

			metadata := JobOutputResponseMetadata{
				ShellID:          params.ShellID,
				Command:          bgShell.Command,
				Description:      bgShell.Description,
				Done:             done,
				WorkingDirectory: bgShell.WorkingDir,
			}

			if output == "" {
				output = BashNoOutput
			}

			result := fmt.Sprintf("状态: %s\n\n%s", status, output)
			return fantasy.WithResponseMetadata(fantasy.NewTextResponse(result), metadata), nil
		})
}
