package config

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/purpose168/crush-cn/internal/fsext"
)

// 初始化标志文件名
const (
	InitFlagFilename = "init"
)

// ProjectInitFlag 项目初始化标志结构体
type ProjectInitFlag struct {
	Initialized bool `json:"initialized"` // 是否已初始化
}

// Init 初始化配置，加载工作目录和数据目录的配置
// workingDir: 工作目录路径
// dataDir: 数据目录路径
// debug: 是否启用调试模式
// 返回: 配置对象和可能的错误
func Init(workingDir, dataDir string, debug bool) (*Config, error) {
	cfg, err := Load(workingDir, dataDir, debug)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

// ProjectNeedsInitialization 检查项目是否需要初始化
// cfg: 配置对象
// 返回: 是否需要初始化和可能的错误
func ProjectNeedsInitialization(cfg *Config) (bool, error) {
	if cfg == nil {
		return false, fmt.Errorf("配置未加载")
	}

	flagFilePath := filepath.Join(cfg.Options.DataDirectory, InitFlagFilename)

	_, err := os.Stat(flagFilePath)
	if err == nil {
		return false, nil
	}

	if !os.IsNotExist(err) {
		return false, fmt.Errorf("检查初始化标志文件失败: %w", err)
	}

	someContextFileExists, err := contextPathsExist(cfg.WorkingDir())
	if err != nil {
		return false, fmt.Errorf("检查上下文文件失败: %w", err)
	}
	if someContextFileExists {
		return false, nil
	}

	// 如果工作目录没有非忽略的文件，跳过初始化步骤
	empty, err := dirHasNoVisibleFiles(cfg.WorkingDir())
	if err != nil {
		return false, fmt.Errorf("检查目录是否为空失败: %w", err)
	}
	if empty {
		return false, nil
	}

	return true, nil
}

// contextPathsExist 检查目录中是否存在默认上下文路径文件
// dir: 要检查的目录路径
// 返回: 是否存在上下文文件和可能的错误
func contextPathsExist(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}

	// 创建小写文件名切片，用于使用 slices.Contains 进行查找
	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, strings.ToLower(entry.Name()))
		}
	}

	// 检查目录中是否存在任何默认上下文路径
	for _, path := range defaultContextPaths {
		// 从路径中提取文件名
		_, filename := filepath.Split(path)
		filename = strings.ToLower(filename)

		if slices.Contains(files, filename) {
			return true, nil
		}
	}

	return false, nil
}

// dirHasNoVisibleFiles 检查目录在应用忽略规则后是否没有文件/目录
// dir: 要检查的目录路径
// 返回: 目录是否为空和可能的错误
func dirHasNoVisibleFiles(dir string) (bool, error) {
	files, _, err := fsext.ListDirectory(dir, nil, 1, 1)
	if err != nil {
		return false, err
	}
	return len(files) == 0, nil
}

// MarkProjectInitialized 标记项目已初始化
// cfg: 配置对象
// 返回: 可能的错误
func MarkProjectInitialized(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("配置未加载")
	}
	flagFilePath := filepath.Join(cfg.Options.DataDirectory, InitFlagFilename)

	file, err := os.Create(flagFilePath)
	if err != nil {
		return fmt.Errorf("创建初始化标志文件失败: %w", err)
	}
	defer file.Close()

	return nil
}

// HasInitialDataConfig 检查是否有初始数据配置
// cfg: 配置对象
// 返回: 是否存在初始数据配置
func HasInitialDataConfig(cfg *Config) bool {
	if cfg == nil {
		return false
	}
	cfgPath := GlobalConfigData()
	if _, err := os.Stat(cfgPath); err != nil {
		return false
	}
	return cfg.IsConfigured()
}
