package tools

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegexCache(t *testing.T) {
	cache := newRegexCache()

	// 测试基本缓存功能
	pattern := "test.*pattern"
	regex1, err := cache.get(pattern)
	if err != nil {
		t.Fatalf("编译正则表达式失败: %v", err)
	}

	regex2, err := cache.get(pattern)
	if err != nil {
		t.Fatalf("获取缓存的正则表达式失败: %v", err)
	}

	// 应该是同一个实例（已缓存）
	if regex1 != regex2 {
		t.Error("期望缓存的正则表达式是同一个实例")
	}

	// 测试正则表达式是否正常工作
	if !regex1.MatchString("test123pattern") {
		t.Error("正则表达式应该匹配测试字符串")
	}
}

func TestGlobToRegexCaching(t *testing.T) {
	// 测试 globToRegex 是否使用预编译的正则表达式
	pattern1 := globToRegex("*.{js,ts}")

	// 应该不会 panic 并且应该正常工作
	regex1, err := regexp.Compile(pattern1)
	if err != nil {
		t.Fatalf("编译 glob 正则表达式失败: %v", err)
	}

	if !regex1.MatchString("test.js") {
		t.Error("Glob 正则表达式应该匹配 .js 文件")
	}
	if !regex1.MatchString("test.ts") {
		t.Error("Glob 正则表达式应该匹配 .ts 文件")
	}
	if regex1.MatchString("test.go") {
		t.Error("Glob 正则表达式不应该匹配 .go 文件")
	}
}

func TestGrepWithIgnoreFiles(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()

	// 创建测试文件
	testFiles := map[string]string{
		"file1.txt":           "hello world",
		"file2.txt":           "hello world",
		"ignored/file3.txt":   "hello world",
		"node_modules/lib.js": "hello world",
		"secret.key":          "hello world",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tempDir, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0o755))
		require.NoError(t, os.WriteFile(fullPath, []byte(content), 0o644))
	}

	// 创建 .gitignore 文件
	gitignoreContent := "ignored/\n*.key\n"
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, ".gitignore"), []byte(gitignoreContent), 0o644))

	// 创建 .crushignore 文件
	crushignoreContent := "node_modules/\n"
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, ".crushignore"), []byte(crushignoreContent), 0o644))

	// 测试两种实现
	for name, fn := range map[string]func(pattern, path, include string) ([]grepMatch, error){
		"regex": searchFilesWithRegex,
		"rg": func(pattern, path, include string) ([]grepMatch, error) {
			return searchWithRipgrep(t.Context(), pattern, path, include)
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if name == "rg" && getRg() == "" {
				t.Skip("rg 不在 $PATH 中")
			}

			matches, err := fn("hello world", tempDir, "")
			require.NoError(t, err)

			// 将匹配结果转换为文件路径集合，以便于测试
			foundFiles := make(map[string]bool)
			for _, match := range matches {
				foundFiles[filepath.Base(match.path)] = true
			}

			// 应该找到 file1.txt 和 file2.txt
			require.True(t, foundFiles["file1.txt"], "应该找到 file1.txt")
			require.True(t, foundFiles["file2.txt"], "应该找到 file2.txt")

			// 不应该找到被忽略的文件
			require.False(t, foundFiles["file3.txt"], "不应该找到 file3.txt（被 .gitignore 忽略）")
			require.False(t, foundFiles["lib.js"], "不应该找到 lib.js（被 .crushignore 忽略）")
			require.False(t, foundFiles["secret.key"], "不应该找到 secret.key（被 .gitignore 忽略）")

			// 应该正好找到 2 个匹配
			require.Equal(t, 2, len(matches), "应该正好找到 2 个匹配")
		})
	}
}

