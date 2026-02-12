package tools

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"charm.land/fantasy"
	"github.com/charmbracelet/x/powernap/pkg/lsp/protocol"
	"github.com/purpose168/crush-cn/internal/lsp"
)

type DiagnosticsParams struct {
	FilePath string `json:"file_path,omitempty" description:"要获取诊断信息的文件路径（留空获取整个项目的诊断信息）"`
}

const DiagnosticsToolName = "lsp_diagnostics"

//go:embed diagnostics.md
var diagnosticsDescription []byte

// NewDiagnosticsTool 创建一个新的诊断工具实例
// lspManager: LSP客户端管理器
func NewDiagnosticsTool(lspManager *lsp.Manager) fantasy.AgentTool {
	return fantasy.NewAgentTool(
		DiagnosticsToolName,
		string(diagnosticsDescription),
		func(ctx context.Context, params DiagnosticsParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if lspManager.Clients().Len() == 0 {
				return fantasy.NewTextErrorResponse("没有可用的LSP客户端"), nil
			}
			notifyLSPs(ctx, lspManager, params.FilePath)
			output := getDiagnostics(params.FilePath, lspManager)
			return fantasy.NewTextResponse(output), nil
		})
}

// notifyLSPs 通知LSP客户端更新诊断信息
// ctx: 上下文对象
// manager: LSP客户端管理器
// filepath: 文件路径
func notifyLSPs(
	ctx context.Context,
	manager *lsp.Manager,
	filepath string,
) {
	if filepath == "" {
		return
	}

	if manager == nil {
		return
	}

	manager.Start(ctx, filepath)

	for client := range manager.Clients().Seq() {
		if !client.HandlesFile(filepath) {
			continue
		}
		_ = client.OpenFileOnDemand(ctx, filepath)
		_ = client.NotifyChange(ctx, filepath)
		client.WaitForDiagnostics(ctx, 5*time.Second)
	}
}

// getDiagnostics 获取文件或项目的诊断信息
// filePath: 文件路径（留空获取整个项目的诊断信息）
// manager: LSP客户端管理器
// 返回格式化的诊断信息字符串
func getDiagnostics(filePath string, manager *lsp.Manager) string {
	if manager == nil {
		return ""
	}

	var fileDiagnostics []string
	var projectDiagnostics []string

	for lspName, client := range manager.Clients().Seq2() {
		for location, diags := range client.GetDiagnostics() {
			path, err := location.Path()
			if err != nil {
				slog.Error("无法将诊断位置URI转换为路径", "uri", location, "error", err)
				continue
			}
			isCurrentFile := path == filePath
			for _, diag := range diags {
				formattedDiag := formatDiagnostic(path, diag, lspName)
				if isCurrentFile {
					fileDiagnostics = append(fileDiagnostics, formattedDiag)
				} else {
					projectDiagnostics = append(projectDiagnostics, formattedDiag)
				}
			}
		}
	}

	sortDiagnostics(fileDiagnostics)
	sortDiagnostics(projectDiagnostics)

	var output strings.Builder
	writeDiagnostics(&output, "file_diagnostics", fileDiagnostics)
	writeDiagnostics(&output, "project_diagnostics", projectDiagnostics)

	if len(fileDiagnostics) > 0 || len(projectDiagnostics) > 0 {
		fileErrors := countSeverity(fileDiagnostics, "Error")
		fileWarnings := countSeverity(fileDiagnostics, "Warn")
		projectErrors := countSeverity(projectDiagnostics, "Error")
		projectWarnings := countSeverity(projectDiagnostics, "Warn")
		output.WriteString("\n<diagnostic_summary>\n")
		fmt.Fprintf(&output, "当前文件: %d 错误, %d 警告\n", fileErrors, fileWarnings)
		fmt.Fprintf(&output, "项目: %d 错误, %d 警告\n", projectErrors, projectWarnings)
		output.WriteString("</diagnostic_summary>\n")
	}

	out := output.String()
	slog.Debug("诊断信息", "output", out)
	return out
}

// writeDiagnostics 写入诊断信息到输出构建器
// output: 字符串构建器
// tag: 诊断信息标签
// in: 诊断信息列表
func writeDiagnostics(output *strings.Builder, tag string, in []string) {
	if len(in) == 0 {
		return
	}
	output.WriteString("\n<" + tag + ">\n")
	if len(in) > 10 {
		output.WriteString(strings.Join(in[:10], "\n"))
		fmt.Fprintf(output, "\n... 还有 %d 条诊断信息", len(in)-10)
	} else {
		output.WriteString(strings.Join(in, "\n"))
	}
	output.WriteString("\n</" + tag + ">\n")
}

// sortDiagnostics 对诊断信息进行排序
// in: 诊断信息列表
// 返回排序后的诊断信息列表
// 排序规则：错误优先，然后按字母顺序
func sortDiagnostics(in []string) []string {
	sort.Slice(in, func(i, j int) bool {
		iIsError := strings.HasPrefix(in[i], "Error")
		jIsError := strings.HasPrefix(in[j], "Error")
		if iIsError != jIsError {
			return iIsError // 错误优先
		}
		return in[i] < in[j] // 然后按字母顺序
	})
	return in
}

// formatDiagnostic 格式化诊断信息
// pth: 文件路径
// diagnostic: 诊断信息对象
// source: 诊断来源
// 返回格式化的诊断信息字符串
func formatDiagnostic(pth string, diagnostic protocol.Diagnostic, source string) string {
	severity := "Info"
	switch diagnostic.Severity {
	case protocol.SeverityError:
		severity = "Error"
	case protocol.SeverityWarning:
		severity = "Warn"
	case protocol.SeverityHint:
		severity = "Hint"
	}

	location := fmt.Sprintf("%s:%d:%d", pth, diagnostic.Range.Start.Line+1, diagnostic.Range.Start.Character+1)

	sourceInfo := source
	if diagnostic.Source != "" {
		sourceInfo += " " + diagnostic.Source
	}

	codeInfo := ""
	if diagnostic.Code != nil {
		codeInfo = fmt.Sprintf("[%v]", diagnostic.Code)
	}

	tagsInfo := ""
	if len(diagnostic.Tags) > 0 {
		var tags []string
		for _, tag := range diagnostic.Tags {
			switch tag {
			case protocol.Unnecessary:
				tags = append(tags, "unnecessary")
			case protocol.Deprecated:
				tags = append(tags, "deprecated")
			}
		}
		if len(tags) > 0 {
			tagsInfo = fmt.Sprintf(" (%s)", strings.Join(tags, ", "))
		}
	}

	return fmt.Sprintf("%s: %s [%s]%s%s %s",
		severity,
		location,
		sourceInfo,
		codeInfo,
		tagsInfo,
		diagnostic.Message)
}

// countSeverity 统计指定严重程度的诊断信息数量
// diagnostics: 诊断信息列表
// severity: 严重程度（如"Error"、"Warn"等）
// 返回指定严重程度的诊断信息数量
func countSeverity(diagnostics []string, severity string) int {
	count := 0
	for _, diag := range diagnostics {
		if strings.HasPrefix(diag, severity) {
			count++
		}
	}
	return count
}
