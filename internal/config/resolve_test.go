package config

import (
	"context"
	"errors"
	"testing"

	"github.com/purpose168/crush-cn/internal/env"
	"github.com/stretchr/testify/require"
)

// mockShell 实现 Shell 接口用于测试 (mockShell implements the Shell interface for testing)
// 这是一个模拟的 Shell 实现，用于在单元测试中模拟 shell 命令的执行
type mockShell struct {
	execFunc func(ctx context.Context, command string) (stdout, stderr string, err error)
}

// Exec 执行 shell 命令并返回标准输出、标准错误和错误信息
// 如果设置了 execFunc，则调用该函数；否则返回空字符串和 nil 错误
func (m *mockShell) Exec(ctx context.Context, command string) (stdout, stderr string, err error) {
	if m.execFunc != nil {
		return m.execFunc(ctx, command)
	}
	return "", "", nil
}

// TestShellVariableResolver_ResolveValue 测试 Shell 变量解析器的值解析功能
// 该测试验证了解析器能够正确处理：
// 1. 非变量字符串
// 2. 环境变量
// 3. shell 命令替换
// 4. 各种错误情况
func TestShellVariableResolver_ResolveValue(t *testing.T) {
	// 定义测试用例，包含测试名称、输入值、环境变量、shell 函数、期望结果和是否期望错误
	tests := []struct {
		name        string
		value       string
		envVars     map[string]string
		shellFunc   func(ctx context.Context, command string) (stdout, stderr string, err error)
		expected    string
		expectError bool
	}{
		{
			name:     "非变量字符串原样返回 (non-variable string returns as-is)",
			value:    "plain-string",
			expected: "plain-string",
		},
		{
			name:     "环境变量解析 (environment variable resolution)",
			value:    "$HOME",
			envVars:  map[string]string{"HOME": "/home/user"},
			expected: "/home/user",
		},
		{
			name:        "缺失的环境变量返回错误 (missing environment variable returns error)",
			value:       "$MISSING_VAR",
			envVars:     map[string]string{},
			expectError: true,
		},

		{
			name:  "shell 命令去除空白字符 (shell command with whitespace trimming)",
			value: "$(echo '  spaced  ')",
			shellFunc: func(ctx context.Context, command string) (stdout, stderr string, err error) {
				if command == "echo '  spaced  '" {
					return "  spaced  \n", "", nil
				}
				return "", "", errors.New("意外的命令 (unexpected command)")
			},
			expected: "spaced",
		},
		{
			name:  "shell 命令执行错误 (shell command execution error)",
			value: "$(false)",
			shellFunc: func(ctx context.Context, command string) (stdout, stderr string, err error) {
				return "", "", errors.New("命令执行失败 (command failed)")
			},
			expectError: true,
		},
		{
			name:        "无效格式返回错误 (invalid format returns error)",
			value:       "$",
			expectError: true,
		},
	}

	// 遍历所有测试用例并执行测试
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 从测试环境变量创建测试环境
			testEnv := env.NewFromMap(tt.envVars)
			// 创建 shell 变量解析器，使用 mock shell 和测试环境
			resolver := &shellVariableResolver{
				shell: &mockShell{execFunc: tt.shellFunc},
				env:   testEnv,
			}

			// 调用解析器的 ResolveValue 方法解析输入值
			result, err := resolver.ResolveValue(tt.value)

			// 根据期望结果验证解析结果
			if tt.expectError {
				// 如果期望错误，验证确实返回了错误
				require.Error(t, err)
			} else {
				// 如果不期望错误，验证没有错误且结果符合预期
				require.NoError(t, err)
				require.Equal(t, tt.expected, result)
			}
		})
	}
}