func TestSearchImplementations(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()

	for path, content := range map[string]string{
		"file1.go":         "package main\nfunc main() {\n\tfmt.Println(\"hello world\")\n}",
		"file2.js":         "console.log('hello world');",
		"file3.txt":        "hello world from text file",
		"binary.exe":       "\x00\x01\x02\x03",
		"empty.txt":        "",
		"subdir/nested.go": "package nested\n// hello world comment",
		".hidden.txt":      "hello world in hidden file",
		"file4.txt":        "hello world from a banana",
		"file5.txt":        "hello world from a grape",
	} {
		fullPath := filepath.Join(tempDir, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0o755))
		require.NoError(t, os.WriteFile(fullPath, []byte(content), 0o644))
	}

	require.NoError(t, os.WriteFile(filepath.Join(tempDir, ".gitignore"), []byte("file4.txt\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, ".crushignore"), []byte("file5.txt\n"), 0o644))

	for name, fn := range map[string]func(pattern, path, include string) ([]grepMatch, error){
		"regex": searchFilesWithRegex,
		"rg": func(pattern, path, include string) ([]grepMatch, error) {
			return searchWithRipgrep(t.Context(), pattern, path, include)
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if name == "rg" && getRg() == "" {
				t.Skip("rg 不在 $PATH 中")
			}

			matches, err := fn("hello world", tempDir, "")
			require.NoError(t, err)

			require.Equal(t, len(matches), 4)
			for _, match := range matches {
				require.NotEmpty(t, match.path)
				require.NotZero(t, match.lineNum)
				require.NotEmpty(t, match.lineText)
				require.NotZero(t, match.modTime)
				require.NotContains(t, match.path, ".hidden.txt")
				require.NotContains(t, match.path, "file4.txt")
				require.NotContains(t, match.path, "file5.txt")
				require.NotContains(t, match.path, "binary.exe")
			}
		})
	}
}

// 基准测试，展示性能改进
func BenchmarkRegexCacheVsCompile(b *testing.B) {
	cache := newRegexCache()
	pattern := "test.*pattern.*[0-9]+"

	b.Run("WithCache", func(b *testing.B) {
		for b.Loop() {
			_, err := cache.get(pattern)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("WithoutCache", func(b *testing.B) {
		for b.Loop() {
			_, err := regexp.Compile(pattern)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func TestIsTextFile(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()

	tests := []struct {
		name     string
		filename string
		content  []byte
		wantText bool
	}{
		{
			name:     "go 文件",
			filename: "test.go",
			content:  []byte("package main\n\nfunc main() {}\n"),
			wantText: true,
		},
		{
			name:     "yaml 文件",
			filename: "config.yaml",
			content:  []byte("key: value\nlist:\n  - item1\n  - item2\n"),
			wantText: true,
		},
		{
			name:     "yml 文件",
			filename: "config.yml",
			content:  []byte("key: value\n"),
			wantText: true,
		},
		{
			name:     "json 文件",
			filename: "data.json",
			content:  []byte(`{"key": "value"}`),
			wantText: true,
		},
		{
			name:     "javascript 文件",
			filename: "script.js",
			content:  []byte("console.log('hello');\n"),
			wantText: true,
		},
		{
			name:     "typescript 文件",
			filename: "script.ts",
			content:  []byte("const x: string = 'hello';\n"),
			wantText: true,
		},
		{
			name:     "markdown 文件",
			filename: "README.md",
			content:  []byte("# Title\n\nSome content\n"),
			wantText: true,
		},
		{
			name:     "shell 脚本",
			filename: "script.sh",
			content:  []byte("#!/bin/bash\necho 'hello'\n"),
			wantText: true,
		},
		{
			name:     "python 文件",
			filename: "script.py",
			content:  []byte("print('hello')\n"),
			wantText: true,
		},
		{
			name:     "xml 文件",
			filename: "data.xml",
			content:  []byte("<?xml version=\"1.0\"?>\n<root></root>\n"),
			wantText: true,
		},
		{
			name:     "纯文本",
			filename: "file.txt",
			content:  []byte("plain text content\n"),
			wantText: true,
		},
		{
			name:     "css 文件",
			filename: "style.css",
			content:  []byte("body { color: red; }\n"),
			wantText: true,
		},
		{
			name:     "scss 文件",
			filename: "style.scss",
			content:  []byte("$primary: blue;\nbody { color: $primary; }\n"),
			wantText: true,
		},
		{
			name:     "sass 文件",
			filename: "style.sass",
			content:  []byte("$primary: blue\nbody\n  color: $primary\n"),
			wantText: true,
		},
		{
			name:     "rust 文件",
			filename: "main.rs",
			content:  []byte("fn main() {\n    println!(\"Hello, world!\");\n}\n"),
			wantText: true,
		},
		{
			name:     "zig 文件",
			filename: "main.zig",
			content:  []byte("const std = @import(\"std\");\npub fn main() void {}\n"),
			wantText: true,
		},
		{
			name:     "java 文件",
			filename: "Main.java",
			content:  []byte("public class Main {\n    public static void main(String[] args) {}\n}\n"),
			wantText: true,
		},
		{
			name:     "c 文件",
			filename: "main.c",
			content:  []byte("#include <stdio.h>\nint main() { return 0; }\n"),
			wantText: true,
		},
		{
			name:     "cpp 文件",
			filename: "main.cpp",
			content:  []byte("#include <iostream>\nint main() { return 0; }\n"),
			wantText: true,
		},
		{
			name:     "fish shell",
			filename: "script.fish",
			content:  []byte("#!/usr/bin/env fish\necho 'hello'\n"),
			wantText: true,
		},
		{
			name:     "powershell 文件",
			filename: "script.ps1",
			content:  []byte("Write-Host 'Hello, World!'\n"),
			wantText: true,
		},
		{
			name:     "cmd 批处理文件",
			filename: "script.bat",
			content:  []byte("@echo off\necho Hello, World!\n"),
			wantText: true,
		},
		{
			name:     "cmd 文件",
			filename: "script.cmd",
			content:  []byte("@echo off\necho Hello, World!\n"),
			wantText: true,
		},
		{
			name:     "二进制 exe",
			filename: "binary.exe",
			content:  []byte{0x4D, 0x5A, 0x90, 0x00, 0x03, 0x00, 0x00, 0x00},
			wantText: false,
		},
		{
			name:     "png 图片",
			filename: "image.png",
			content:  []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
			wantText: false,
		},
		{
			name:     "jpeg 图片",
			filename: "image.jpg",
			content:  []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46},
			wantText: false,
		},
		{
			name:     "zip 归档",
			filename: "archive.zip",
			content:  []byte{0x50, 0x4B, 0x03, 0x04, 0x14, 0x00, 0x00, 0x00},
			wantText: false,
		},
		{
			name:     "pdf 文件",
			filename: "document.pdf",
			content:  []byte("%PDF-1.4\n%âãÏÓ\n"),
			wantText: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			filePath := filepath.Join(tempDir, tt.filename)
			require.NoError(t, os.WriteFile(filePath, tt.content, 0o644))

			got := isTextFile(filePath)
			require.Equal(t, tt.wantText, got, "isTextFile(%s) = %v, want %v", tt.filename, got, tt.wantText)
		})
	}
}

func TestColumnMatch(t *testing.T) {
	t.Parallel()

	// 测试两种实现
	for name, fn := range map[string]func(pattern, path, include string) ([]grepMatch, error){
		"regex": searchFilesWithRegex,
		"rg": func(pattern, path, include string) ([]grepMatch, error) {
			return searchWithRipgrep(t.Context(), pattern, path, include)
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if name == "rg" && getRg() == "" {
				t.Skip("rg 不在 $PATH 中")
			}

			matches, err := fn("THIS", "./testdata/", "")
			require.NoError(t, err)
			require.Len(t, matches, 1)
			match := matches[0]
			require.Equal(t, 2, match.lineNum)
			require.Equal(t, 14, match.charNum)
			require.Equal(t, "I wanna grep THIS particular word", match.lineText)
			require.Equal(t, "testdata/grep.txt", filepath.ToSlash(filepath.Clean(match.path)))
		})
	}
}
