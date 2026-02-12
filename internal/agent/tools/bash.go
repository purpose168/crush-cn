package tools

import (
	"bytes"
	"cmp"
	"context"
	_ "embed"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"charm.land/fantasy"
	"github.com/purpose168/crush-cn/internal/config"
	"github.com/purpose168/crush-cn/internal/permission"
	"github.com/purpose168/crush-cn/internal/shell"
)

type BashParams struct {
	Description     string `json:"description" description:"命令的简要描述，尽量保持在30个字符以内"`
	Command         string `json:"command" description:"要执行的命令"`
	WorkingDir      string `json:"working_dir,omitempty" description:"执行命令的工作目录（默认为当前目录）"`
	RunInBackground bool   `json:"run_in_background,omitempty" description:"设置为 true（布尔值）以在后台运行此命令。稍后使用 job_output 读取输出。"`
}

type BashPermissionsParams struct {
	Description     string `json:"description"`
	Command         string `json:"command"`
	WorkingDir      string `json:"working_dir"`
	RunInBackground bool   `json:"run_in_background"`
}

type BashResponseMetadata struct {
	StartTime        int64  `json:"start_time"`
	EndTime          int64  `json:"end_time"`
	Output           string `json:"output"`
	Description      string `json:"description"`
	WorkingDirectory string `json:"working_directory"`
	Background       bool   `json:"background,omitempty"`
	ShellID          string `json:"shell_id,omitempty"`
}

const (
	BashToolName = "bash"

	AutoBackgroundThreshold = 1 * time.Minute // 执行时间超过此阈值的命令会自动成为后台作业
	MaxOutputLength         = 30000
	BashNoOutput            = "no output"
)

//go:embed bash.tpl
var bashDescriptionTmpl []byte

var bashDescriptionTpl = template.Must(
	template.New("bashDescription").
		Parse(string(bashDescriptionTmpl)),
)

type bashDescriptionData struct {
	BannedCommands  string
	MaxOutputLength int
	Attribution     config.Attribution
	ModelName       string
}

var bannedCommands = []string{
	// 网络/下载工具
	"alias",
	"aria2c",
	"axel",
	"chrome",
	"curl",
	"curlie",
	"firefox",
	"http-prompt",
	"httpie",
	"links",
	"lynx",
	"nc",
	"safari",
	"scp",
	"ssh",
	"telnet",
	"w3m",
	"wget",
	"xh",

	// 系统管理
	"doas",
	"su",
	"sudo",

	// 包管理器
	"apk",
	"apt",
	"apt-cache",
	"apt-get",
	"dnf",
	"dpkg",
	"emerge",
	"home-manager",
	"makepkg",
	"opkg",
	"pacman",
	"paru",
	"pkg",
	"pkg_add",
	"pkg_delete",
	"portage",
	"rpm",
	"yay",
	"yum",
	"zypper",

	// 系统修改
	"at",
	"batch",
	"chkconfig",
	"crontab",
	"fdisk",
	"mkfs",
	"mount",
	"parted",
	"service",
	"systemctl",
	"umount",

	// 网络配置
	"firewall-cmd",
	"ifconfig",
	"ip",
	"iptables",
	"netstat",
	"pfctl",
	"route",
	"ufw",
}

func bashDescription(attribution *config.Attribution, modelName string) string {
	bannedCommandsStr := strings.Join(bannedCommands, ", ")
	var out bytes.Buffer
	if err := bashDescriptionTpl.Execute(&out, bashDescriptionData{
		BannedCommands:  bannedCommandsStr,
		MaxOutputLength: MaxOutputLength,
		Attribution:     *attribution,
		ModelName:       modelName,
	}); err != nil {
		// 这应该永远不会发生
		panic("执行 bash 描述模板失败: " + err.Error())
	}
	return out.String()
}

func blockFuncs() []shell.BlockFunc {
	return []shell.BlockFunc{
		shell.CommandsBlocker(bannedCommands),

		// 系统包管理器
		shell.ArgumentsBlocker("apk", []string{"add"}, nil),
		shell.ArgumentsBlocker("apt", []string{"install"}, nil),
		shell.ArgumentsBlocker("apt-get", []string{"install"}, nil),
		shell.ArgumentsBlocker("dnf", []string{"install"}, nil),
		shell.ArgumentsBlocker("pacman", nil, []string{"-S"}),
		shell.ArgumentsBlocker("pkg", []string{"install"}, nil),
		shell.ArgumentsBlocker("yum", []string{"install"}, nil),
		shell.ArgumentsBlocker("zypper", []string{"install"}, nil),

		// 语言特定的包管理器
		shell.ArgumentsBlocker("brew", []string{"install"}, nil),
		shell.ArgumentsBlocker("cargo", []string{"install"}, nil),
		shell.ArgumentsBlocker("gem", []string{"install"}, nil),
		shell.ArgumentsBlocker("go", []string{"install"}, nil),
		shell.ArgumentsBlocker("npm", []string{"install"}, []string{"--global"}),
		shell.ArgumentsBlocker("npm", []string{"install"}, []string{"-g"}),
		shell.ArgumentsBlocker("pip", []string{"install"}, []string{"--user"}),
		shell.ArgumentsBlocker("pip3", []string{"install"}, []string{"--user"}),
		shell.ArgumentsBlocker("pnpm", []string{"add"}, []string{"--global"}),
		shell.ArgumentsBlocker("pnpm", []string{"add"}, []string{"-g"}),
		shell.ArgumentsBlocker("yarn", []string{"global", "add"}, nil),

		// `go test -exec` 可以运行任意命令
		shell.ArgumentsBlocker("go", []string{"test"}, []string{"-exec"}),
	}
}

