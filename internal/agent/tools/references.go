package tools

import (
	"cmp"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"charm.land/fantasy"
	"github.com/charmbracelet/x/powernap/pkg/lsp/protocol"
	"github.com/purpose168/crush-cn/internal/lsp"
)

type ReferencesParams struct {
	Symbol string `json:"symbol" description:"要搜索的符号名称（例如，函数名、变量名、类型名）"`
	Path   string `json:"path,omitempty" description:"要搜索的目录。使用目录/文件来缩小符号搜索范围。默认为当前工作目录。"`
}

type referencesTool struct {
	lspManager *lsp.Manager
}

const ReferencesToolName = "lsp_references"

//go:embed references.md
var referencesDescription []byte

// NewReferencesTool 创建一个新的引用搜索工具实例
// lspManager: LSP客户端管理器
func NewReferencesTool(lspManager *lsp.Manager) fantasy.AgentTool {
	return fantasy.NewAgentTool(
		ReferencesToolName,
		string(referencesDescription),
		func(ctx context.Context, params ReferencesParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if params.Symbol == "" {
				return fantasy.NewTextErrorResponse("symbol是必需的"), nil
			}

			if lspManager.Clients().Len() == 0 {
				return fantasy.NewTextErrorResponse("没有可用的LSP客户端"), nil
			}

			workingDir := cmp.Or(params.Path, ".")

			matches, _, err := searchFiles(ctx, regexp.QuoteMeta(params.Symbol), workingDir, "", 100)
			if err != nil {
				return fantasy.NewTextErrorResponse(fmt.Sprintf("搜索符号失败: %s", err)), nil
			}

			if len(matches) == 0 {
				return fantasy.NewTextResponse(fmt.Sprintf("未找到符号 '%s'", params.Symbol)), nil
			}

			var allLocations []protocol.Location
			var allErrs error
			for _, match := range matches {
				locations, err := find(ctx, lspManager, params.Symbol, match)
				if err != nil {
					if strings.Contains(err.Error(), "no identifier found") {
						// grep可能匹配了注释、字符串值或其他不相关的内容
						continue
					}
					slog.Error("查找引用失败", "error", err, "symbol", params.Symbol, "path", match.path, "line", match.lineNum, "char", match.charNum)
					allErrs = errors.Join(allErrs, err)
					continue
				}
				allLocations = append(allLocations, locations...)
				// XXX: 我们应该在这里中断还是查找所有结果？
			}

			if len(allLocations) > 0 {
				output := formatReferences(cleanupLocations(allLocations))
				return fantasy.NewTextResponse(output), nil
			}

			if allErrs != nil {
				return fantasy.NewTextErrorResponse(allErrs.Error()), nil
			}
			return fantasy.NewTextResponse(fmt.Sprintf("未找到符号 '%s' 的引用", params.Symbol)), nil
		})
}

func (r *referencesTool) Name() string {
	return ReferencesToolName
}

// find 查找符号的引用位置
// ctx: 上下文对象
// lspManager: LSP客户端管理器
// symbol: 要查找的符号
// match: 符号匹配信息
// 返回引用位置列表
func find(ctx context.Context, lspManager *lsp.Manager, symbol string, match grepMatch) ([]protocol.Location, error) {
	absPath, err := filepath.Abs(match.path)
	if err != nil {
		return nil, fmt.Errorf("获取绝对路径失败: %s", err)
	}

	var client *lsp.Client
	for c := range lspManager.Clients().Seq() {
		if c.HandlesFile(absPath) {
			client = c
			break
		}
	}

	if client == nil {
		slog.Warn("没有LSP客户端可以处理", "path", match.path)
		return nil, nil
	}

	return client.FindReferences(
		ctx,
		absPath,
		match.lineNum,
		match.charNum+getSymbolOffset(symbol),
		true,
	)
}

// getSymbolOffset 返回限定符号中实际符号名称的字符偏移量
// 例如，"foo.Bar"中的"Bar"或"Class::method"中的"method"。
func getSymbolOffset(symbol string) int {
	// 检查 :: 分隔符（Rust、C++、Ruby 模块/类、PHP 静态方法）。
	if idx := strings.LastIndex(symbol, "::"); idx != -1 {
		return idx + 2
	}
	// 检查 . 分隔符（Go、Python、JavaScript、Java、C#、Ruby 方法）。
	if idx := strings.LastIndex(symbol, "."); idx != -1 {
		return idx + 1
	}
	// 检查 \ 分隔符（PHP 命名空间）。
	if idx := strings.LastIndex(symbol, "\\"); idx != -1 {
		return idx + 1
	}
	return 0
}

// cleanupLocations 清理和排序位置列表
// locations: 位置列表
// 返回清理和排序后的位置列表
func cleanupLocations(locations []protocol.Location) []protocol.Location {
	slices.SortFunc(locations, func(a, b protocol.Location) int {
		if a.URI != b.URI {
			return strings.Compare(string(a.URI), string(b.URI))
		}
		if a.Range.Start.Line != b.Range.Start.Line {
			return cmp.Compare(a.Range.Start.Line, b.Range.Start.Line)
		}
		return cmp.Compare(a.Range.Start.Character, b.Range.Start.Character)
	})
	return slices.CompactFunc(locations, func(a, b protocol.Location) bool {
		return a.URI == b.URI &&
			a.Range.Start.Line == b.Range.Start.Line &&
			a.Range.Start.Character == b.Range.Start.Character
	})
}

// groupByFilename 按文件名分组位置列表
// locations: 位置列表
// 返回按文件名分组的位置映射
func groupByFilename(locations []protocol.Location) map[string][]protocol.Location {
	files := make(map[string][]protocol.Location)
	for _, loc := range locations {
		path, err := loc.URI.Path()
		if err != nil {
			slog.Error("无法将位置URI转换为路径", "uri", loc.URI, "error", err)
			continue
		}
		files[path] = append(files[path], loc)
	}
	return files
}

// formatReferences 格式化引用位置
// locations: 位置列表
// 返回格式化的引用位置字符串
func formatReferences(locations []protocol.Location) string {
	fileRefs := groupByFilename(locations)
	files := slices.Collect(maps.Keys(fileRefs))
	sort.Strings(files)

	var output strings.Builder
	output.WriteString(fmt.Sprintf("在 %d 个文件中找到 %d 个引用:\n\n", len(files), len(locations)))

	for _, file := range files {
		refs := fileRefs[file]
		output.WriteString(fmt.Sprintf("%s (%d 个引用):\n", file, len(refs)))
		for _, ref := range refs {
			line := ref.Range.Start.Line + 1
			char := ref.Range.Start.Character + 1
			output.WriteString(fmt.Sprintf("  第 %d 行, 第 %d 列\n", line, char))
		}
		output.WriteString("\n")
	}

	return output.String()
}
