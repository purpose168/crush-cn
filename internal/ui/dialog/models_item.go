package dialog

import (
	"charm.land/catwalk/pkg/catwalk"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/ui/common"
	"github.com/charmbracelet/crush/internal/ui/styles"
	"github.com/charmbracelet/x/ansi"
	"github.com/sahilm/fuzzy"
)

// ModelGroup 表示一组模型项目。
type ModelGroup struct {
	Title      string
	Items      []*ModelItem
	configured bool
	t          *styles.Styles
}

// NewModelGroup 创建一个新的 ModelGroup。
func NewModelGroup(t *styles.Styles, title string, configured bool, items ...*ModelItem) ModelGroup {
	return ModelGroup{
		Title:      title,
		Items:      items,
		configured: configured,
		t:          t,
	}
}

// AppendItems 将 [ModelItem] 追加到组中。
func (m *ModelGroup) AppendItems(items ...*ModelItem) {
	m.Items = append(m.Items, items...)
}

// Render 实现 [list.Item] 接口。
func (m *ModelGroup) Render(width int) string {
	var configured string
	if m.configured {
		configuredIcon := m.t.ToolCallSuccess.Render()
		configuredText := m.t.Subtle.Render("已配置")
		configured = configuredIcon + " " + configuredText
	}

	title := " " + m.Title + " "
	title = ansi.Truncate(title, max(0, width-lipgloss.Width(configured)-1), "…")

	return common.Section(m.t, title, width, configured)
}

// ModelItem 表示模型类型的列表项目。
type ModelItem struct {
	prov      catwalk.Provider
	model     catwalk.Model
	modelType ModelType

	cache        map[int]string
	t            *styles.Styles
	m            fuzzy.Match
	focused      bool
	showProvider bool
}

// SelectedModel 返回此模型项目作为 [config.SelectedModel] 实例。
func (m *ModelItem) SelectedModel() config.SelectedModel {
	return config.SelectedModel{
		Model:           m.model.ID,
		Provider:        string(m.prov.ID),
		ReasoningEffort: m.model.DefaultReasoningEffort,
		MaxTokens:       m.model.DefaultMaxTokens,
	}
}

// SelectedModelType 返回此项目表示的模型类型。
func (m *ModelItem) SelectedModelType() config.SelectedModelType {
	return m.modelType.Config()
}

var _ ListItem = &ModelItem{}

// NewModelItem 创建一个新的 ModelItem。
func NewModelItem(t *styles.Styles, prov catwalk.Provider, model catwalk.Model, typ ModelType, showProvider bool) *ModelItem {
	return &ModelItem{
		prov:         prov,
		model:        model,
		modelType:    typ,
		t:            t,
		cache:        make(map[int]string),
		showProvider: showProvider,
	}
}

// Filter 实现 ListItem 接口。
func (m *ModelItem) Filter() string {
	return m.model.Name
}

// ID 实现 ListItem 接口。
func (m *ModelItem) ID() string {
	return modelKey(string(m.prov.ID), m.model.ID)
}

// Render 实现 ListItem 接口。
func (m *ModelItem) Render(width int) string {
	var providerInfo string
	if m.showProvider {
		providerInfo = string(m.prov.Name)
	}
	styles := ListItemStyles{
		ItemBlurred:     m.t.Dialog.NormalItem,
		ItemFocused:     m.t.Dialog.SelectedItem,
		InfoTextBlurred: m.t.Base,
		InfoTextFocused: m.t.Base,
	}
	return renderItem(styles, m.model.Name, providerInfo, m.focused, width, m.cache, &m.m)
}

// SetFocused 实现 ListItem 接口。
func (m *ModelItem) SetFocused(focused bool) {
	if m.focused != focused {
		m.cache = nil
	}
	m.focused = focused
}

// SetMatch 实现 ListItem 接口。
func (m *ModelItem) SetMatch(fm fuzzy.Match) {
	m.cache = nil
	m.m = fm
}
