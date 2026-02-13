package fsext

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/charlievieth/fastwalk"
	"github.com/purpose168/crush-cn/internal/csync"
	"github.com/purpose168/crush-cn/internal/home"
	ignore "github.com/sabhiram/go-gitignore"
)

// commonIgnorePatterns 包含常见的忽略文件和目录
var commonIgnorePatterns = sync.OnceValue(func() ignore.IgnoreParser {
	return ignore.CompileIgnoreLines(
		// 版本控制
		".git",
		".svn",
		".hg",
		".bzr",

		// IDE 和编辑器文件
		".vscode",
		".idea",
		"*.swp",
		"*.swo",
		"*~",
		".DS_Store",
		"Thumbs.db",

		// 构建产物和依赖
		"node_modules",
		"target",
		"build",
		"dist",
		"out",
		"bin",
		"obj",
		"*.o",
		"*.so",
		"*.dylib",
		"*.dll",
		"*.exe",

		// 日志和临时文件
		"*.log",
		"*.tmp",
		"*.temp",
		".cache",
		".tmp",

		// 特定语言相关
		"__pycache__",
		"*.pyc",
		"*.pyo",
		".pytest_cache",
		"vendor",
		"Cargo.lock",
		"package-lock.json",
		"yarn.lock",
		"pnpm-lock.yaml",

		// 操作系统生成的文件
		".Trash",
		".Spotlight-V100",
		".fseventsd",

		// Crush 相关
		".crush",

		// macOS 相关内容
		"OrbStack",
		".local",
		".share",
	)
})

var homeIgnore = sync.OnceValue(func() ignore.IgnoreParser {
	home := home.Dir()
	var lines []string
	for _, name := range []string{
		filepath.Join(home, ".gitignore"),
		filepath.Join(home, ".config", "git", "ignore"),
		filepath.Join(home, ".config", "crush", "ignore"),
	} {
		if bts, err := os.ReadFile(name); err == nil {
			lines = append(lines, strings.Split(string(bts), "\n")...)
		}
	}
	return ignore.CompileIgnoreLines(lines...)
})

type directoryLister struct {
	ignores  *csync.Map[string, ignore.IgnoreParser]
	rootPath string
}

func NewDirectoryLister(rootPath string) *directoryLister {
	dl := &directoryLister{
		rootPath: rootPath,
		ignores:  csync.NewMap[string, ignore.IgnoreParser](),
	}
	dl.getIgnore(rootPath)
	return dl
}

// git 检查顺序如下：
// - ./.gitignore, ../.gitignore 等，直到仓库根目录
// ~/.config/git/ignore
// ~/.gitignore
//
// 此函数将执行以下检查：
// - 给定的 ignorePatterns
// - [commonIgnorePatterns]
// - ./.gitignore, ../.gitignore 等，直到 dl.rootPath
// - ./.crushignore, ../.crushignore 等，直到 dl.rootPath
// ~/.config/git/ignore
// ~/.gitignore
// ~/.config/crush/ignore
func (dl *directoryLister) shouldIgnore(path string, ignorePatterns []string) bool {
	if len(ignorePatterns) > 0 {
		base := filepath.Base(path)
		for _, pattern := range ignorePatterns {
			if matched, err := filepath.Match(pattern, base); err == nil && matched {
				return true
			}
		}
	}

	// 不要对根目录本身应用 gitignore 规则
	// 在 gitignore 语义中，模式不适用于仓库根目录
	if path == dl.rootPath {
		return false
	}

	relPath, err := filepath.Rel(dl.rootPath, path)
	if err != nil {
		relPath = path
	}

	if commonIgnorePatterns().MatchesPath(relPath) {
		slog.Debug("Ignoring common pattern", "path", relPath)
		return true
	}

	parentDir := filepath.Dir(path)
	ignoreParser := dl.getIgnore(parentDir)
	if ignoreParser.MatchesPath(relPath) {
		slog.Debug("Ignoring dir pattern", "path", relPath, "dir", parentDir)
		return true
	}

	// 对于目录，也要检查带尾部斜杠的路径（gitignore 约定）
	if ignoreParser.MatchesPath(relPath + "/") {
		slog.Debug("Ignoring dir pattern with slash", "path", relPath+"/", "dir", parentDir)
		return true
	}

	if dl.checkParentIgnores(relPath) {
		return true
	}

	if homeIgnore().MatchesPath(relPath) {
		slog.Debug("Ignoring home dir pattern", "path", relPath)
		return true
	}

	return false
}

func (dl *directoryLister) checkParentIgnores(path string) bool {
	parent := filepath.Dir(filepath.Dir(path))
	for parent != "." && path != "." {
		if dl.getIgnore(parent).MatchesPath(path) {
			slog.Debug("Ignoring parent dir pattern", "path", path, "dir", parent)
			return true
		}
		if parent == dl.rootPath {
			break
		}
		parent = filepath.Dir(parent)
	}
	return false
}

func (dl *directoryLister) getIgnore(path string) ignore.IgnoreParser {
	return dl.ignores.GetOrSet(path, func() ignore.IgnoreParser {
		var lines []string
		for _, ign := range []string{".crushignore", ".gitignore"} {
			name := filepath.Join(path, ign)
			if content, err := os.ReadFile(name); err == nil {
				lines = append(lines, strings.Split(string(content), "\n")...)
			}
		}
		if len(lines) == 0 {
			// 返回一个空操作解析器以避免空指针检查
			return ignore.CompileIgnoreLines()
		}
		return ignore.CompileIgnoreLines(lines...)
	})
}

// ListDirectory 列出指定路径中的文件和目录
func ListDirectory(initialPath string, ignorePatterns []string, depth, limit int) ([]string, bool, error) {
	found := csync.NewSlice[string]()
	dl := NewDirectoryLister(initialPath)

	slog.Debug("Listing directory", "path", initialPath, "depth", depth, "limit", limit, "ignorePatterns", ignorePatterns)

	conf := fastwalk.Config{
		Follow:   true,
		ToSlash:  fastwalk.DefaultToSlash(),
		Sort:     fastwalk.SortDirsFirst,
		MaxDepth: depth,
	}

	err := fastwalk.Walk(&conf, initialPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // 跳过无权访问的文件
		}

		if dl.shouldIgnore(path, ignorePatterns) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if path != initialPath {
			if d.IsDir() {
				path = path + string(filepath.Separator)
			}
			found.Append(path)
		}

		if limit > 0 && found.Len() >= limit {
			return filepath.SkipAll
		}

		return nil
	})
	if err != nil && !errors.Is(err, filepath.SkipAll) {
		return nil, false, err
	}

	matches, truncated := truncate(slices.Collect(found.Seq()), limit)
	return matches, truncated || errors.Is(err, filepath.SkipAll), nil
}
