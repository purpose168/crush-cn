package config

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/purpose168/crush-cn/internal/env"
	"github.com/purpose168/crush-cn/internal/shell"
)

// VariableResolver 定义了变量解析器的接口，用于解析字符串中的变量引用
type VariableResolver interface {
	ResolveValue(value string) (string, error)
}

// Shell 定义了Shell命令执行接口
type Shell interface {
	Exec(ctx context.Context, command string) (stdout, stderr string, err error)
}

// shellVariableResolver 是一个基于Shell的变量解析器实现
// 支持解析命令替换和环境变量替换
type shellVariableResolver struct {
	shell Shell
	env   env.Env
}

// NewShellVariableResolver 创建一个新的Shell变量解析器
// 参数 env: 环境变量管理器
func NewShellVariableResolver(env env.Env) VariableResolver {
	return &shellVariableResolver{
		env: env,
		shell: shell.NewShell(
			&shell.Options{
				Env: env.Env(),
			},
		),
	}
}

// ResolveValue 是用于解析值的方法，例如环境变量
// 它将解析字符串中任何位置的类似Shell的变量替换，包括：
// - $(command) 用于命令替换（command substitution）
// - $VAR 或 ${VAR} 用于环境变量
func (r *shellVariableResolver) ResolveValue(value string) (string, error) {
	// 特殊情况：单独的 $ 是一个错误（向后兼容性）
	if value == "$" {
		return "", fmt.Errorf("无效的值格式: %s", value)
	}

	// 如果没有找到 $，则原样返回
	if !strings.Contains(value, "$") {
		return value, nil
	}

	result := value

	// 处理命令替换：$(command)
	for {
		start := strings.Index(result, "$(")
		if start == -1 {
			break
		}

		// 查找匹配的右括号
		depth := 0
		end := -1
		for i := start + 2; i < len(result); i++ {
			if result[i] == '(' {
				depth++
			} else if result[i] == ')' {
				if depth == 0 {
					end = i
					break
				}
				depth--
			}
		}

		if end == -1 {
			return "", fmt.Errorf("值中未匹配的 $( : %s", value)
		}

		command := result[start+2 : end]
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

		stdout, _, err := r.shell.Exec(ctx, command)
		cancel()
		if err != nil {
			return "", fmt.Errorf("命令执行失败 '%s': %w", command, err)
		}

		// 用命令输出替换 $(command)
		replacement := strings.TrimSpace(stdout)
		result = result[:start] + replacement + result[end+1:]
	}

	// 处理环境变量：$VAR 和 ${VAR}
	searchStart := 0
	for {
		start := strings.Index(result[searchStart:], "$")
		if start == -1 {
			break
		}
		start += searchStart // 调整偏移量

		// 如果这是我们已经处理过的 $( 的一部分，则跳过
		if start+1 < len(result) && result[start+1] == '(' {
			// 跳过这个 $(...)
			searchStart = start + 1
			continue
		}
		var varName string
		var end int

		if start+1 < len(result) && result[start+1] == '{' {
			// 处理 ${VAR} 格式
			closeIdx := strings.Index(result[start+2:], "}")
			if closeIdx == -1 {
				return "", fmt.Errorf("值中未匹配的 ${ : %s", value)
			}
			varName = result[start+2 : start+2+closeIdx]
			end = start + 2 + closeIdx + 1
		} else {
			// 处理 $VAR 格式 - 变量名必须以字母或下划线开头
			if start+1 >= len(result) {
				return "", fmt.Errorf("字符串末尾的变量引用不完整: %s", value)
			}

			if result[start+1] != '_' &&
				(result[start+1] < 'a' || result[start+1] > 'z') &&
				(result[start+1] < 'A' || result[start+1] > 'Z') {
				return "", fmt.Errorf("无效的变量名，以 '%c' 开头: %s", result[start+1], value)
			}

			end = start + 1
			for end < len(result) && (result[end] == '_' ||
				(result[end] >= 'a' && result[end] <= 'z') ||
				(result[end] >= 'A' && result[end] <= 'Z') ||
				(result[end] >= '0' && result[end] <= '9')) {
				end++
			}
			varName = result[start+1 : end]
		}

		envValue := r.env.Get(varName)
		if envValue == "" {
			return "", fmt.Errorf("环境变量 %q 未设置", varName)
		}

		result = result[:start] + envValue + result[end:]
		searchStart = start + len(envValue) // 在替换后继续搜索
	}

	return result, nil
}

// environmentVariableResolver 是一个环境变量解析器实现
// 仅支持解析简单的环境变量引用
type environmentVariableResolver struct {
	env env.Env
}

// NewEnvironmentVariableResolver 创建一个新的环境变量解析器
// 参数 env: 环境变量管理器
func NewEnvironmentVariableResolver(env env.Env) VariableResolver {
	return &environmentVariableResolver{
		env: env,
	}
}

// ResolveValue 从提供的 env.Env 中解析环境变量
// 参数 value: 要解析的值，必须以 $ 开头
func (r *environmentVariableResolver) ResolveValue(value string) (string, error) {
	if !strings.HasPrefix(value, "$") {
		return value, nil
	}

	varName := strings.TrimPrefix(value, "$")
	resolvedValue := r.env.Get(varName)
	if resolvedValue == "" {
		return "", fmt.Errorf("环境变量 %q 未设置", varName)
	}
	return resolvedValue, nil
}