func NewBashTool(permissions permission.Service, workingDir string, attribution *config.Attribution, modelName string) fantasy.AgentTool {
	return fantasy.NewAgentTool(
		BashToolName,
		string(bashDescription(attribution, modelName)),
		func(ctx context.Context, params BashParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if params.Command == "" {
				return fantasy.NewTextErrorResponse("缺少命令"), nil
			}

			// 确定工作目录
			execWorkingDir := cmp.Or(params.WorkingDir, workingDir)

			isSafeReadOnly := false
			cmdLower := strings.ToLower(params.Command)

			for _, safe := range safeCommands {
				if strings.HasPrefix(cmdLower, safe) {
					if len(cmdLower) == len(safe) || cmdLower[len(safe)] == ' ' || cmdLower[len(safe)] == '-' {
						isSafeReadOnly = true
						break
					}
				}
			}

			sessionID := GetSessionFromContext(ctx)
			if sessionID == "" {
				return fantasy.ToolResponse{}, fmt.Errorf("执行 shell 命令需要会话 ID")
			}
			if !isSafeReadOnly {
				p, err := permissions.Request(ctx,
					permission.CreatePermissionRequest{
						SessionID:   sessionID,
						Path:        execWorkingDir,
						ToolCallID:  call.ID,
						ToolName:    BashToolName,
						Action:      "execute",
						Description: fmt.Sprintf("执行命令: %s", params.Command),
						Params:      BashPermissionsParams(params),
					},
				)
				if err != nil {
					return fantasy.ToolResponse{}, err
				}
				if !p {
					return fantasy.ToolResponse{}, permission.ErrorPermissionDenied
				}
			}

			// 如果明确要求在后台运行，立即使用分离的上下文启动
			if params.RunInBackground {
				startTime := time.Now()
				bgManager := shell.GetBackgroundShellManager()
				bgManager.Cleanup()
				// 使用后台上下文，以便在工具返回后继续运行
				bgShell, err := bgManager.Start(context.Background(), execWorkingDir, blockFuncs(), params.Command, params.Description)
				if err != nil {
					return fantasy.ToolResponse{}, fmt.Errorf("启动后台 shell 错误: %w", err)
				}

				// 等待一小段时间以检测快速失败（被阻止的命令、语法错误等）
				time.Sleep(1 * time.Second)
				stdout, stderr, done, execErr := bgShell.GetOutput()

				if done {
					// 命令失败或很快完成
					bgManager.Remove(bgShell.ID)

					interrupted := shell.IsInterrupt(execErr)
					exitCode := shell.ExitCode(execErr)
					if exitCode == 0 && !interrupted && execErr != nil {
						return fantasy.ToolResponse{}, fmt.Errorf("[作业 %s] 执行命令错误: %w", bgShell.ID, execErr)
					}

					stdout = formatOutput(stdout, stderr, execErr)

					metadata := BashResponseMetadata{
						StartTime:        startTime.UnixMilli(),
						EndTime:          time.Now().UnixMilli(),
						Output:           stdout,
						Description:      params.Description,
						Background:       params.RunInBackground,
						WorkingDirectory: bgShell.WorkingDir,
					}
					if stdout == "" {
						return fantasy.WithResponseMetadata(fantasy.NewTextResponse(BashNoOutput), metadata), nil
					}
					stdout += fmt.Sprintf("\n\n<cwd>%s</cwd>", normalizeWorkingDir(bgShell.WorkingDir))
					return fantasy.WithResponseMetadata(fantasy.NewTextResponse(stdout), metadata), nil
				}

				// 快速失败检查后仍在运行 - 作为后台作业返回
				metadata := BashResponseMetadata{
					StartTime:        startTime.UnixMilli(),
					EndTime:          time.Now().UnixMilli(),
					Description:      params.Description,
					WorkingDirectory: bgShell.WorkingDir,
					Background:       true,
					ShellID:          bgShell.ID,
				}
				response := fmt.Sprintf("后台 shell 已启动，ID: %s\n\n使用 job_output 工具查看输出或使用 job_kill 终止。", bgShell.ID)
				return fantasy.WithResponseMetadata(fantasy.NewTextResponse(response), metadata), nil
			}

			// 启动具有自动后台支持的同步执行
			startTime := time.Now()

			// 使用分离的上下文启动，以便在移至后台时能够继续运行
			bgManager := shell.GetBackgroundShellManager()
			bgManager.Cleanup()
			bgShell, err := bgManager.Start(context.Background(), execWorkingDir, blockFuncs(), params.Command, params.Description)
			if err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("启动 shell 错误: %w", err)
			}

			// 等待完成、自动后台阈值或上下文取消
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()
			timeout := time.After(AutoBackgroundThreshold)

			var stdout, stderr string
			var done bool
			var execErr error

		waitLoop:
			for {
				select {
				case <-ticker.C:
					stdout, stderr, done, execErr = bgShell.GetOutput()
					if done {
						break waitLoop
					}
				case <-timeout:
					stdout, stderr, done, execErr = bgShell.GetOutput()
					break waitLoop
				case <-ctx.Done():
					// 传入的上下文在我们移至后台之前被取消
					// 终止 shell 并返回错误
					bgManager.Kill(bgShell.ID)
					return fantasy.ToolResponse{}, ctx.Err()
				}
			}

			if done {
				// 命令在阈值内完成 - 同步返回
				// 从后台管理器中移除，因为我们直接返回
				// 不要调用 Kill()，因为它会取消上下文并损坏退出代码
				bgManager.Remove(bgShell.ID)

				interrupted := shell.IsInterrupt(execErr)
				exitCode := shell.ExitCode(execErr)
				if exitCode == 0 && !interrupted && execErr != nil {
					return fantasy.ToolResponse{}, fmt.Errorf("[作业 %s] 执行命令错误: %w", bgShell.ID, execErr)
				}

				stdout = formatOutput(stdout, stderr, execErr)

				metadata := BashResponseMetadata{
					StartTime:        startTime.UnixMilli(),
					EndTime:          time.Now().UnixMilli(),
					Output:           stdout,
					Description:      params.Description,
					Background:       params.RunInBackground,
					WorkingDirectory: bgShell.WorkingDir,
				}
				if stdout == "" {
					return fantasy.WithResponseMetadata(fantasy.NewTextResponse(BashNoOutput), metadata), nil
				}
				stdout += fmt.Sprintf("\n\n<cwd>%s</cwd>", normalizeWorkingDir(bgShell.WorkingDir))
				return fantasy.WithResponseMetadata(fantasy.NewTextResponse(stdout), metadata), nil
			}

			// 仍在运行 - 保持为后台作业
			metadata := BashResponseMetadata{
				StartTime:        startTime.UnixMilli(),
				EndTime:          time.Now().UnixMilli(),
				Description:      params.Description,
				WorkingDirectory: bgShell.WorkingDir,
				Background:       true,
				ShellID:          bgShell.ID,
			}
			response := fmt.Sprintf("命令执行时间超出预期，已移至后台。\n\n后台 shell ID: %s\n\n使用 job_output 工具查看输出或使用 job_kill 终止。", bgShell.ID)
			return fantasy.WithResponseMetadata(fantasy.NewTextResponse(response), metadata), nil
		})
}

