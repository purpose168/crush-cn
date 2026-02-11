package dialog

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/catwalk/pkg/catwalk"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/purpose168/crush-cn/internal/config"
	"github.com/purpose168/crush-cn/internal/ui/common"
	"github.com/purpose168/crush-cn/internal/ui/util"
)

// ModelType 表示要选择的模型类型。
type ModelType int

const (
	ModelTypeLarge ModelType = iota
	ModelTypeSmall
)

// String 返回 [ModelType] 的字符串表示。
func (mt ModelType) String() string {
	switch mt {
	case ModelTypeLarge:
		return "大型任务"
	case ModelTypeSmall:
		return "小型任务"
	default:
		return "未知"
	}
}

// Config 返回对应的配置模型类型。
func (mt ModelType) Config() config.SelectedModelType {
	switch mt {
	case ModelTypeLarge:
		return config.SelectedModelTypeLarge
	case ModelTypeSmall:
		return config.SelectedModelTypeSmall
	default:
		return ""
	}
}

// Placeholder 返回模型类型的输入占位符。
func (mt ModelType) Placeholder() string {
	switch mt {
	case ModelTypeLarge:
		return largeModelInputPlaceholder
	case ModelTypeSmall:
		return smallModelInputPlaceholder
	default:
		return ""
	}
}

const (
	onboardingModelInputPlaceholder = "查找您喜欢的"
	largeModelInputPlaceholder      = "为大型复杂任务选择模型"
	smallModelInputPlaceholder      = "为小型简单任务选择模型"
)

// ModelsID 是模型选择对话框的标识符。
const ModelsID = "models"

const defaultModelsDialogMaxWidth = 73

// Models 表示一个模型选择对话框。
type Models struct {
	com          *common.Common
	isOnboarding bool

	modelType ModelType
	providers []catwalk.Provider

	keyMap struct {
		Tab      key.Binding
		UpDown   key.Binding
		Select   key.Binding
		Edit     key.Binding
		Next     key.Binding
		Previous key.Binding
		Close    key.Binding
	}
	list  *ModelsList
	input textinput.Model
	help  help.Model
}

var _ Dialog = (*Models)(nil)

// NewModels 创建一个新的 Models 对话框。
func NewModels(com *common.Common, isOnboarding bool) (*Models, error) {
	t := com.Styles
	m := &Models{}
	m.com = com
	m.isOnboarding = isOnboarding

	help := help.New()
	help.Styles = t.DialogHelpStyles()

	m.help = help
	m.list = NewModelsList(t)
	m.list.Focus()
	m.list.SetSelected(0)

	m.input = textinput.New()
	m.input.SetVirtualCursor(false)
	m.input.Placeholder = onboardingModelInputPlaceholder
	m.input.SetStyles(com.Styles.TextInput)
	m.input.Focus()

	m.keyMap.Tab = key.NewBinding(
		key.WithKeys("tab", "shift+tab"),
		key.WithHelp("tab", "切换类型"),
	)
	m.keyMap.Select = key.NewBinding(
		key.WithKeys("enter", "ctrl+y"),
		key.WithHelp("enter", "确认"),
	)
	m.keyMap.Edit = key.NewBinding(
		key.WithKeys("ctrl+e"),
		key.WithHelp("ctrl+e", "编辑"),
	)
	m.keyMap.UpDown = key.NewBinding(
		key.WithKeys("up", "down"),
		key.WithHelp("↑/↓", "选择"),
	)
	m.keyMap.Next = key.NewBinding(
		key.WithKeys("down", "ctrl+n"),
		key.WithHelp("↓", "下一项"),
	)
	m.keyMap.Previous = key.NewBinding(
		key.WithKeys("up", "ctrl+p"),
		key.WithHelp("↑", "上一项"),
	)
	m.keyMap.Close = CloseKey

	providers, err := getFilteredProviders(com.Config())
	if err != nil {
		return nil, fmt.Errorf("无法获取提供者: %w", err)
	}

	m.providers = providers
	if err := m.setProviderItems(); err != nil {
		return nil, fmt.Errorf("无法设置提供者项目: %w", err)
	}

	return m, nil
}

