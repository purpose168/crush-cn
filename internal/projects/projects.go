// Package projects 提供项目目录跟踪和管理功能
// 该包负责维护一个项目列表，记录每个项目的工作目录、数据目录和最后访问时间
package projects

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/purpose168/crush-cn/internal/config"
)

// projectsFileName 定义项目列表文件的名称常量
const projectsFileName = "projects.json"

// Project 表示一个被跟踪的项目目录
// 该结构体记录了项目的基本信息，包括路径、数据目录位置和最后访问时间
type Project struct {
	// Path 项目的根目录路径（工作目录）
	Path string `json:"path"`
	// DataDir 项目数据存储目录路径
	DataDir string `json:"data_dir"`
	// LastAccessed 项目最后被访问的时间（UTC时间）
	LastAccessed time.Time `json:"last_accessed"`
}

// ProjectList 保存被跟踪项目的列表
// 该结构体用于序列化和反序列化项目列表数据
type ProjectList struct {
	// Projects 项目列表，按最后访问时间降序排列
	Projects []Project `json:"projects"`
}

// mu 用于保护项目列表并发访问的互斥锁
var mu sync.Mutex

// projectsFilePath 返回 projects.json 文件的完整路径
// 该文件位于全局配置数据目录的同级目录下
// 返回值：projects.json 文件的绝对路径
func projectsFilePath() string {
	return filepath.Join(filepath.Dir(config.GlobalConfigData()), projectsFileName)
}

// Load 从磁盘读取项目列表
// 如果文件不存在，返回空的项目列表而不是错误
// 该函数是线程安全的，使用互斥锁保护并发访问
// 返回值：
//   - *ProjectList: 项目列表指针
//   - error: 读取或解析过程中的错误，文件不存在时返回空列表
func Load() (*ProjectList, error) {
	mu.Lock()
	defer mu.Unlock()

	path := projectsFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		// 如果文件不存在，返回空的项目列表
		if os.IsNotExist(err) {
			return &ProjectList{Projects: []Project{}}, nil
		}
		return nil, err
	}

	var list ProjectList
	// 解析 JSON 数据到项目列表结构体
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}

	return &list, nil
}

// Save 将项目列表写入磁盘
// 该函数会自动创建必要的目录结构，并以格式化的 JSON 格式保存数据
// 该函数是线程安全的，使用互斥锁保护并发访问
// 参数：
//   - list: 要保存的项目列表指针
// 返回值：
//   - error: 保存过程中的错误
func Save(list *ProjectList) error {
	mu.Lock()
	defer mu.Unlock()

	path := projectsFilePath()

	// 确保目录存在，如果不存在则创建
	// 权限设置为 0700（仅所有者可读写执行）
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	// 将项目列表序列化为格式化的 JSON 数据
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}

	// 写入文件，权限设置为 0600（仅所有者可读写）
	return os.WriteFile(path, data, 0o600)
}

// Register 在列表中添加或更新项目
// 如果项目已存在（通过路径匹配），则更新其数据目录和最后访问时间
// 如果项目不存在，则添加新项目到列表
// 注册后，项目列表会按最后访问时间降序排序（最近访问的在前）
// 参数：
//   - workingDir: 项目的工作目录路径
//   - dataDir: 项目的数据存储目录路径
// 返回值：
//   - error: 加载或保存过程中的错误
func Register(workingDir, dataDir string) error {
	list, err := Load()
	if err != nil {
		return err
	}

	now := time.Now().UTC()

	// 检查项目是否已存在于列表中
	found := false
	for i, p := range list.Projects {
		if p.Path == workingDir {
			// 项目已存在，更新其数据目录和最后访问时间
			list.Projects[i].DataDir = dataDir
			list.Projects[i].LastAccessed = now
			found = true
			break
		}
	}

	if !found {
		// 项目不存在，添加新项目到列表
		list.Projects = append(list.Projects, Project{
			Path:         workingDir,
			DataDir:      dataDir,
			LastAccessed: now,
		})
	}

	// 按最后访问时间排序（最近的在前）
	// 使用 slices.SortFunc 进行自定义排序
	slices.SortFunc(list.Projects, func(a, b Project) int {
		if a.LastAccessed.After(b.LastAccessed) {
			return -1 // a 在 b 之前
		}
		if a.LastAccessed.Before(b.LastAccessed) {
			return 1 // a 在 b 之后
		}
		return 0 // 时间相等
	})

	return Save(list)
}

// List 返回所有被跟踪的项目，按最后访问时间降序排序
// 该函数提供了获取项目列表的便捷方法
// 返回值：
//   - []Project: 项目列表，按最后访问时间降序排列
//   - error: 加载过程中的错误
func List() ([]Project, error) {
	list, err := Load()
	if err != nil {
		return nil, err
	}
	return list.Projects, nil
}