// TestShellVariableResolver_EnhancedResolveValue 测试 Shell 变量解析器的增强值解析功能
// 该测试验证了解析器能够正确处理更复杂的场景：
// 1. 字符串中的命令替换
// 2. 字符串中的环境变量
// 3. 混合命令和环境变量替换
// 4. 多个命令替换
// 5. 嵌套括号
// 6. 各种边界情况和错误处理
func TestShellVariableResolver_EnhancedResolveValue(t *testing.T) {
	// 定义测试用例，包含测试名称、输入值、环境变量、shell 函数、期望结果和是否期望错误
	tests := []struct {
		name        string
		value       string
		envVars     map[string]string
		shellFunc   func(ctx context.Context, command string) (stdout, stderr string, err error)
		expected    string
		expectError bool
	}{
		{
			name:  "字符串中的命令替换 (command substitution within string)",
			value: "Bearer $(echo token123)",
			shellFunc: func(ctx context.Context, command string) (stdout, stderr string, err error) {
				if command == "echo token123" {
					return "token123\n", "", nil
				}
				return "", "", errors.New("意外的命令 (unexpected command)")
			},
			expected: "Bearer token123",
		},
		{
			name:     "字符串中的环境变量 (environment variable within string)",
			value:    "Bearer $TOKEN",
			envVars:  map[string]string{"TOKEN": "sk-ant-123"},
			expected: "Bearer sk-ant-123",
		},
		{
			name:     "字符串中使用花括号的环境变量 (environment variable with braces within string)",
			value:    "Bearer ${TOKEN}",
			envVars:  map[string]string{"TOKEN": "sk-ant-456"},
			expected: "Bearer sk-ant-456",
		},
		{
			name:  "混合命令和环境变量替换 (mixed command and environment substitution)",
			value: "$USER-$(date +%Y)-$HOST",
			envVars: map[string]string{
				"USER": "testuser",
				"HOST": "localhost",
			},
			shellFunc: func(ctx context.Context, command string) (stdout, stderr string, err error) {
				if command == "date +%Y" {
					return "2024\n", "", nil
				}
				return "", "", errors.New("意外的命令 (unexpected command)")
			},
			expected: "testuser-2024-localhost",
		},
		{
			name:  "多个命令替换 (multiple command substitutions)",
			value: "$(echo hello) $(echo world)",
			shellFunc: func(ctx context.Context, command string) (stdout, stderr string, err error) {
				switch command {
				case "echo hello":
					return "hello\n", "", nil
				case "echo world":
					return "world\n", "", nil
				}
				return "", "", errors.New("意外的命令 (unexpected command)")
			},
			expected: "hello world",
		},
		{
			name:  "命令中的嵌套括号 (nested parentheses in command)",
			value: "$(echo $(echo inner))",
			shellFunc: func(ctx context.Context, command string) (stdout, stderr string, err error) {
				if command == "echo $(echo inner)" {
					return "nested\n", "", nil
				}
				return "", "", errors.New("意外的命令 (unexpected command)")
			},
			expected: "nested",
		},
		{
			name:        "美元符号后跟非变量字符 (lone dollar with non-variable chars)",
			value:       "prefix$123suffix", // 数字不能作为变量名的开头
			expectError: true,
		},
		{
			name:        "美元符号后跟特殊字符 (dollar with special chars)",
			value:       "a$@b$#c", // 特殊字符在变量名中无效
			expectError: true,
		},
		{
			name:        "空环境变量替换 (empty environment variable substitution)",
			value:       "Bearer $EMPTY_VAR",
			envVars:     map[string]string{},
			expectError: true,
		},
		{
			name:        "未闭合的命令替换开头 (unmatched command substitution opening)",
			value:       "Bearer $(echo test",
			expectError: true,
		},
		{
			name:        "未闭合的环境变量花括号 (unmatched environment variable braces)",
			value:       "Bearer ${TOKEN",
			expectError: true,
		},
		{
			name:  "命令替换出错 (command substitution with error)",
			value: "Bearer $(false)",
			shellFunc: func(ctx context.Context, command string) (stdout, stderr string, err error) {
				return "", "", errors.New("命令执行失败 (command failed)")
			},
			expectError: true,
		},
		{
			name:  "复杂的真实世界示例 (complex real-world example)",
			value: "Bearer $(cat /tmp/token.txt | base64 -w 0)",
			shellFunc: func(ctx context.Context, command string) (stdout, stderr string, err error) {
				if command == "cat /tmp/token.txt | base64 -w 0" {
					return "c2stYW50LXRlc3Q=\n", "", nil
				}
				return "", "", errors.New("意外的命令 (unexpected command)")
			},
			expected: "Bearer c2stYW50LXRlc3Q=",
		},
		{
			name:     "包含下划线和数字的环境变量 (environment variable with underscores and numbers)",
			value:    "Bearer $API_KEY_V2",
			envVars:  map[string]string{"API_KEY_V2": "sk-test-123"},
			expected: "Bearer sk-test-123",
		},
		{
			name:     "无需替换 (no substitution needed)",
			value:    "Bearer sk-ant-static-token",
			expected: "Bearer sk-ant-static-token",
		},
		{
			name:        "结尾处不完整的变量 (incomplete variable at end)",
			value:       "Bearer $",
			expectError: true,
		},
		{
			name:        "包含无效字符的变量 (variable with invalid character)",
			value:       "Bearer $VAR-NAME", // 连字符不允许在变量名中使用
			expectError: true,
		},
		{
			name:        "多个无效变量 (multiple invalid variables)",
			value:       "$1$2$3",
			expectError: true,
		},
	}

	// 遍历所有测试用例并执行测试
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 从测试环境变量创建测试环境
			testEnv := env.NewFromMap(tt.envVars)
			// 创建 shell 变量解析器，使用 mock shell 和测试环境
			resolver := &shellVariableResolver{
				shell: &mockShell{execFunc: tt.shellFunc},
				env:   testEnv,
			}

			// 调用解析器的 ResolveValue 方法解析输入值
			result, err := resolver.ResolveValue(tt.value)

			// 根据期望结果验证解析结果
			if tt.expectError {
				// 如果期望错误，验证确实返回了错误
				require.Error(t, err)
			} else {
				// 如果不期望错误，验证没有错误且结果符合预期
				require.NoError(t, err)
				require.Equal(t, tt.expected, result)
			}
		})
	}
}