// ID 实现 Dialog 接口。
func (m *Models) ID() string {
	return ModelsID
}

// HandleMsg 实现 Dialog 接口。
func (m *Models) HandleMsg(msg tea.Msg) Action {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keyMap.Close):
			return ActionClose{}
		case key.Matches(msg, m.keyMap.Previous):
			m.list.Focus()
			if m.list.IsSelectedFirst() {
				m.list.SelectLast()
				m.list.ScrollToBottom()
				break
			}
			m.list.SelectPrev()
			m.list.ScrollToSelected()
		case key.Matches(msg, m.keyMap.Next):
			m.list.Focus()
			if m.list.IsSelectedLast() {
				m.list.SelectFirst()
				m.list.ScrollToTop()
				break
			}
			m.list.SelectNext()
			m.list.ScrollToSelected()
		case key.Matches(msg, m.keyMap.Select, m.keyMap.Edit):
			selectedItem := m.list.SelectedItem()
			if selectedItem == nil {
				break
			}

			modelItem, ok := selectedItem.(*ModelItem)
			if !ok {
				break
			}

			isEdit := key.Matches(msg, m.keyMap.Edit)

			return ActionSelectModel{
				Provider:       modelItem.prov,
				Model:          modelItem.SelectedModel(),
				ModelType:      modelItem.SelectedModelType(),
				ReAuthenticate: isEdit,
			}
		case key.Matches(msg, m.keyMap.Tab):
			if m.isOnboarding {
				break
			}
			if m.modelType == ModelTypeLarge {
				m.modelType = ModelTypeSmall
			} else {
				m.modelType = ModelTypeLarge
			}
			if err := m.setProviderItems(); err != nil {
				return util.ReportError(err)
			}
		default:
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			value := m.input.Value()
			m.list.Focus()
			m.list.SetFilter(value)
			m.list.SelectFirst()
			m.list.ScrollToTop()
			return ActionCmd{cmd}
		}
	}
	return nil
}

// Cursor 返回对话框的光标。
func (m *Models) Cursor() *tea.Cursor {
	return InputCursor(m.com.Styles, m.input.Cursor())
}

// modelTypeRadioView 返回模型类型选择的单选视图。
func (m *Models) modelTypeRadioView() string {
	t := m.com.Styles
	textStyle := t.HalfMuted
	largeRadioStyle := t.RadioOff
	smallRadioStyle := t.RadioOff
	if m.modelType == ModelTypeLarge {
		largeRadioStyle = t.RadioOn
	} else {
		smallRadioStyle = t.RadioOn
	}

	largeRadio := largeRadioStyle.Padding(0, 1).Render()
	smallRadio := smallRadioStyle.Padding(0, 1).Render()

	return fmt.Sprintf("%s%s  %s%s",
		largeRadio, textStyle.Render(ModelTypeLarge.String()),
		smallRadio, textStyle.Render(ModelTypeSmall.String()))
}

// Draw 实现 [Dialog] 接口。
func (m *Models) Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor {
	t := m.com.Styles
	width := max(0, min(defaultModelsDialogMaxWidth, area.Dx()-t.Dialog.View.GetHorizontalBorderSize()))
	height := max(0, min(defaultDialogHeight, area.Dy()-t.Dialog.View.GetVerticalBorderSize()))
	innerWidth := width - t.Dialog.View.GetHorizontalFrameSize()
	heightOffset := t.Dialog.Title.GetVerticalFrameSize() + titleContentHeight +
		t.Dialog.InputPrompt.GetVerticalFrameSize() + inputContentHeight +
		t.Dialog.HelpView.GetVerticalFrameSize() +
		t.Dialog.View.GetVerticalFrameSize()

	m.input.SetWidth(max(0, innerWidth-t.Dialog.InputPrompt.GetHorizontalFrameSize()-1)) // (1) cursor padding
	m.list.SetSize(innerWidth, height-heightOffset)
	m.help.SetWidth(innerWidth)

	rc := NewRenderContext(t, width)
	rc.Title = "切换模型"
	rc.TitleInfo = m.modelTypeRadioView()

	if m.isOnboarding {
		titleText := t.Dialog.PrimaryText.Render("要开始，让我们选择一个提供者和模型。")
		rc.AddPart(titleText)
	}

	inputView := t.Dialog.InputPrompt.Render(m.input.View())
	rc.AddPart(inputView)

	listView := t.Dialog.List.Height(m.list.Height()).Render(m.list.Render())
	rc.AddPart(listView)

	rc.Help = m.help.View(m)

	cur := m.Cursor()

	if m.isOnboarding {
		rc.Title = ""
		rc.TitleInfo = ""
		rc.IsOnboarding = true
		view := rc.Render()
		DrawOnboardingCursor(scr, area, view, cur)

		// FIXME(@andreynering): 找出如何正确修复这个问题
		if cur != nil {
			cur.Y -= 1
			cur.X -= 1
		}
	} else {
		view := rc.Render()
		DrawCenterCursor(scr, area, view, cur)
	}
	return cur
}

