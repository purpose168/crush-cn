package config

import (
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkLoadFromConfigPaths(b *testing.B) {
	// 创建包含真实内容的临时配置文件
	tmpDir := b.TempDir()

	// 设置全局配置文件路径
	globalConfig := filepath.Join(tmpDir, "global.json")
	// 设置本地配置文件路径
	localConfig := filepath.Join(tmpDir, "local.json")

	// 全局配置内容：包含多个提供商的API密钥和基础URL，以及TUI主题选项
	globalContent := []byte(`{
		"providers": {
			"openai": {
				"api_key": "$OPENAI_API_KEY",
				"base_url": "https://api.openai.com/v1"
			},
			"anthropic": {
				"api_key": "$ANTHROPIC_API_KEY",
				"base_url": "https://api.anthropic.com"
			}
		},
		"options": {
			"tui": {
				"theme": "dark"
			}
		}
	}`)

	// 本地配置内容：覆盖全局配置中的某些设置，如API密钥和上下文路径
	localContent := []byte(`{
		"providers": {
			"openai": {
				"api_key": "sk-override-key"
			}
		},
		"options": {
			"context_paths": ["README.md", "AGENTS.md"]
		}
	}`)

	// 写入全局配置文件
	if err := os.WriteFile(globalConfig, globalContent, 0o644); err != nil {
		b.Fatal(err)
	}
	// 写入本地配置文件
	if err := os.WriteFile(localConfig, localContent, 0o644); err != nil {
		b.Fatal(err)
	}

	// 配置文件路径列表
	configPaths := []string{globalConfig, localConfig}

	// 报告内存分配统计信息
	b.ReportAllocs()
	// 循环执行基准测试
	for b.Loop() {
		_, err := loadFromConfigPaths(configPaths)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLoadFromConfigPaths_MissingFiles(b *testing.B) {
	// 测试混合使用已存在和不存在的路径
	tmpDir := b.TempDir()

	// 创建存在的配置文件
	existingConfig := filepath.Join(tmpDir, "exists.json")
	// 配置内容：包含TUI主题选项
	content := []byte(`{"options": {"tui": {"theme": "dark"}}}`)
	if err := os.WriteFile(existingConfig, content, 0o644); err != nil {
		b.Fatal(err)
	}

	// 配置路径列表：包含不存在的文件和存在的文件
	configPaths := []string{
		filepath.Join(tmpDir, "nonexistent1.json"),
		existingConfig,
		filepath.Join(tmpDir, "nonexistent2.json"),
	}

	// 报告内存分配统计信息
	b.ReportAllocs()
	// 循环执行基准测试
	for b.Loop() {
		_, err := loadFromConfigPaths(configPaths)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLoadFromConfigPaths_Empty(b *testing.B) {
	// 测试没有配置文件的情况
	tmpDir := b.TempDir()
	// 配置路径列表：所有文件都不存在
	configPaths := []string{
		filepath.Join(tmpDir, "nonexistent1.json"),
		filepath.Join(tmpDir, "nonexistent2.json"),
	}

	// 报告内存分配统计信息
	b.ReportAllocs()
	// 循环执行基准测试
	for b.Loop() {
		_, err := loadFromConfigPaths(configPaths)
		if err != nil {
			b.Fatal(err)
		}
	}
}
