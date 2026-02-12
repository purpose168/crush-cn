package tools

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"charm.land/fantasy"
	"github.com/purpose168/crush-cn/internal/fsext"
)

// regexCache 提供编译正则表达式模式的线程安全缓存
type regexCache struct {
	cache map[string]*regexp.Regexp
	mu    sync.RWMutex
}

// newRegexCache 创建一个新的正则表达式缓存
func newRegexCache() *regexCache {
	return &regexCache{
		cache: make(map[string]*regexp.Regexp),
	}
}

// get 从缓存中检索编译后的正则表达式，或编译并缓存它
func (rc *regexCache) get(pattern string) (*regexp.Regexp, error) {
	// 首先尝试从缓存获取（读锁）
	rc.mu.RLock()
	if regex, exists := rc.cache[pattern]; exists {
		rc.mu.RUnlock()
		return regex, nil
	}
	rc.mu.RUnlock()

	// 编译正则表达式（写锁）
	rc.mu.Lock()
	defer rc.mu.Unlock()

	// 再次检查，以防另一个 goroutine 在我们等待时编译了它
	if regex, exists := rc.cache[pattern]; exists {
		return regex, nil
	}

	// 编译并缓存正则表达式
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	rc.cache[pattern] = regex
	return regex, nil
}

// 全局正则表达式缓存实例
var (
	searchRegexCache = newRegexCache()
	globRegexCache   = newRegexCache()
	// 用于 glob 转换的预编译正则表达式（频繁使用）
	globBraceRegex = regexp.MustCompile(`\{([^}]+)\}`)
)

type GrepParams struct {
	Pattern     string `json:"pattern" description:"在文件内容中搜索的正则表达式模式"`
	Path        string `json:"path,omitempty" description:"要搜索的目录。默认为当前工作目录。"`
	Include     string `json:"include,omitempty" description:"要包含在搜索中的文件模式（例如 \"*.js\"，\"*.{ts,tsx}\"）"`
	LiteralText bool   `json:"literal_text,omitempty" description:"如果为 true，模式将被视为字面文本，特殊正则表达式字符会被转义。默认为 false。"`
}

type grepMatch struct {
	path     string
	modTime  time.Time
	lineNum  int
	charNum  int
	lineText string
}

type GrepResponseMetadata struct {
	NumberOfMatches int  `json:"number_of_matches"`
	Truncated       bool `json:"truncated"`
}

const (
	GrepToolName        = "grep"
	maxGrepContentWidth = 500
)

//go:embed grep.md
var grepDescription []byte

// escapeRegexPattern 转义特殊正则表达式字符，使其被视为字面字符
func escapeRegexPattern(pattern string) string {
	specialChars := []string{"\\", ".", "+", "*", "?", "(", ")", "[", "]", "{", "}", "^", "$", "|"}
	escaped := pattern

	for _, char := range specialChars {
		escaped = strings.ReplaceAll(escaped, char, "\\"+char)
	}

	return escaped
}

func NewGrepTool(workingDir string) fantasy.AgentTool {
	return fantasy.NewAgentTool(
		GrepToolName,
		string(grepDescription),
		func(ctx context.Context, params GrepParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if params.Pattern == "" {
				return fantasy.NewTextErrorResponse("需要提供模式"), nil
			}

			// 如果 literal_text 为 true，转义模式
			searchPattern := params.Pattern
			if params.LiteralText {
				searchPattern = escapeRegexPattern(params.Pattern)
			}

			searchPath := params.Path
			if searchPath == "" {
				searchPath = workingDir
			}

			matches, truncated, err := searchFiles(ctx, searchPattern, searchPath, params.Include, 100)
			if err != nil {
				return fantasy.NewTextErrorResponse(fmt.Sprintf("搜索文件错误: %v", err)), nil
			}

			var output strings.Builder
			if len(matches) == 0 {
				output.WriteString("未找到文件")
			} else {
				fmt.Fprintf(&output, "找到 %d 个匹配项\n", len(matches))

				currentFile := ""
				for _, match := range matches {
					if currentFile != match.path {
						if currentFile != "" {
							output.WriteString("\n")
						}
						currentFile = match.path
						fmt.Fprintf(&output, "%s:\n", filepath.ToSlash(match.path))
					}
					if match.lineNum > 0 {
						lineText := match.lineText
						if len(lineText) > maxGrepContentWidth {
							lineText = lineText[:maxGrepContentWidth] + "..."
						}
						if match.charNum > 0 {
							fmt.Fprintf(&output, "  第 %d 行，第 %d 字符: %s\n", match.lineNum, match.charNum, lineText)
						} else {
							fmt.Fprintf(&output, "  第 %d 行: %s\n", match.lineNum, lineText)
						}
					} else {
						fmt.Fprintf(&output, "  %s\n", match.path)
					}
				}

				if truncated {
					output.WriteString("\n(结果已截断。考虑使用更具体的路径或模式。)")
				}
			}

			return fantasy.WithResponseMetadata(
				fantasy.NewTextResponse(output.String()),
				GrepResponseMetadata{
					NumberOfMatches: len(matches),
					Truncated:       truncated,
				},
			), nil
		})
}

func searchFiles(ctx context.Context, pattern, rootPath, include string, limit int) ([]grepMatch, bool, error) {
	matches, err := searchWithRipgrep(ctx, pattern, rootPath, include)
	if err != nil {
		matches, err = searchFilesWithRegex(pattern, rootPath, include)
		if err != nil {
			return nil, false, err
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].modTime.After(matches[j].modTime)
	})

	truncated := len(matches) > limit
	if truncated {
		matches = matches[:limit]
	}

	return matches, truncated, nil
}