// formatOutput 格式化已完成命令的输出，包含错误处理
func formatOutput(stdout, stderr string, execErr error) string {
	interrupted := shell.IsInterrupt(execErr)
	exitCode := shell.ExitCode(execErr)

	stdout = truncateOutput(stdout)
	stderr = truncateOutput(stderr)

	errorMessage := stderr
	if errorMessage == "" && execErr != nil {
		errorMessage = execErr.Error()
	}

	if interrupted {
		if errorMessage != "" {
			errorMessage += "\n"
		}
		errorMessage += "命令在完成前被中止"
	} else if exitCode != 0 {
		if errorMessage != "" {
			errorMessage += "\n"
		}
		errorMessage += fmt.Sprintf("退出码 %d", exitCode)
	}

	hasBothOutputs := stdout != "" && stderr != ""

	if hasBothOutputs {
		stdout += "\n"
	}

	if errorMessage != "" {
		stdout += "\n" + errorMessage
	}

	return stdout
}

func truncateOutput(content string) string {
	if len(content) <= MaxOutputLength {
		return content
	}

	halfLength := MaxOutputLength / 2
	start := content[:halfLength]
	end := content[len(content)-halfLength:]

	truncatedLinesCount := countLines(content[halfLength : len(content)-halfLength])
	return fmt.Sprintf("%s\n\n... [%d 行被截断] ...\n\n%s", start, truncatedLinesCount, end)
}

func countLines(s string) int {
	if s == "" {
		return 0
	}
	return len(strings.Split(s, "\n"))
}

func normalizeWorkingDir(path string) string {
	if runtime.GOOS == "windows" {
		cwd, err := os.Getwd()
		if err != nil {
			cwd = "C:"
		}
		path = strings.ReplaceAll(path, filepath.VolumeName(cwd), "")
	}

	return filepath.ToSlash(path)
}