// TestEnvironmentVariableResolver_ResolveValue 测试环境变量解析器的值解析功能
// 该测试验证了解析器能够正确处理：
// 1. 非变量字符串
// 2. 环境变量解析
// 3. 包含复杂值的环境变量（如 PATH）
// 4. 缺失的环境变量
// 5. 空环境变量
func TestEnvironmentVariableResolver_ResolveValue(t *testing.T) {
	// 定义测试用例，包含测试名称、输入值、环境变量、期望结果和是否期望错误
	tests := []struct {
		name        string
		value       string
		envVars     map[string]string
		expected    string
		expectError bool
	}{
		{
			name:     "非变量字符串原样返回 (non-variable string returns as-is)",
			value:    "plain-string",
			expected: "plain-string",
		},
		{
			name:     "环境变量解析 (environment variable resolution)",
			value:    "$HOME",
			envVars:  map[string]string{"HOME": "/home/user"},
			expected: "/home/user",
		},
		{
			name:     "包含复杂值的环境变量 (environment variable with complex value)",
			value:    "$PATH",
			envVars:  map[string]string{"PATH": "/usr/bin:/bin:/usr/local/bin"},
			expected: "/usr/bin:/bin:/usr/local/bin",
		},
		{
			name:        "缺失的环境变量返回错误 (missing environment variable returns error)",
			value:       "$MISSING_VAR",
			envVars:     map[string]string{},
			expectError: true,
		},
		{
			name:        "空环境变量返回错误 (empty environment variable returns error)",
			value:       "$EMPTY_VAR",
			envVars:     map[string]string{"EMPTY_VAR": ""},
			expectError: true,
		},
	}

	// 遍历所有测试用例并执行测试
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 从测试环境变量创建测试环境
			testEnv := env.NewFromMap(tt.envVars)
			// 创建环境变量解析器
			resolver := NewEnvironmentVariableResolver(testEnv)

			// 调用解析器的 ResolveValue 方法解析输入值
			result, err := resolver.ResolveValue(tt.value)

			// 根据期望结果验证解析结果
			if tt.expectError {
				// 如果期望错误，验证确实返回了错误
				require.Error(t, err)
			} else {
				// 如果不期望错误，验证没有错误且结果符合预期
				require.NoError(t, err)
				require.Equal(t, tt.expected, result)
			}
		})
	}
}

// TestNewShellVariableResolver 测试 NewShellVariableResolver 构造函数
// 验证构造函数能够正确创建 shell 变量解析器实例
func TestNewShellVariableResolver(t *testing.T) {
	// 创建测试环境
	testEnv := env.NewFromMap(map[string]string{"TEST": "value"})
	// 创建 shell 变量解析器
	resolver := NewShellVariableResolver(testEnv)

	// 验证解析器不为 nil
	require.NotNil(t, resolver)
	// 验证解析器实现了 VariableResolver 接口
	require.Implements(t, (*VariableResolver)(nil), resolver)
}

// TestNewEnvironmentVariableResolver 测试 NewEnvironmentVariableResolver 构造函数
// 验证构造函数能够正确创建环境变量解析器实例
func TestNewEnvironmentVariableResolver(t *testing.T) {
	// 创建测试环境
	testEnv := env.NewFromMap(map[string]string{"TEST": "value"})
	// 创建环境变量解析器
	resolver := NewEnvironmentVariableResolver(testEnv)

	// 验证解析器不为 nil
	require.NotNil(t, resolver)
	// 验证解析器实现了 VariableResolver 接口
	require.Implements(t, (*VariableResolver)(nil), resolver)
}
