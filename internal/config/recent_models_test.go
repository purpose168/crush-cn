package config

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// readConfigJSON 读取并解析指定路径的JSON配置文件
func readConfigJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	baseDir := filepath.Dir(path)
	fileName := filepath.Base(path)
	b, err := fs.ReadFile(os.DirFS(baseDir), fileName)
	require.NoError(t, err)
	var out map[string]any
	require.NoError(t, json.Unmarshal(b, &out))
	return out
}

// readRecentModels 从配置文件中读取 recent_models 部分
func readRecentModels(t *testing.T, path string) map[string]any {
	t.Helper()
	out := readConfigJSON(t, path)
	rm, ok := out["recent_models"].(map[string]any)
	require.True(t, ok)
	return rm
}

func TestRecordRecentModel_AddsAndPersists(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := &Config{}
	cfg.setDefaults(dir, "")
	cfg.dataConfigDir = filepath.Join(dir, "config.json")

	err := cfg.recordRecentModel(SelectedModelTypeLarge, SelectedModel{Provider: "openai", Model: "gpt-4o"})
	require.NoError(t, err)

	// 内存中的状态
	require.Len(t, cfg.RecentModels[SelectedModelTypeLarge], 1)
	require.Equal(t, "openai", cfg.RecentModels[SelectedModelTypeLarge][0].Provider)
	require.Equal(t, "gpt-4o", cfg.RecentModels[SelectedModelTypeLarge][0].Model)

	// 持久化状态
	rm := readRecentModels(t, cfg.dataConfigDir)
	large, ok := rm[string(SelectedModelTypeLarge)].([]any)
	require.True(t, ok)
	require.Len(t, large, 1)
	item, ok := large[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "openai", item["provider"])
	require.Equal(t, "gpt-4o", item["model"])
}

func TestRecordRecentModel_DedupeAndMoveToFront(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := &Config{}
	cfg.setDefaults(dir, "")
	cfg.dataConfigDir = filepath.Join(dir, "config.json")

	// 添加两个条目
	require.NoError(t, cfg.recordRecentModel(SelectedModelTypeLarge, SelectedModel{Provider: "openai", Model: "gpt-4o"}))
	require.NoError(t, cfg.recordRecentModel(SelectedModelTypeLarge, SelectedModel{Provider: "anthropic", Model: "claude"}))
	// 重新添加第一个；应该移到前面且不重复
	require.NoError(t, cfg.recordRecentModel(SelectedModelTypeLarge, SelectedModel{Provider: "openai", Model: "gpt-4o"}))

	got := cfg.RecentModels[SelectedModelTypeLarge]
	require.Len(t, got, 2)
	require.Equal(t, SelectedModel{Provider: "openai", Model: "gpt-4o"}, got[0])
	require.Equal(t, SelectedModel{Provider: "anthropic", Model: "claude"}, got[1])
}

func TestRecordRecentModel_TrimsToMax(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := &Config{}
	cfg.setDefaults(dir, "")
	cfg.dataConfigDir = filepath.Join(dir, "config.json")

	// 插入6个不同的模型；最大值为5
	entries := []SelectedModel{
		{Provider: "p1", Model: "m1"},
		{Provider: "p2", Model: "m2"},
		{Provider: "p3", Model: "m3"},
		{Provider: "p4", Model: "m4"},
		{Provider: "p5", Model: "m5"},
		{Provider: "p6", Model: "m6"},
	}
	for _, e := range entries {
		require.NoError(t, cfg.recordRecentModel(SelectedModelTypeLarge, e))
	}

	// 内存中的状态
	got := cfg.RecentModels[SelectedModelTypeLarge]
	require.Len(t, got, 5)
	// 最新的在前，限制为5个：p6..p2
	require.Equal(t, SelectedModel{Provider: "p6", Model: "m6"}, got[0])
	require.Equal(t, SelectedModel{Provider: "p5", Model: "m5"}, got[1])
	require.Equal(t, SelectedModel{Provider: "p4", Model: "m4"}, got[2])
	require.Equal(t, SelectedModel{Provider: "p3", Model: "m3"}, got[3])
	require.Equal(t, SelectedModel{Provider: "p2", Model: "m2"}, got[4])

	// 持久化状态：验证已裁剪为5个且顺序为最新的在前
	rm := readRecentModels(t, cfg.dataConfigDir)
	large, ok := rm[string(SelectedModelTypeLarge)].([]any)
	require.True(t, ok)
	require.Len(t, large, 5)
	// 构建 provider:model 标识符并验证顺序
	var ids []string
	for _, v := range large {
		m := v.(map[string]any)
		ids = append(ids, m["provider"].(string)+":"+m["model"].(string))
	}
	require.Equal(t, []string{"p6:m6", "p5:m5", "p4:m4", "p3:m3", "p2:m2"}, ids)
}

