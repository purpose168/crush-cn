package env

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOsEnv_Get(t *testing.T) {
	env := &osEnv{}

	// 测试获取已存在的环境变量
	t.Setenv("TEST_VAR", "test_value")

	value := env.Get("TEST_VAR")
	require.Equal(t, "test_value", value)

	// 测试获取不存在的环境变量
	value = env.Get("NON_EXISTENT_VAR")
	require.Equal(t, "", value)
}

func TestOsEnv_Env(t *testing.T) {
	env := &osEnv{}

	envVars := env.Env()

	// 环境变量在正常情况下不应为空
	require.NotNil(t, envVars)
	require.Greater(t, len(envVars), 0)

	// 每个环境变量应采用 key=value 格式
	for _, envVar := range envVars {
		require.Contains(t, envVar, "=")
	}
}

func TestNewFromMap(t *testing.T) {
	testMap := map[string]string{
		"KEY1": "value1",
		"KEY2": "value2",
	}

	env := NewFromMap(testMap)
	require.NotNil(t, env)
	require.IsType(t, &mapEnv{}, env)
}

func TestMapEnv_Get(t *testing.T) {
	testMap := map[string]string{
		"KEY1": "value1",
		"KEY2": "value2",
	}

	env := NewFromMap(testMap)

	// 测试获取已存在的键
	require.Equal(t, "value1", env.Get("KEY1"))
	require.Equal(t, "value2", env.Get("KEY2"))

	// 测试获取不存在的键
	require.Equal(t, "", env.Get("NON_EXISTENT"))
}

func TestMapEnv_Env(t *testing.T) {
	t.Run("with values", func(t *testing.T) {
		testMap := map[string]string{
			"KEY1": "value1",
			"KEY2": "value2",
		}

		env := NewFromMap(testMap)
		envVars := env.Env()

		require.Len(t, envVars, 2)

		// 转换为 map 以便于测试（顺序不保证）
		envMap := make(map[string]string)
		for _, envVar := range envVars {
			parts := strings.SplitN(envVar, "=", 2)
			require.Len(t, parts, 2)
			envMap[parts[0]] = parts[1]
		}

		require.Equal(t, "value1", envMap["KEY1"])
		require.Equal(t, "value2", envMap["KEY2"])
	})

	t.Run("empty map", func(t *testing.T) {
		env := NewFromMap(map[string]string{})
		envVars := env.Env()
		require.NotNil(t, envVars)
		require.Len(t, envVars, 0)
	})

	t.Run("nil map", func(t *testing.T) {
		env := NewFromMap(nil)
		envVars := env.Env()
		require.NotNil(t, envVars)
		require.Len(t, envVars, 0)
	})
}

func TestMapEnv_GetEmptyValue(t *testing.T) {
	testMap := map[string]string{
		"EMPTY_KEY":  "",
		"NORMAL_KEY": "value",
	}

	env := NewFromMap(testMap)

	// 测试空值是否正确返回
	require.Equal(t, "", env.Get("EMPTY_KEY"))
	require.Equal(t, "value", env.Get("NORMAL_KEY"))
}

func TestMapEnv_EnvFormat(t *testing.T) {
	testMap := map[string]string{
		"KEY_WITH_EQUALS": "value=with=equals",
		"KEY_WITH_SPACES": "value with spaces",
	}

	env := NewFromMap(testMap)
	envVars := env.Env()

	require.Len(t, envVars, 2)

	// 检查格式是否正确，即使包含特殊字符
	found := make(map[string]bool)
	for _, envVar := range envVars {
		if envVar == "KEY_WITH_EQUALS=value=with=equals" {
			found["equals"] = true
		}
		if envVar == "KEY_WITH_SPACES=value with spaces" {
			found["spaces"] = true
		}
	}

	require.True(t, found["equals"], "Should handle values with equals signs")
	require.True(t, found["spaces"], "Should handle values with spaces")
}
