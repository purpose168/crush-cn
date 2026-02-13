package fsext

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/charlievieth/fastwalk"
	"github.com/purpose168/crush-cn/internal/csync"
	"github.com/purpose168/crush-cn/internal/home"
)

type FileInfo struct {
	Path    string
	ModTime time.Time
}

func SkipHidden(path string) bool {
	// 检查隐藏文件（以点开头）
	base := filepath.Base(path)
	if base != "." && strings.HasPrefix(base, ".") {
		return true
	}

	commonIgnoredDirs := map[string]bool{
		".crush":           true,
		"node_modules":     true,
		"vendor":           true,
		"dist":             true,
		"build":            true,
		"target":           true,
		".git":             true,
		".idea":            true,
		".vscode":          true,
		"__pycache__":      true,
		"bin":              true,
		"obj":              true,
		"out":              true,
		"coverage":         true,
		"logs":             true,
		"generated":        true,
		"bower_components": true,
		"jspm_packages":    true,
	}

	parts := strings.SplitSeq(path, string(os.PathSeparator))
	for part := range parts {
		if commonIgnoredDirs[part] {
			return true
		}
	}
	return false
}

// FastGlobWalker 提供支持 gitignore 的文件遍历功能（使用 fastwalk）
// 它采用类似 git 的分层忽略检查机制，从根目录到目标路径检查每个目录中的
// .gitignore/.crushignore 文件。
type FastGlobWalker struct {
	directoryLister *directoryLister
}

func NewFastGlobWalker(searchPath string) *FastGlobWalker {
	return &FastGlobWalker{
		directoryLister: NewDirectoryLister(searchPath),
	}
}

// ShouldSkip 根据分层的 gitignore、crushignore 和隐藏文件规则检查路径是否应该被跳过
func (w *FastGlobWalker) ShouldSkip(path string) bool {
	return w.directoryLister.shouldIgnore(path, nil)
}

func GlobWithDoubleStar(pattern, searchPath string, limit int) ([]string, bool, error) {
	// 将模式规范化为正斜杠（在 Windows 上），以便配置可以使用反斜杠
	pattern = filepath.ToSlash(pattern)

	walker := NewFastGlobWalker(searchPath)
	found := csync.NewSlice[FileInfo]()
	conf := fastwalk.Config{
		Follow:  true,
		ToSlash: fastwalk.DefaultToSlash(),
		Sort:    fastwalk.SortFilesFirst,
	}
	err := fastwalk.Walk(&conf, searchPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // 跳过无法访问的文件
		}

		if d.IsDir() {
			if walker.ShouldSkip(path) {
				return filepath.SkipDir
			}
		}

		if walker.ShouldSkip(path) {
			return nil
		}

		relPath, err := filepath.Rel(searchPath, path)
		if err != nil {
			relPath = path
		}

		// 将分隔符规范化为正斜杠
		relPath = filepath.ToSlash(relPath)

		// 检查路径是否匹配模式
		matched, err := doublestar.Match(pattern, relPath)
		if err != nil || !matched {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		found.Append(FileInfo{Path: path, ModTime: info.ModTime()})
		if limit > 0 && found.Len() >= limit*2 { // 注意：为什么乘以2？
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil && !errors.Is(err, filepath.SkipAll) {
		return nil, false, fmt.Errorf("fastwalk error: %w", err)
	}

	matches := slices.SortedFunc(found.Seq(), func(a, b FileInfo) int {
		return b.ModTime.Compare(a.ModTime)
	})
	matches, truncated := truncate(matches, limit)

	results := make([]string, len(matches))
	for i, m := range matches {
		results[i] = m.Path
	}
	return results, truncated || errors.Is(err, filepath.SkipAll), nil
}

// ShouldExcludeFile 根据常见模式和忽略规则检查文件是否应该被排除在处理之外
func ShouldExcludeFile(rootPath, filePath string) bool {
	return NewDirectoryLister(rootPath).
		shouldIgnore(filePath, nil)
}

func PrettyPath(path string) string {
	return home.Short(path)
}

func DirTrim(pwd string, lim int) string {
	var (
		out string
		sep = string(filepath.Separator)
	)
	dirs := strings.Split(pwd, sep)
	if lim > len(dirs)-1 || lim <= 0 {
		return pwd
	}
	for i := len(dirs) - 1; i > 0; i-- {
		out = sep + out
		if i == len(dirs)-1 {
			out = dirs[i]
		} else if i >= len(dirs)-lim {
			out = string(dirs[i][0]) + out
		} else {
			out = "..." + out
			break
		}
	}
	out = filepath.Join("~", out)
	return out
}

// PathOrPrefix 如果路径以 prefix 开头则返回 prefix，否则返回路径本身
func PathOrPrefix(path, prefix string) string {
	if HasPrefix(path, prefix) {
		return prefix
	}
	return path
}

// HasPrefix 检查给定路径是否以指定的 prefix 开头
// 使用 filepath.Rel 来判断路径是否在 prefix 内
func HasPrefix(path, prefix string) bool {
	rel, err := filepath.Rel(prefix, path)
	if err != nil {
		return false
	}
	// 如果路径在 prefix 内，Rel 不会返回以 ".." 开头的路径
	return !strings.HasPrefix(rel, "..")
}

// ToUnixLineEndings 将 Windows 换行符（CRLF）转换为 Unix 换行符（LF）
func ToUnixLineEndings(content string) (string, bool) {
	if strings.Contains(content, "\r\n") {
		return strings.ReplaceAll(content, "\r\n", "\n"), true
	}
	return content, false
}

// ToWindowsLineEndings 将 Unix 换行符（LF）转换为 Windows 换行符（CRLF）
func ToWindowsLineEndings(content string) (string, bool) {
	if !strings.Contains(content, "\r\n") {
		return strings.ReplaceAll(content, "\n", "\r\n"), true
	}
	return content, false
}

func truncate[T any](input []T, limit int) ([]T, bool) {
	if limit > 0 && len(input) > limit {
		return input[:limit], true
	}
	return input, false
}
