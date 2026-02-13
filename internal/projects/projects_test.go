// Package projects 提供项目注册和管理的测试功能
package projects

import (
	"path/filepath"
	"testing"
	"time"
)

// TestRegisterAndList 测试项目注册和列表功能
// 该测试验证了项目注册和列表查询的基本功能，包括：
// 1. 注册单个项目
// 2. 列出已注册的项目
// 3. 验证项目信息正确性
// 4. 验证多个项目的注册顺序（最新的项目排在前面）
func TestRegisterAndList(t *testing.T) {
	// 为测试创建临时目录
	tmpDir := t.TempDir()

	// 为测试覆盖项目文件路径，设置环境变量
	t.Setenv("XDG_DATA_HOME", tmpDir)
	t.Setenv("CRUSH_GLOBAL_DATA", filepath.Join(tmpDir, "crush"))

	// 测试注册项目功能
	err := Register("/home/user/project1", "/home/user/project1/.crush")
	if err != nil {
		t.Fatalf("注册失败: %v", err)
	}

	// 列出已注册的项目
	projects, err := List()
	if err != nil {
		t.Fatalf("列表查询失败: %v", err)
	}

	// 验证项目数量是否为1
	if len(projects) != 1 {
		t.Fatalf("期望1个项目，实际得到%d个", len(projects))
	}

	// 验证项目路径是否正确
	if projects[0].Path != "/home/user/project1" {
		t.Errorf("期望路径为/home/user/project1，实际得到%s", projects[0].Path)
	}

	// 验证数据目录是否正确
	if projects[0].DataDir != "/home/user/project1/.crush" {
		t.Errorf("期望数据目录为/home/user/project1/.crush，实际得到%s", projects[0].DataDir)
	}

	// 注册另一个项目
	err = Register("/home/user/project2", "/home/user/project2/.crush")
	if err != nil {
		t.Fatalf("注册失败: %v", err)
	}

	// 再次列出项目
	projects, err = List()
	if err != nil {
		t.Fatalf("列表查询失败: %v", err)
	}

	// 验证项目数量是否为2
	if len(projects) != 2 {
		t.Fatalf("期望2个项目，实际得到%d个", len(projects))
	}

	// 验证最新注册的项目应该排在第一位
	if projects[0].Path != "/home/user/project2" {
		t.Errorf("期望最新的项目排在第一位，实际得到%s", projects[0].Path)
	}
}

// TestRegisterUpdatesExisting 测试更新已存在项目的功能
// 该测试验证了当重新注册已存在的项目时：
// 1. 项目数量不会增加
// 2. 项目信息会被更新（如数据目录）
// 3. 最后访问时间会被更新
func TestRegisterUpdatesExisting(t *testing.T) {
	// 创建临时目录并设置环境变量
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)
	t.Setenv("CRUSH_GLOBAL_DATA", filepath.Join(tmpDir, "crush"))

	// 注册一个项目
	err := Register("/home/user/project1", "/home/user/project1/.crush")
	if err != nil {
		t.Fatalf("注册失败: %v", err)
	}

	// 获取项目列表并记录首次访问时间
	projects, _ := List()
	firstAccess := projects[0].LastAccessed

	// 等待一小段时间后重新注册，确保时间戳有差异
	time.Sleep(10 * time.Millisecond)

	// 使用新的数据目录重新注册同一项目
	err = Register("/home/user/project1", "/home/user/project1/.crush-new")
	if err != nil {
		t.Fatalf("注册失败: %v", err)
	}

	// 再次获取项目列表
	projects, _ = List()

	// 验证项目数量仍为1（未重复添加）
	if len(projects) != 1 {
		t.Fatalf("更新后期望1个项目，实际得到%d个", len(projects))
	}

	// 验证数据目录已更新
	if projects[0].DataDir != "/home/user/project1/.crush-new" {
		t.Errorf("期望更新后的数据目录，实际得到%s", projects[0].DataDir)
	}

	// 验证最后访问时间已更新
	if !projects[0].LastAccessed.After(firstAccess) {
		t.Error("期望最后访问时间已更新")
	}
}

