package lsp

import (
	"context"
	"testing"

	"github.com/purpose168/crush-cn/internal/config"
	"github.com/purpose168/crush-cn/internal/env"
)

// TestClient 测试LSP客户端的基本功能
// 该测试验证：
// 1. 客户端能够正常创建
// 2. GetName方法返回正确的名称
// 3. HandlesFile方法正确判断文件类型
// 4. 服务器状态的设置和获取
func TestClient(t *testing.T) {
	ctx := context.Background()

	// 创建用于测试的简单配置
	cfg := config.LSPConfig{
		Command:   "$THE_CMD",  // 使用echo作为不会失败的虚拟命令
		Args:      []string{"hello"},
		FileTypes: []string{"go"},
		Env:       map[string]string{},
	}

	// 测试创建powernap客户端 - 这可能会因为echo命令失败
	// 但我们仍然可以测试基本结构
	client, err := New(ctx, "test", cfg, config.NewEnvironmentVariableResolver(env.NewFromMap(map[string]string{
		"THE_CMD": "echo",
	})), false)
	if err != nil {
		// 预期会因为虚拟命令失败，跳过后续测试
		t.Skipf("使用虚拟命令创建Powernap客户端失败（符合预期）: %v", err)
		return
	}

	// 如果能执行到这里，测试基本接口方法
	if client.GetName() != "test" {
		t.Errorf("期望名称为'test'，实际得到'%s'", client.GetName())
	}

	if !client.HandlesFile("test.go") {
		t.Error("期望客户端能处理.go文件")
	}

	if client.HandlesFile("test.py") {
		t.Error("期望客户端不能处理.py文件")
	}

	// 测试服务器状态
	client.SetServerState(StateReady)
	if client.GetServerState() != StateReady {
		t.Error("期望服务器状态为StateReady")
	}

	// 清理 - 预期会因为echo命令失败
	if err := client.Close(t.Context()); err != nil {
		// 预期会因为虚拟命令失败
		t.Logf("关闭失败（符合预期，使用虚拟命令）: %v", err)
	}
}
