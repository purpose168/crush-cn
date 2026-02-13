package app

import (
	"fmt"
	"strings"

	xstrings "github.com/charmbracelet/x/exp/strings"
	"github.com/purpose168/crush-cn/internal/config"
)

// parseModelStr 将模型字符串解析为提供商过滤器和模型 ID。
// 格式："model-name" 或 "provider/model-name" 或 "synthetic/moonshot/kimi-k2"。
// 此函数仅检查第一个组件是否为有效的提供商名称；如果不是，
// 则将整个字符串视为模型 ID（可能包含斜杠）。
func parseModelStr(providers map[string]config.ProviderConfig, modelStr string) (providerFilter, modelID string) {
	parts := strings.Split(modelStr, "/")
	if len(parts) == 1 {
		return "", parts[0]
	}
	// 检查第一部分是否为有效的提供商名称
	if _, ok := providers[parts[0]]; ok {
		return parts[0], strings.Join(parts[1:], "/")
	}

	// 第一部分不是有效的提供商，将整个字符串视为模型 ID
	return "", modelStr
}

// modelMatch 表示找到的模型。
type modelMatch struct {
	provider string
	modelID  string
}

// findModels 查找匹配的大型模型和小型模型。
func findModels(providers map[string]config.ProviderConfig, largeModel, smallModel string) ([]modelMatch, []modelMatch, error) {
	largeProviderFilter, largeModelID := parseModelStr(providers, largeModel)
	smallProviderFilter, smallModelID := parseModelStr(providers, smallModel)

	// 验证提供商过滤器是否存在。
	for _, pf := range []struct {
		filter, label string
	}{
		{largeProviderFilter, "大型"},
		{smallProviderFilter, "小型"},
	} {
		if pf.filter != "" {
			if _, ok := providers[pf.filter]; !ok {
				return nil, nil, fmt.Errorf("%s 模型：提供商 %q 在配置中未找到。使用 'crush models' 列出可用模型", pf.label, pf.filter)
			}
		}
	}

	// 在单次遍历中查找匹配的模型。
	var largeMatches, smallMatches []modelMatch
	for name, provider := range providers {
		if provider.Disable {
			continue
		}
		for _, m := range provider.Models {
			if filter(largeModelID, largeProviderFilter, m.ID, name) {
				largeMatches = append(largeMatches, modelMatch{provider: name, modelID: m.ID})
			}
			if filter(smallModelID, smallProviderFilter, m.ID, name) {
				smallMatches = append(smallMatches, modelMatch{provider: name, modelID: m.ID})
			}
		}
	}

	return largeMatches, smallMatches, nil
}

// filter 检查模型是否匹配给定的过滤器。
func filter(modelFilter, providerFilter, model, provider string) bool {
	return modelFilter != "" && model == modelFilter &&
		(providerFilter == "" || provider == providerFilter)
}

// validateMatches 验证并返回单个匹配项。
func validateMatches(matches []modelMatch, modelID, label string) (modelMatch, error) {
	switch {
	case len(matches) == 0:
		return modelMatch{}, fmt.Errorf("%s 模型 %q 未找到", label, modelID)
	case len(matches) > 1:
		names := make([]string, len(matches))
		for i, m := range matches {
			names[i] = m.provider
		}
		return modelMatch{}, fmt.Errorf(
			"%s 模型：模型 %q 在多个提供商中找到：%s。请使用 'provider/model' 格式指定提供商",
			label,
			modelID,
			xstrings.EnglishJoin(names, true),
		)
	}
	return matches[0], nil
}
