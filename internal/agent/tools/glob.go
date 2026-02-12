package tools

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"charm.land/fantasy"
	"github.com/purpose168/crush-cn/internal/fsext"
)

const GlobToolName = "glob"

//go:embed glob.md
var globDescription []byte

type GlobParams struct {
	Pattern string `json:"pattern" description:"用于匹配文件的 glob 模式"`
	Path    string `json:"path,omitempty" description:"要搜索的目录。默认为当前工作目录。"`
}

type GlobResponseMetadata struct {
	NumberOfFiles int  `json:"number_of_files"`
	Truncated     bool `json:"truncated"`
}

func NewGlobTool(workingDir string) fantasy.AgentTool {
	return fantasy.NewAgentTool(
		GlobToolName,
		string(globDescription),
		func(ctx context.Context, params GlobParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if params.Pattern == "" {
				return fantasy.NewTextErrorResponse("需要提供模式"), nil
			}

			searchPath := params.Path
			if searchPath == "" {
				searchPath = workingDir
			}

			files, truncated, err := globFiles(ctx, params.Pattern, searchPath, 100)
			if err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("查找文件错误: %w", err)
			}

			var output string
			if len(files) == 0 {
				output = "未找到文件"
			} else {
				normalizeFilePaths(files)
				output = strings.Join(files, "\n")
				if truncated {
					output += "\n\n(结果已截断。考虑使用更具体的路径或模式。)"
				}
			}

			return fantasy.WithResponseMetadata(
				fantasy.NewTextResponse(output),
				GlobResponseMetadata{
					NumberOfFiles: len(files),
					Truncated:     truncated,
				},
			), nil
		})
}

func globFiles(ctx context.Context, pattern, searchPath string, limit int) ([]string, bool, error) {
	cmdRg := getRgCmd(ctx, pattern)
	if cmdRg != nil {
		cmdRg.Dir = searchPath
		matches, err := runRipgrep(cmdRg, searchPath, limit)
		if err == nil {
			return matches, len(matches) >= limit && limit > 0, nil
		}
		slog.Warn("Ripgrep 执行失败，回退到 doublestar", "error", err)
	}

	return fsext.GlobWithDoubleStar(pattern, searchPath, limit)
}

func runRipgrep(cmd *exec.Cmd, searchRoot string, limit int) ([]string, error) {
	out, err := cmd.CombinedOutput()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 1 {
			return nil, nil
		}
		return nil, fmt.Errorf("ripgrep: %w\n%s", err, out)
	}

	var matches []string
	for p := range bytes.SplitSeq(out, []byte{0}) {
		if len(p) == 0 {
			continue
		}
		absPath := string(p)
		if !filepath.IsAbs(absPath) {
			absPath = filepath.Join(searchRoot, absPath)
		}
		if fsext.SkipHidden(absPath) {
			continue
		}
		matches = append(matches, absPath)
	}

	sort.SliceStable(matches, func(i, j int) bool {
		return len(matches[i]) < len(matches[j])
	})

	if limit > 0 && len(matches) > limit {
		matches = matches[:limit]
	}
	return matches, nil
}

func normalizeFilePaths(paths []string) {
	for i, p := range paths {
		paths[i] = filepath.ToSlash(p)
	}
}