func TestRecordRecentModel_SkipsEmptyValues(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := &Config{}
	cfg.setDefaults(dir, "")
	cfg.dataConfigDir = filepath.Join(dir, "config.json")

	// 缺少 provider
	require.NoError(t, cfg.recordRecentModel(SelectedModelTypeLarge, SelectedModel{Provider: "", Model: "m"}))
	// 缺少 model
	require.NoError(t, cfg.recordRecentModel(SelectedModelTypeLarge, SelectedModel{Provider: "p", Model: ""}))

	_, ok := cfg.RecentModels[SelectedModelTypeLarge]
	// 映射可能已初始化，但应该没有条目
	if ok {
		require.Len(t, cfg.RecentModels[SelectedModelTypeLarge], 0)
	}
	// 不应该写入文件（通过 fs.FS 进行 stat 检查）
	baseDir := filepath.Dir(cfg.dataConfigDir)
	fileName := filepath.Base(cfg.dataConfigDir)
	_, err := fs.Stat(os.DirFS(baseDir), fileName)
	require.True(t, os.IsNotExist(err))
}

func TestRecordRecentModel_NoPersistOnNoop(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := &Config{}
	cfg.setDefaults(dir, "")
	cfg.dataConfigDir = filepath.Join(dir, "config.json")

	entry := SelectedModel{Provider: "openai", Model: "gpt-4o"}
	require.NoError(t, cfg.recordRecentModel(SelectedModelTypeLarge, entry))

	baseDir := filepath.Dir(cfg.dataConfigDir)
	fileName := filepath.Base(cfg.dataConfigDir)
	before, err := fs.ReadFile(os.DirFS(baseDir), fileName)
	require.NoError(t, err)

	// 获取文件 ModTime 以验证没有发生写入操作
	stBefore, err := fs.Stat(os.DirFS(baseDir), fileName)
	require.NoError(t, err)
	beforeMod := stBefore.ModTime()

	// 重新记录相同的条目应该是一个无操作（不写入）
	require.NoError(t, cfg.recordRecentModel(SelectedModelTypeLarge, entry))

	after, err := fs.ReadFile(os.DirFS(baseDir), fileName)
	require.NoError(t, err)
	require.Equal(t, string(before), string(after))

	// 验证 ModTime 未更改以确保确实没有发生写入
	stAfter, err := fs.Stat(os.DirFS(baseDir), fileName)
	require.NoError(t, err)
	require.True(t, stAfter.ModTime().Equal(beforeMod), "file ModTime should not change on noop")
}

func TestUpdatePreferredModel_UpdatesRecents(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := &Config{}
	cfg.setDefaults(dir, "")
	cfg.dataConfigDir = filepath.Join(dir, "config.json")

	sel := SelectedModel{Provider: "openai", Model: "gpt-4o"}
	require.NoError(t, cfg.UpdatePreferredModel(SelectedModelTypeSmall, sel))

	// 内存中
	require.Equal(t, sel, cfg.Models[SelectedModelTypeSmall])
	require.Len(t, cfg.RecentModels[SelectedModelTypeSmall], 1)

	// 持久化（通过 fs.FS 读取）
	rm := readRecentModels(t, cfg.dataConfigDir)
	small, ok := rm[string(SelectedModelTypeSmall)].([]any)
	require.True(t, ok)
	require.Len(t, small, 1)
}

func TestRecordRecentModel_TypeIsolation(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := &Config{}
	cfg.setDefaults(dir, "")
	cfg.dataConfigDir = filepath.Join(dir, "config.json")

	// 向 large 和 small 类型添加模型
	largeModel := SelectedModel{Provider: "openai", Model: "gpt-4o"}
	smallModel := SelectedModel{Provider: "anthropic", Model: "claude"}

	require.NoError(t, cfg.recordRecentModel(SelectedModelTypeLarge, largeModel))
	require.NoError(t, cfg.recordRecentModel(SelectedModelTypeSmall, smallModel))

	// 内存中：验证类型维护独立的历史记录
	require.Len(t, cfg.RecentModels[SelectedModelTypeLarge], 1)
	require.Len(t, cfg.RecentModels[SelectedModelTypeSmall], 1)
	require.Equal(t, largeModel, cfg.RecentModels[SelectedModelTypeLarge][0])
	require.Equal(t, smallModel, cfg.RecentModels[SelectedModelTypeSmall][0])

	// 向 large 添加另一个，验证 small 未更改
	anotherLarge := SelectedModel{Provider: "google", Model: "gemini"}
	require.NoError(t, cfg.recordRecentModel(SelectedModelTypeLarge, anotherLarge))

	require.Len(t, cfg.RecentModels[SelectedModelTypeLarge], 2)
	require.Len(t, cfg.RecentModels[SelectedModelTypeSmall], 1)
	require.Equal(t, smallModel, cfg.RecentModels[SelectedModelTypeSmall][0])

	// 持久化状态：验证两种类型都存在且具有正确的长度和内容
	rm := readRecentModels(t, cfg.dataConfigDir)

	large, ok := rm[string(SelectedModelTypeLarge)].([]any)
	require.True(t, ok)
	require.Len(t, large, 2)
	// 验证 large 类型的最新的在前
	require.Equal(t, "google", large[0].(map[string]any)["provider"])
	require.Equal(t, "gemini", large[0].(map[string]any)["model"])
	require.Equal(t, "openai", large[1].(map[string]any)["provider"])
	require.Equal(t, "gpt-4o", large[1].(map[string]any)["model"])

	small, ok := rm[string(SelectedModelTypeSmall)].([]any)
	require.True(t, ok)
	require.Len(t, small, 1)
	require.Equal(t, "anthropic", small[0].(map[string]any)["provider"])
	require.Equal(t, "claude", small[0].(map[string]any)["model"])
}