// ShortHelp 返回简短的帮助视图。
func (m *Models) ShortHelp() []key.Binding {
	if m.isOnboarding {
		return []key.Binding{
			m.keyMap.UpDown,
			m.keyMap.Select,
		}
	}
	h := []key.Binding{
		m.keyMap.UpDown,
		m.keyMap.Tab,
		m.keyMap.Select,
	}
	if m.isSelectedConfigured() {
		h = append(h, m.keyMap.Edit)
	}
	h = append(h, m.keyMap.Close)
	return h
}

// FullHelp 返回完整的帮助视图。
func (m *Models) FullHelp() [][]key.Binding {
	return [][]key.Binding{m.ShortHelp()}
}

func (m *Models) isSelectedConfigured() bool {
	selectedItem := m.list.SelectedItem()
	if selectedItem == nil {
		return false
	}
	modelItem, ok := selectedItem.(*ModelItem)
	if !ok {
		return false
	}
	providerID := string(modelItem.prov.ID)
	_, isConfigured := m.com.Config().Providers.Get(providerID)
	return isConfigured
}

// setProviderItems 在列表中设置提供者项目。
func (m *Models) setProviderItems() error {
	t := m.com.Styles
	cfg := m.com.Config()

	var selectedItemID string
	selectedType := m.modelType.Config()
	currentModel := cfg.Models[selectedType]
	recentItems := cfg.RecentModels[selectedType]

	// 跟踪已添加的提供者以避免重复
	addedProviders := make(map[string]bool)

	// 获取已知提供者列表以进行比较
	knownProviders, err := config.Providers(cfg)
	if err != nil {
		return fmt.Errorf("无法获取提供者: %w", err)
	}

	containsProviderFunc := func(id string) func(p catwalk.Provider) bool {
		return func(p catwalk.Provider) bool {
			return p.ID == catwalk.InferenceProvider(id)
		}
	}

	// itemsMap 包含已添加的模型项目的键。
	itemsMap := make(map[string]*ModelItem)
	groups := []ModelGroup{}
	for id, p := range cfg.Providers.Seq2() {
		if p.Disable {
			continue
		}

		// 检查此提供者是否不在已知提供者列表中
		if !slices.ContainsFunc(knownProviders, containsProviderFunc(id)) ||
			!slices.ContainsFunc(m.providers, containsProviderFunc(id)) {
			provider := p.ToProvider()

			// 将此未知提供者添加到列表
			name := cmp.Or(p.Name, id)

			addedProviders[id] = true

			group := NewModelGroup(t, name, true)
			for _, model := range p.Models {
				item := NewModelItem(t, provider, model, m.modelType, false)
				group.AppendItems(item)
				itemsMap[item.ID()] = item
				if model.ID == currentModel.Model && string(provider.ID) == currentModel.Provider {
					selectedItemID = item.ID()
				}
			}
			if len(group.Items) > 0 {
				groups = append(groups, group)
			}
		}
	}

	// 将"Charm Hyper"移动到第一个位置。
	// （但仍在最近使用的模型和自定义提供者之后）。
	slices.SortStableFunc(m.providers, func(a, b catwalk.Provider) int {
		switch {
		case a.ID == "hyper":
			return -1
		case b.ID == "hyper":
			return 1
		default:
			return 0
		}
	})

	// 现在从预定义列表中添加已知提供者
	for _, provider := range m.providers {
		providerID := string(provider.ID)
		if addedProviders[providerID] {
			continue
		}

		providerConfig, providerConfigured := cfg.Providers.Get(providerID)
		if providerConfigured && providerConfig.Disable {
			continue
		}

		displayProvider := provider
		if providerConfigured {
			displayProvider.Name = cmp.Or(providerConfig.Name, displayProvider.Name)
			modelIndex := make(map[string]int, len(displayProvider.Models))
			for i, model := range displayProvider.Models {
				modelIndex[model.ID] = i
			}
			for _, model := range providerConfig.Models {
				if model.ID == "" {
					continue
				}
				if idx, ok := modelIndex[model.ID]; ok {
					if model.Name != "" {
						displayProvider.Models[idx].Name = model.Name
					}
					continue
				}
				if model.Name == "" {
					model.Name = model.ID
				}
				displayProvider.Models = append(displayProvider.Models, model)
				modelIndex[model.ID] = len(displayProvider.Models) - 1
			}
		}

		name := displayProvider.Name
		if name == "" {
			name = providerID
		}

		group := NewModelGroup(t, name, providerConfigured)
		for _, model := range displayProvider.Models {
			item := NewModelItem(t, provider, model, m.modelType, false)
			group.AppendItems(item)
			itemsMap[item.ID()] = item
			if model.ID == currentModel.Model && string(provider.ID) == currentModel.Provider {
				selectedItemID = item.ID()
			}
		}

		groups = append(groups, group)
	}

	if len(recentItems) > 0 {
		recentGroup := NewModelGroup(t, "最近使用", false)

		var validRecentItems []config.SelectedModel
		for _, recent := range recentItems {
			key := modelKey(recent.Provider, recent.Model)
			item, ok := itemsMap[key]
			if !ok {
				continue
			}

			// 显示最近项目的提供者
			item = NewModelItem(t, item.prov, item.model, m.modelType, true)
			item.showProvider = true

			validRecentItems = append(validRecentItems, recent)
			recentGroup.AppendItems(item)
			if recent.Model == currentModel.Model && recent.Provider == currentModel.Provider {
				selectedItemID = item.ID()
			}
		}

		if len(validRecentItems) != len(recentItems) {
			// FIXME: 这需要在这里吗？这是在读取期间修改配置吗？
			if err := cfg.SetConfigField(fmt.Sprintf("recent_models.%s", selectedType), validRecentItems); err != nil {
				return fmt.Errorf("无法更新最近模型: %w", err)
			}
		}

		if len(recentGroup.Items) > 0 {
			groups = append([]ModelGroup{recentGroup}, groups...)
		}
	}

	// 在列表中设置模型组。
	m.list.SetGroups(groups...)
	m.list.SetSelectedItem(selectedItemID)
	m.list.ScrollToTop()

	// 根据模型类型更新占位符
	if !m.isOnboarding {
		m.input.Placeholder = m.modelType.Placeholder()
	}

	return nil
}

func getFilteredProviders(cfg *config.Config) ([]catwalk.Provider, error) {
	providers, err := config.Providers(cfg)
	if err != nil {
		return nil, fmt.Errorf("无法获取提供者: %w", err)
	}
	var filteredProviders []catwalk.Provider
	for _, p := range providers {
		var (
			isAzure         = p.ID == catwalk.InferenceProviderAzure
			isCopilot       = p.ID == catwalk.InferenceProviderCopilot
			isHyper         = string(p.ID) == "hyper"
			hasAPIKeyEnv    = strings.HasPrefix(p.APIKey, "$")
			_, isConfigured = cfg.Providers.Get(string(p.ID))
		)
		if isAzure || isCopilot || isHyper || hasAPIKeyEnv || isConfigured {
			filteredProviders = append(filteredProviders, p)
		}
	}
	return filteredProviders, nil
}

func modelKey(providerID, modelID string) string {
	if providerID == "" || modelID == "" {
		return ""
	}
	return providerID + ":" + modelID
}