// TestLoadEmptyFile 测试加载空文件的情况
// 该测试验证了在没有任何项目注册时，列表查询应该返回空列表而不报错
func TestLoadEmptyFile(t *testing.T) {
	// 创建临时目录并设置环境变量
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)
	t.Setenv("CRUSH_GLOBAL_DATA", filepath.Join(tmpDir, "crush"))

	// 在没有任何项目存在的情况下列出项目
	projects, err := List()
	if err != nil {
		t.Fatalf("列表查询失败: %v", err)
	}

	// 验证返回的项目列表为空
	if len(projects) != 0 {
		t.Errorf("期望0个项目，实际得到%d个", len(projects))
	}
}

// TestProjectsFilePath 测试项目文件路径的生成
// 该测试验证了项目文件路径是否按照预期格式生成
// 预期路径格式：<XDG_DATA_HOME>/crush/projects.json
func TestProjectsFilePath(t *testing.T) {
	// 创建临时目录并设置环境变量
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)
	t.Setenv("CRUSH_GLOBAL_DATA", filepath.Join(tmpDir, "crush"))

	// 构建预期的项目文件路径
	expected := filepath.Join(tmpDir, "crush", "projects.json")
	// 获取实际的路径
	actual := projectsFilePath()

	// 验证路径是否正确
	if actual != expected {
		t.Errorf("期望%s，实际得到%s", expected, actual)
	}
}

// TestRegisterWithParentDataDir 测试数据目录在父目录中的情况
// 该测试验证了当 .crush 目录位于父目录时的项目注册功能
// 例如：工作目录为 /home/user/monorepo/packages/app，但 .crush 位于 /home/user/monorepo/.crush
// 这种情况常见于 monorepo（单体仓库）项目中
func TestRegisterWithParentDataDir(t *testing.T) {
	// 创建临时目录并设置环境变量
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)
	t.Setenv("CRUSH_GLOBAL_DATA", filepath.Join(tmpDir, "crush"))

	// 注册一个项目，其中 .crush 位于父目录中
	// 例如：在 /home/user/monorepo/packages/app 工作但 .crush 位于 /home/user/monorepo/.crush
	err := Register("/home/user/monorepo/packages/app", "/home/user/monorepo/.crush")
	if err != nil {
		t.Fatalf("注册失败: %v", err)
	}

	// 获取项目列表
	projects, err := List()
	if err != nil {
		t.Fatalf("列表查询失败: %v", err)
	}

	// 验证项目数量
	if len(projects) != 1 {
		t.Fatalf("期望1个项目，实际得到%d个", len(projects))
	}

	// 验证项目路径（工作目录路径）
	if projects[0].Path != "/home/user/monorepo/packages/app" {
		t.Errorf("期望路径为/home/user/monorepo/packages/app，实际得到%s", projects[0].Path)
	}

	// 验证数据目录（父目录中的 .crush）
	if projects[0].DataDir != "/home/user/monorepo/.crush" {
		t.Errorf("期望数据目录为/home/user/monorepo/.crush，实际得到%s", projects[0].DataDir)
	}
}

// TestRegisterWithExternalDataDir 测试数据目录在外部位置的情况
// 该测试验证了当 .crush 目录位于完全不同的位置时的项目注册功能
// 例如：项目位于 /home/user/project，但数据存储在 /var/data/crush/myproject
// 这种情况适用于需要将项目数据集中存储或存储在特定位置的场景
func TestRegisterWithExternalDataDir(t *testing.T) {
	// 创建临时目录并设置环境变量
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)
	t.Setenv("CRUSH_GLOBAL_DATA", filepath.Join(tmpDir, "crush"))

	// 注册一个项目，其中 .crush 位于完全不同的位置
	// 例如：项目位于 /home/user/project 但数据存储在 /var/data/crush/myproject
	err := Register("/home/user/project", "/var/data/crush/myproject")
	if err != nil {
		t.Fatalf("注册失败: %v", err)
	}

	// 获取项目列表
	projects, err := List()
	if err != nil {
		t.Fatalf("列表查询失败: %v", err)
	}

	// 验证项目数量
	if len(projects) != 1 {
		t.Fatalf("期望1个项目，实际得到%d个", len(projects))
	}

	// 验证项目路径
	if projects[0].Path != "/home/user/project" {
		t.Errorf("期望路径为/home/user/project，实际得到%s", projects[0].Path)
	}

	// 验证数据目录（外部位置）
	if projects[0].DataDir != "/var/data/crush/myproject" {
		t.Errorf("期望数据目录为/var/data/crush/myproject，实际得到%s", projects[0].DataDir)
	}
}
