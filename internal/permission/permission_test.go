package permission

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPermissionService_AllowedCommands 测试权限服务的允许命令列表功能
func TestPermissionService_AllowedCommands(t *testing.T) {
	tests := []struct {
		name         string
		allowedTools []string
		toolName     string
		action       string
		expected     bool
	}{
		{
			name:         "工具在允许列表中",
			allowedTools: []string{"bash", "view"},
			toolName:     "bash",
			action:       "execute",
			expected:     true,
		},
		{
			name:         "工具:操作在允许列表中",
			allowedTools: []string{"bash:execute", "edit:create"},
			toolName:     "bash",
			action:       "execute",
			expected:     true,
		},
		{
			name:         "工具不在允许列表中",
			allowedTools: []string{"view", "ls"},
			toolName:     "bash",
			action:       "execute",
			expected:     false,
		},
		{
			name:         "工具:操作不在允许列表中",
			allowedTools: []string{"bash:read", "edit:create"},
			toolName:     "bash",
			action:       "execute",
			expected:     false,
		},
		{
			name:         "空的允许列表",
			allowedTools: []string{},
			toolName:     "bash",
			action:       "execute",
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewPermissionService("/tmp", false, tt.allowedTools)

			// 创建一个通道来捕获权限请求
			// 由于我们正在测试允许列表逻辑，需要模拟请求
			ps := service.(*permissionService)

			// 直接测试允许列表逻辑
			commandKey := tt.toolName + ":" + tt.action
			allowed := false
			for _, cmd := range ps.allowedTools {
				if cmd == commandKey || cmd == tt.toolName {
					allowed = true
					break
				}
			}

			if allowed != tt.expected {
				t.Errorf("期望 %v，但得到 %v，工具 %s 操作 %s，允许列表 %v",
					tt.expected, allowed, tt.toolName, tt.action, tt.allowedTools)
			}
		})
	}
}

// TestPermissionService_SkipMode 测试权限服务的跳过模式功能
func TestPermissionService_SkipMode(t *testing.T) {
	service := NewPermissionService("/tmp", true, []string{})

	result, err := service.Request(t.Context(), CreatePermissionRequest{
		SessionID:   "test-session",
		ToolName:    "bash",
		Action:      "execute",
		Description: "测试命令",
		Path:        "/tmp",
	})
	if err != nil {
		t.Errorf("意外的错误: %v", err)
	}
	if !result {
		t.Error("在跳过模式下应该授予权限")
	}
}

// TestPermissionService_SequentialProperties 测试权限服务的顺序请求属性
func TestPermissionService_SequentialProperties(t *testing.T) {
	t.Run("带有持久授权的顺序权限请求", func(t *testing.T) {
		service := NewPermissionService("/tmp", false, []string{})

		req1 := CreatePermissionRequest{
			SessionID:   "session1",
			ToolName:    "file_tool",
			Description: "读取文件",
			Action:      "read",
			Params:      map[string]string{"file": "test.txt"},
			Path:        "/tmp/test.txt",
		}

		var result1 bool
		var wg sync.WaitGroup
		wg.Add(1)

		events := service.Subscribe(t.Context())

		go func() {
			defer wg.Done()
			result1, _ = service.Request(t.Context(), req1)
		}()

		var permissionReq PermissionRequest
		event := <-events

		permissionReq = event.Payload
		service.GrantPersistent(permissionReq)

		wg.Wait()
		assert.True(t, result1, "第一个请求应该被授权")

		// 第二个相同的请求由于持久权限应该自动批准
		req2 := CreatePermissionRequest{
			SessionID:   "session1",
			ToolName:    "file_tool",
			Description: "再次读取文件",
			Action:      "read",
			Params:      map[string]string{"file": "test.txt"},
			Path:        "/tmp/test.txt",
		}
		result2, err := service.Request(t.Context(), req2)
		require.NoError(t, err)
		assert.True(t, result2, "第二个请求应该自动批准")
	})
	t.Run("带有临时授权的顺序请求", func(t *testing.T) {
		service := NewPermissionService("/tmp", false, []string{})

		req := CreatePermissionRequest{
			SessionID:   "session2",
			ToolName:    "file_tool",
			Description: "写入文件",
			Action:      "write",
			Params:      map[string]string{"file": "test.txt"},
			Path:        "/tmp/test.txt",
		}

		events := service.Subscribe(t.Context())
		var result1 bool
		var wg sync.WaitGroup

		wg.Go(func() {
			result1, _ = service.Request(t.Context(), req)
		})

		var permissionReq PermissionRequest
		event := <-events
		permissionReq = event.Payload

		service.Grant(permissionReq)
		wg.Wait()
		assert.True(t, result1, "第一个请求应该被授权")

		var result2 bool

		wg.Go(func() {
			result2, _ = service.Request(t.Context(), req)
		})

		event = <-events
		permissionReq = event.Payload
		service.Deny(permissionReq)
		wg.Wait()
		assert.False(t, result2, "第二个请求应该被拒绝")
	})
	t.Run("具有不同结果的并发请求", func(t *testing.T) {
		service := NewPermissionService("/tmp", false, []string{})

		events := service.Subscribe(t.Context())

		var wg sync.WaitGroup
		results := make([]bool, 3)

		requests := []CreatePermissionRequest{
			{
				SessionID:   "concurrent1",
				ToolName:    "tool1",
				Action:      "action1",
				Path:        "/tmp/file1.txt",
				Description: "第一个并发请求",
			},
			{
				SessionID:   "concurrent2",
				ToolName:    "tool2",
				Action:      "action2",
				Path:        "/tmp/file2.txt",
				Description: "第二个并发请求",
			},
			{
				SessionID:   "concurrent3",
				ToolName:    "tool3",
				Action:      "action3",
				Path:        "/tmp/file3.txt",
				Description: "第三个并发请求",
			},
		}

		for i, req := range requests {
			wg.Add(1)
			go func(index int, request CreatePermissionRequest) {
				defer wg.Done()
				result, _ := service.Request(t.Context(), request)
				results[index] = result
			}(i, req)
		}

		for range 3 {
			event := <-events
			switch event.Payload.ToolName {
			case "tool1":
				service.Grant(event.Payload)
			case "tool2":
				service.GrantPersistent(event.Payload)
			case "tool3":
				service.Deny(event.Payload)
			}
		}
		wg.Wait()
		grantedCount := 0
		for _, result := range results {
			if result {
				grantedCount++
			}
		}

		assert.Equal(t, 2, grantedCount, "应该有2个授权和1个拒绝")
		secondReq := requests[1]
		secondReq.Description = "重复第二个请求"
		result, err := service.Request(t.Context(), secondReq)
		require.NoError(t, err)
		assert.True(t, result, "重复的请求由于持久权限应该自动批准")
	})
}