func searchWithRipgrep(ctx context.Context, pattern, path, include string) ([]grepMatch, error) {
	cmd := getRgSearchCmd(ctx, pattern, path, include)
	if cmd == nil {
		return nil, fmt.Errorf("在 $PATH 中未找到 ripgrep")
	}

	// 仅在忽略文件存在时添加
	for _, ignoreFile := range []string{".gitignore", ".crushignore"} {
		ignorePath := filepath.Join(path, ignoreFile)
		if _, err := os.Stat(ignorePath); err == nil {
			cmd.Args = append(cmd.Args, "--ignore-file", ignorePath)
		}
	}

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return []grepMatch{}, nil
		}
		return nil, err
	}

	var matches []grepMatch
	for line := range bytes.SplitSeq(bytes.TrimSpace(output), []byte{'\n'}) {
		if len(line) == 0 {
			continue
		}
		var match ripgrepMatch
		if err := json.Unmarshal(line, &match); err != nil {
			continue
		}
		if match.Type != "match" {
			continue
		}
		for _, m := range match.Data.Submatches {
			fi, err := os.Stat(match.Data.Path.Text)
			if err != nil {
				continue // 跳过无法访问的文件
			}
			matches = append(matches, grepMatch{
				path:     match.Data.Path.Text,
				modTime:  fi.ModTime(),
				lineNum:  match.Data.LineNumber,
				charNum:  m.Start + 1, // 确保从 1 开始
				lineText: strings.TrimSpace(match.Data.Lines.Text),
			})
			// 只获取每行的第一个匹配项
			break
		}
	}
	return matches, nil
}

type ripgrepMatch struct {
	Type string `json:"type"`
	Data struct {
		Path struct {
			Text string `json:"text"`
		} `json:"path"`
		Lines struct {
			Text string `json:"text"`
		} `json:"lines"`
		LineNumber int `json:"line_number"`
		Submatches []struct {
			Start int `json:"start"`
		} `json:"submatches"`
	} `json:"data"`
}

func searchFilesWithRegex(pattern, rootPath, include string) ([]grepMatch, error) {
	matches := []grepMatch{}

	// 使用缓存的正则表达式编译
	regex, err := searchRegexCache.get(pattern)
	if err != nil {
		return nil, fmt.Errorf("无效的正则表达式模式: %w", err)
	}

	var includePattern *regexp.Regexp
	if include != "" {
		regexPattern := globToRegex(include)
		includePattern, err = globRegexCache.get(regexPattern)
		if err != nil {
			return nil, fmt.Errorf("无效的包含模式: %w", err)
		}
	}

	// 创建支持 gitignore 和 crushignore 的遍历器
	walker := fsext.NewFastGlobWalker(rootPath)

	err = filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 跳过错误
		}

		if info.IsDir() {
			// 检查是否应该跳过目录
			if walker.ShouldSkip(path) {
				return filepath.SkipDir
			}
			return nil // 继续进入目录
		}

		// 对文件使用遍历器的 shouldSkip 方法
		if walker.ShouldSkip(path) {
			return nil
		}

		// 跳过隐藏文件（以点开头）以匹配 ripgrep 的默认行为
		base := filepath.Base(path)
		if base != "." && strings.HasPrefix(base, ".") {
			return nil
		}

		if includePattern != nil && !includePattern.MatchString(path) {
			return nil
		}

		match, lineNum, charNum, lineText, err := fileContainsPattern(path, regex)
		if err != nil {
			return nil // 跳过无法读取的文件
		}

		if match {
			matches = append(matches, grepMatch{
				path:     path,
				modTime:  info.ModTime(),
				lineNum:  lineNum,
				charNum:  charNum,
				lineText: lineText,
			})

			if len(matches) >= 200 {
				return filepath.SkipAll
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return matches, nil
}

func fileContainsPattern(filePath string, pattern *regexp.Regexp) (bool, int, int, string, error) {
	// 只搜索文本文件
	if !isTextFile(filePath) {
		return false, 0, 0, "", nil
	}

	file, err := os.Open(filePath)
	if err != nil {
		return false, 0, 0, "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if loc := pattern.FindStringIndex(line); loc != nil {
			charNum := loc[0] + 1
			return true, lineNum, charNum, line, nil
		}
	}

	return false, 0, 0, "", scanner.Err()
}

// isTextFile 通过检查文件的 MIME 类型来判断它是否为文本文件
func isTextFile(filePath string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer file.Close()

	// 读取前 512 字节用于 MIME 类型检测
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return false
	}

	// 检测内容类型
	contentType := http.DetectContentType(buffer[:n])

	// 检查是否为文本 MIME 类型
	return strings.HasPrefix(contentType, "text/") ||
		contentType == "application/json" ||
		contentType == "application/xml" ||
		contentType == "application/javascript" ||
		contentType == "application/x-sh"
}

func globToRegex(glob string) string {
	regexPattern := strings.ReplaceAll(glob, ".", "\\.")
	regexPattern = strings.ReplaceAll(regexPattern, "*", ".*")
	regexPattern = strings.ReplaceAll(regexPattern, "?", ".")

	// 使用预编译的正则表达式而不是每次都编译
	regexPattern = globBraceRegex.ReplaceAllStringFunc(regexPattern, func(match string) string {
		inner := match[1 : len(match)-1]
		return "(" + strings.ReplaceAll(inner, ",", "|") + ")"
	})

	return regexPattern
}
