package styles

import (
	"image/color"
	"strings"

	"charm.land/bubbles/v2/filepicker"
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2/ansi"
	"charm.land/lipgloss/v2"
	"github.com/alecthomas/chroma/v2"
	"github.com/charmbracelet/crush/internal/ui/diffview"
	"github.com/charmbracelet/x/exp/charmtone"
)

// 图标常量定义
const (
	CheckIcon   string = "✓" // 勾选图标
	SpinnerIcon string = "⋯" // 加载中图标
	LoadingIcon string = "⟳" // 刷新图标
	ModelIcon   string = "◇" // 模型图标

	ArrowRightIcon string = "→" // 右箭头图标

	ToolPending string = "●" // 工具等待状态图标
	ToolSuccess string = "✓" // 工具成功状态图标
	ToolError   string = "×" // 工具错误状态图标

	RadioOn  string = "◉" // 单选按钮选中状态
	RadioOff string = "○" // 单选按钮未选中状态

	BorderThin  string = "│" // 细边框
	BorderThick string = "▌" // 粗边框

	SectionSeparator string = "─" // 节分隔线

	TodoCompletedIcon  string = "✓" // 待办事项完成图标
	TodoPendingIcon    string = "•" // 待办事项待处理图标
	TodoInProgressIcon string = "→" // 待办事项进行中图标

	ImageIcon string = "■" // 图片图标
	TextIcon  string = "≡" // 文本图标

	ScrollbarThumb string = "┃" // 滚动条滑块
	ScrollbarTrack string = "│" // 滚动条轨道

	LSPErrorIcon   string = "E" // LSP错误图标
	LSPWarningIcon string = "W" // LSP警告图标
	LSPInfoIcon    string = "I" // LSP信息图标
	LSPHintIcon    string = "H" // LSP提示图标
)

// 默认样式常量
const (
	defaultMargin     = 2 // 默认边距
	defaultListIndent = 2 // 默认列表缩进
)

// Styles 定义了UI的所有样式

type Styles struct {
	WindowTooSmall lipgloss.Style // 窗口过小时的提示样式

	// 可复用文本样式
	Base      lipgloss.Style // 基础文本样式
	Muted     lipgloss.Style // 弱化文本样式
	HalfMuted lipgloss.Style // 半弱化文本样式
	Subtle    lipgloss.Style // 细微文本样式

	// 标签样式
	TagBase  lipgloss.Style // 基础标签样式
	TagError lipgloss.Style // 错误标签样式
	TagInfo  lipgloss.Style // 信息标签样式

	// 头部样式
	Header struct {
		Charm        lipgloss.Style // "Charm™" 标签样式
		Diagonals    lipgloss.Style // 对角线分隔符样式 (╱)
		Percentage   lipgloss.Style // 上下文百分比样式
		Keystroke    lipgloss.Style // 快捷键提示样式 (例如: "ctrl+d")
		KeystrokeTip lipgloss.Style // 快捷键操作文本样式 (例如: "打开", "关闭")
		WorkingDir   lipgloss.Style // 当前工作目录样式
		Separator    lipgloss.Style // 分隔点样式 (•)
	}

	// 紧凑详情样式
	CompactDetails struct {
		View    lipgloss.Style // 视图样式
		Version lipgloss.Style // 版本信息样式
		Title   lipgloss.Style // 标题样式
	}

	// 面板样式
	PanelMuted lipgloss.Style // 弱化面板样式
	PanelBase  lipgloss.Style // 基础面板样式

	// 代码块行号样式
	LineNumber lipgloss.Style

	// 消息边框
	FocusedMessageBorder lipgloss.Border // 聚焦消息边框

	// 工具调用样式
	ToolCallPending   lipgloss.Style // 工具调用等待样式
	ToolCallError     lipgloss.Style // 工具调用错误样式
	ToolCallSuccess   lipgloss.Style // 工具调用成功样式
	ToolCallCancelled lipgloss.Style // 工具调用取消样式
	EarlyStateMessage lipgloss.Style // 早期状态消息样式

	// 文本选择样式
	TextSelection lipgloss.Style

	// LSP和MCP状态指示器
	ItemOfflineIcon lipgloss.Style // 离线状态图标样式
	ItemBusyIcon    lipgloss.Style // 忙碌状态图标样式
	ItemErrorIcon   lipgloss.Style // 错误状态图标样式
	ItemOnlineIcon  lipgloss.Style // 在线状态图标样式

	// Markdown和Chroma样式
	Markdown      ansi.StyleConfig // Markdown样式配置
	PlainMarkdown ansi.StyleConfig // 纯文本Markdown样式配置

	// 输入框样式
	TextInput textinput.Styles // 文本输入框样式
	TextArea  textarea.Styles  // 文本区域样式

	// 帮助样式
	Help help.Styles // 帮助文本样式

	// 差异视图样式
	Diff diffview.Style // 差异视图样式

	// 文件选择器样式
	FilePicker filepicker.Styles // 文件选择器样式

	// 按钮样式
	ButtonFocus lipgloss.Style // 按钮聚焦样式
	ButtonBlur  lipgloss.Style // 按钮失焦样式

	// 边框样式
	BorderFocus lipgloss.Style // 边框聚焦样式
	BorderBlur  lipgloss.Style // 边框失焦样式

	// 编辑器提示样式
	EditorPromptNormalFocused   lipgloss.Style // 普通编辑器提示聚焦样式
	EditorPromptNormalBlurred   lipgloss.Style // 普通编辑器提示失焦样式
	EditorPromptYoloIconFocused lipgloss.Style // Yolo编辑器提示图标聚焦样式
	EditorPromptYoloIconBlurred lipgloss.Style // Yolo编辑器提示图标失焦样式
	EditorPromptYoloDotsFocused lipgloss.Style // Yolo编辑器提示点聚焦样式
	EditorPromptYoloDotsBlurred lipgloss.Style // Yolo编辑器提示点失焦样式

	// 单选按钮样式
	RadioOn  lipgloss.Style // 单选按钮选中样式
	RadioOff lipgloss.Style // 单选按钮未选中样式

	// 背景颜色
	Background color.Color

	// Logo样式
	LogoFieldColor   color.Color // Logo字段颜色
	LogoTitleColorA  color.Color // Logo标题颜色A
	LogoTitleColorB  color.Color // Logo标题颜色B
	LogoCharmColor   color.Color // Logo Charm颜色
	LogoVersionColor color.Color // Logo版本颜色

	// 语义颜色，用于工具渲染
	Primary       color.Color // 主色调
	Secondary     color.Color // 辅助色调
	Tertiary      color.Color // 第三色调
	BgBase        color.Color // 基础背景色
	BgBaseLighter color.Color // 浅基础背景色
	BgSubtle      color.Color // 细微背景色
	BgOverlay     color.Color // 覆盖层背景色
	FgBase        color.Color // 基础前景色
	FgMuted       color.Color // 弱化前景色
	FgHalfMuted   color.Color // 半弱化前景色
	FgSubtle      color.Color // 细微前景色
	Border        color.Color // 边框颜色
	BorderColor   color.Color // 边框聚焦颜色
	Error         color.Color // 错误颜色
	Warning       color.Color // 警告颜色
	Info          color.Color // 信息颜色
	White         color.Color // 白色
	BlueLight     color.Color // 浅蓝色
	Blue          color.Color // 蓝色
	BlueDark      color.Color // 深蓝色
	GreenLight    color.Color // 浅绿色
	Green         color.Color // 绿色
	GreenDark     color.Color // 深绿色
	Red           color.Color // 红色
	RedDark       color.Color // 深红色
	Yellow        color.Color // 黄色

	// 节标题样式
	Section struct {
		Title lipgloss.Style // 节标题样式
		Line  lipgloss.Style // 节分隔线样式
	}

	// 初始化样式
	Initialize struct {
		Header  lipgloss.Style // 初始化头部样式
		Content lipgloss.Style // 初始化内容样式
		Accent  lipgloss.Style // 初始化强调样式
	}

	// LSP样式
	LSP struct {
		ErrorDiagnostic   lipgloss.Style // LSP错误诊断样式
		WarningDiagnostic lipgloss.Style // LSP警告诊断样式
		HintDiagnostic    lipgloss.Style // LSP提示诊断样式
		InfoDiagnostic    lipgloss.Style // LSP信息诊断样式
	}

	// 文件样式
	Files struct {
		Path      lipgloss.Style // 文件路径样式
		Additions lipgloss.Style // 文件新增内容样式
		Deletions lipgloss.Style // 文件删除内容样式
	}

	// 聊天样式
	Chat struct {
		// 消息项样式
		Message struct {
			UserBlurred      lipgloss.Style // 用户消息失焦样式
			UserFocused      lipgloss.Style // 用户消息聚焦样式
			AssistantBlurred lipgloss.Style // 助手消息失焦样式
			AssistantFocused lipgloss.Style // 助手消息聚焦样式
			NoContent        lipgloss.Style // 无内容消息样式
			Thinking         lipgloss.Style // 思考中消息样式
			ErrorTag         lipgloss.Style // 错误标签样式
			ErrorTitle       lipgloss.Style // 错误标题样式
			ErrorDetails     lipgloss.Style // 错误详情样式
			ToolCallFocused  lipgloss.Style // 工具调用聚焦样式
			ToolCallCompact  lipgloss.Style // 工具调用紧凑样式
			ToolCallBlurred  lipgloss.Style // 工具调用失焦样式
			SectionHeader    lipgloss.Style // 节标题样式

			// 思考节样式
			ThinkingBox            lipgloss.Style // 思考内容背景样式
			ThinkingTruncationHint lipgloss.Style // "… (N行隐藏)" 提示样式
			ThinkingFooterTitle    lipgloss.Style // "思考于" 文本样式
			ThinkingFooterDuration lipgloss.Style // 时长值样式
			AssistantInfoIcon      lipgloss.Style // 助手信息图标样式
			AssistantInfoModel     lipgloss.Style // 助手信息模型样式
			AssistantInfoProvider  lipgloss.Style // 助手信息提供者样式
			AssistantInfoDuration  lipgloss.Style // 助手信息时长样式
		}
	}

	// 工具调用渲染样式
	Tool struct {
		// 带状态的图标样式
		IconPending   lipgloss.Style // 等待操作图标样式
		IconSuccess   lipgloss.Style // 成功操作图标样式
		IconError     lipgloss.Style // 错误操作图标样式
		IconCancelled lipgloss.Style // 取消操作图标样式

		// 工具名称样式
		NameNormal lipgloss.Style // 普通工具名称样式
		NameNested lipgloss.Style // 嵌套工具名称样式

		// 参数列表样式
		ParamMain lipgloss.Style // 主要参数样式
		ParamKey  lipgloss.Style // 参数键样式

		// 内容渲染样式
		ContentLine           lipgloss.Style // 带背景和宽度的单个内容行样式
		ContentTruncation     lipgloss.Style // 截断消息 "… (N行)" 样式
		ContentCodeLine       lipgloss.Style // 带背景和宽度的代码行样式
		ContentCodeTruncation lipgloss.Style // 带bgBase的代码截断消息样式
		ContentCodeBg         color.Color    // 语法高亮背景色
		Body                  lipgloss.Style // 主体内容内边距样式 (左内边距2)

		// 已弃用 - 为向后兼容性保留
		ContentBg         lipgloss.Style // 内容背景样式
		ContentText       lipgloss.Style // 内容文本样式
		ContentLineNumber lipgloss.Style // 代码行号样式

		// 状态消息样式
		StateWaiting   lipgloss.Style // "等待工具响应..." 样式
		StateCancelled lipgloss.Style // "已取消。" 样式

		// 错误样式
		ErrorTag     lipgloss.Style // ERROR标签样式
		ErrorMessage lipgloss.Style // 错误消息文本样式

		// 差异样式
		DiffTruncation lipgloss.Style // 带内边距的差异截断消息样式

		// 多编辑注释样式
		NoteTag     lipgloss.Style // NOTE标签样式 (黄色背景)
		NoteMessage lipgloss.Style // 注释消息文本样式

		// Bash作业头部样式
		JobIconPending lipgloss.Style // 等待作业图标样式 (深绿色)
		JobIconError   lipgloss.Style // 错误作业图标样式 (深红色)
		JobIconSuccess lipgloss.Style // 成功作业图标样式 (绿色)
		JobToolName    lipgloss.Style // 作业工具名称 "Bash" 样式 (蓝色)
		JobAction      lipgloss.Style // 操作文本样式 (启动、输出、终止)
		JobPID         lipgloss.Style // PID文本样式
		JobDescription lipgloss.Style // 描述文本样式

		// 代理任务样式
		AgentTaskTag lipgloss.Style // 代理任务标签样式 (蓝色背景, 粗体)
		AgentPrompt  lipgloss.Style // 代理提示文本样式

		// 代理获取样式
		AgenticFetchPromptTag lipgloss.Style // 代理获取提示标签样式 (绿色背景, 粗体)

		// 待办事项样式
		TodoRatio          lipgloss.Style // 待办事项比率样式 (例如: "2/5")
		TodoCompletedIcon  lipgloss.Style // 已完成待办事项图标样式
		TodoInProgressIcon lipgloss.Style // 进行中待办事项图标样式
		TodoPendingIcon    lipgloss.Style // 待处理待办事项图标样式

		// MCP工具样式
		MCPName     lipgloss.Style // MCP名称样式
		MCPToolName lipgloss.Style // MCP工具名称样式
		MCPArrow    lipgloss.Style // MCP箭头图标样式
	}

	// Dialog styles
	Dialog struct {
		Title       lipgloss.Style
		TitleText   lipgloss.Style
		TitleError  lipgloss.Style
		TitleAccent lipgloss.Style
		// View is the main content area style.
		View          lipgloss.Style
		PrimaryText   lipgloss.Style
		SecondaryText lipgloss.Style
		// HelpView is the line that contains the help.
		HelpView lipgloss.Style
		Help     struct {
			Ellipsis       lipgloss.Style
			ShortKey       lipgloss.Style
			ShortDesc      lipgloss.Style
			ShortSeparator lipgloss.Style
			FullKey        lipgloss.Style
			FullDesc       lipgloss.Style
			FullSeparator  lipgloss.Style
		}

		NormalItem   lipgloss.Style
		SelectedItem lipgloss.Style
		InputPrompt  lipgloss.Style

		List lipgloss.Style

		Spinner lipgloss.Style

		// ContentPanel is used for content blocks with subtle background.
		ContentPanel lipgloss.Style

		// Scrollbar styles for scrollable content.
		ScrollbarThumb lipgloss.Style
		ScrollbarTrack lipgloss.Style

		// Arguments
		Arguments struct {
			Content                  lipgloss.Style
			Description              lipgloss.Style
			InputLabelBlurred        lipgloss.Style
			InputLabelFocused        lipgloss.Style
			InputRequiredMarkBlurred lipgloss.Style
			InputRequiredMarkFocused lipgloss.Style
		}

		Commands struct{}

		ImagePreview lipgloss.Style

		Sessions struct {
			// styles for when we are in delete mode
			DeletingView                   lipgloss.Style
			DeletingItemFocused            lipgloss.Style
			DeletingItemBlurred            lipgloss.Style
			DeletingTitle                  lipgloss.Style
			DeletingMessage                lipgloss.Style
			DeletingTitleGradientFromColor color.Color
			DeletingTitleGradientToColor   color.Color

			// styles for when we are in update mode
			RenamingView                   lipgloss.Style
			RenamingingItemFocused         lipgloss.Style
			RenamingItemBlurred            lipgloss.Style
			RenamingingTitle               lipgloss.Style
			RenamingingMessage             lipgloss.Style
			RenamingTitleGradientFromColor color.Color
			RenamingTitleGradientToColor   color.Color
			RenamingPlaceholder            lipgloss.Style
		}
	}

	// Status bar and help
	Status struct {
		Help lipgloss.Style

		ErrorIndicator   lipgloss.Style
		WarnIndicator    lipgloss.Style
		InfoIndicator    lipgloss.Style
		UpdateIndicator  lipgloss.Style
		SuccessIndicator lipgloss.Style

		ErrorMessage   lipgloss.Style
		WarnMessage    lipgloss.Style
		InfoMessage    lipgloss.Style
		UpdateMessage  lipgloss.Style
		SuccessMessage lipgloss.Style
	}

	// Completions popup styles
	Completions struct {
		Normal  lipgloss.Style
		Focused lipgloss.Style
		Match   lipgloss.Style
	}

	// Attachments styles
	Attachments struct {
		Normal   lipgloss.Style
		Image    lipgloss.Style
		Text     lipgloss.Style
		Deleting lipgloss.Style
	}

	// Pills styles for todo/queue pills
	Pills struct {
		Base            lipgloss.Style // Base pill style with padding
		Focused         lipgloss.Style // Focused pill with visible border
		Blurred         lipgloss.Style // Blurred pill with hidden border
		QueueItemPrefix lipgloss.Style // Prefix for queue list items
		HelpKey         lipgloss.Style // Keystroke hint style
		HelpText        lipgloss.Style // Help action text style
		Area            lipgloss.Style // Pills area container
		TodoSpinner     lipgloss.Style // Todo spinner style
	}
}

// ChromaTheme converts the current markdown chroma styles to a chroma
// StyleEntries map.
func (s *Styles) ChromaTheme() chroma.StyleEntries {
	rules := s.Markdown.CodeBlock

	return chroma.StyleEntries{
		chroma.Text:                chromaStyle(rules.Chroma.Text),
		chroma.Error:               chromaStyle(rules.Chroma.Error),
		chroma.Comment:             chromaStyle(rules.Chroma.Comment),
		chroma.CommentPreproc:      chromaStyle(rules.Chroma.CommentPreproc),
		chroma.Keyword:             chromaStyle(rules.Chroma.Keyword),
		chroma.KeywordReserved:     chromaStyle(rules.Chroma.KeywordReserved),
		chroma.KeywordNamespace:    chromaStyle(rules.Chroma.KeywordNamespace),
		chroma.KeywordType:         chromaStyle(rules.Chroma.KeywordType),
		chroma.Operator:            chromaStyle(rules.Chroma.Operator),
		chroma.Punctuation:         chromaStyle(rules.Chroma.Punctuation),
		chroma.Name:                chromaStyle(rules.Chroma.Name),
		chroma.NameBuiltin:         chromaStyle(rules.Chroma.NameBuiltin),
		chroma.NameTag:             chromaStyle(rules.Chroma.NameTag),
		chroma.NameAttribute:       chromaStyle(rules.Chroma.NameAttribute),
		chroma.NameClass:           chromaStyle(rules.Chroma.NameClass),
		chroma.NameConstant:        chromaStyle(rules.Chroma.NameConstant),
		chroma.NameDecorator:       chromaStyle(rules.Chroma.NameDecorator),
		chroma.NameException:       chromaStyle(rules.Chroma.NameException),
		chroma.NameFunction:        chromaStyle(rules.Chroma.NameFunction),
		chroma.NameOther:           chromaStyle(rules.Chroma.NameOther),
		chroma.Literal:             chromaStyle(rules.Chroma.Literal),
		chroma.LiteralNumber:       chromaStyle(rules.Chroma.LiteralNumber),
		chroma.LiteralDate:         chromaStyle(rules.Chroma.LiteralDate),
		chroma.LiteralString:       chromaStyle(rules.Chroma.LiteralString),
		chroma.LiteralStringEscape: chromaStyle(rules.Chroma.LiteralStringEscape),
		chroma.GenericDeleted:      chromaStyle(rules.Chroma.GenericDeleted),
		chroma.GenericEmph:         chromaStyle(rules.Chroma.GenericEmph),
		chroma.GenericInserted:     chromaStyle(rules.Chroma.GenericInserted),
		chroma.GenericStrong:       chromaStyle(rules.Chroma.GenericStrong),
		chroma.GenericSubheading:   chromaStyle(rules.Chroma.GenericSubheading),
		chroma.Background:          chromaStyle(rules.Chroma.Background),
	}
}

// DialogHelpStyles returns the styles for dialog help.
func (s *Styles) DialogHelpStyles() help.Styles {
	return help.Styles(s.Dialog.Help)
}

// DefaultStyles returns the default styles for the UI.
func DefaultStyles() Styles {
	var (
		primary   = charmtone.Charple
		secondary = charmtone.Dolly
		tertiary  = charmtone.Bok
		// accent    = charmtone.Zest

		// Backgrounds
		bgBase        = charmtone.Pepper
		bgBaseLighter = charmtone.BBQ
		bgSubtle      = charmtone.Charcoal
		bgOverlay     = charmtone.Iron

		// Foregrounds
		fgBase      = charmtone.Ash
		fgMuted     = charmtone.Squid
		fgHalfMuted = charmtone.Smoke
		fgSubtle    = charmtone.Oyster
		// fgSelected  = charmtone.Salt

		// Borders
		border      = charmtone.Charcoal
		borderFocus = charmtone.Charple

		// Status
		error   = charmtone.Sriracha
		warning = charmtone.Zest
		info    = charmtone.Malibu

		// Colors
		white = charmtone.Butter

		blueLight = charmtone.Sardine
		blue      = charmtone.Malibu
		blueDark  = charmtone.Damson

		// yellow = charmtone.Mustard
		yellow = charmtone.Mustard
		// citron = charmtone.Citron

		greenLight = charmtone.Bok
		green      = charmtone.Julep
		greenDark  = charmtone.Guac
		// greenLight = charmtone.Bok

		red     = charmtone.Coral
		redDark = charmtone.Sriracha
		// redLight = charmtone.Salmon
		// cherry   = charmtone.Cherry
	)

	normalBorder := lipgloss.NormalBorder()

	base := lipgloss.NewStyle().Foreground(fgBase)

	s := Styles{}

	s.Background = bgBase

	// Populate color fields
	s.Primary = primary
	s.Secondary = secondary
	s.Tertiary = tertiary
	s.BgBase = bgBase
	s.BgBaseLighter = bgBaseLighter
	s.BgSubtle = bgSubtle
	s.BgOverlay = bgOverlay
	s.FgBase = fgBase
	s.FgMuted = fgMuted
	s.FgHalfMuted = fgHalfMuted
	s.FgSubtle = fgSubtle
	s.Border = border
	s.BorderColor = borderFocus
	s.Error = error
	s.Warning = warning
	s.Info = info
	s.White = white
	s.BlueLight = blueLight
	s.Blue = blue
	s.BlueDark = blueDark
	s.GreenLight = greenLight
	s.Green = green
	s.GreenDark = greenDark
	s.Red = red
	s.RedDark = redDark
	s.Yellow = yellow

	s.TextInput = textinput.Styles{
		Focused: textinput.StyleState{
			Text:        base,
			Placeholder: base.Foreground(fgSubtle),
			Prompt:      base.Foreground(tertiary),
			Suggestion:  base.Foreground(fgSubtle),
		},
		Blurred: textinput.StyleState{
			Text:        base.Foreground(fgMuted),
			Placeholder: base.Foreground(fgSubtle),
			Prompt:      base.Foreground(fgMuted),
			Suggestion:  base.Foreground(fgSubtle),
		},
		Cursor: textinput.CursorStyle{
			Color: secondary,
			Shape: tea.CursorBlock,
			Blink: true,
		},
	}

	s.TextArea = textarea.Styles{
		Focused: textarea.StyleState{
			Base:             base,
			Text:             base,
			LineNumber:       base.Foreground(fgSubtle),
			CursorLine:       base,
			CursorLineNumber: base.Foreground(fgSubtle),
			Placeholder:      base.Foreground(fgSubtle),
			Prompt:           base.Foreground(tertiary),
		},
		Blurred: textarea.StyleState{
			Base:             base,
			Text:             base.Foreground(fgMuted),
			LineNumber:       base.Foreground(fgMuted),
			CursorLine:       base,
			CursorLineNumber: base.Foreground(fgMuted),
			Placeholder:      base.Foreground(fgSubtle),
			Prompt:           base.Foreground(fgMuted),
		},
		Cursor: textarea.CursorStyle{
			Color: secondary,
			Shape: tea.CursorBlock,
			Blink: true,
		},
	}

	s.Markdown = ansi.StyleConfig{
		Document: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				// BlockPrefix: "\n",
				// BlockSuffix: "\n",
				Color: stringPtr(charmtone.Smoke.Hex()),
			},
			// Margin: uintPtr(defaultMargin),
		},
		BlockQuote: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{},
			Indent:         uintPtr(1),
			IndentToken:    stringPtr("│ "),
		},
		List: ansi.StyleList{
			LevelIndent: defaultListIndent,
		},
		Heading: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				BlockSuffix: "\n",
				Color:       stringPtr(charmtone.Malibu.Hex()),
				Bold:        boolPtr(true),
			},
		},
		H1: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix:          " ",
				Suffix:          " ",
				Color:           stringPtr(charmtone.Zest.Hex()),
				BackgroundColor: stringPtr(charmtone.Charple.Hex()),
				Bold:            boolPtr(true),
			},
		},
		H2: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix: "## ",
			},
		},
		H3: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix: "### ",
			},
		},
		H4: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix: "#### ",
			},
		},
		H5: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix: "##### ",
			},
		},
		H6: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix: "###### ",
				Color:  stringPtr(charmtone.Guac.Hex()),
				Bold:   boolPtr(false),
			},
		},
		Strikethrough: ansi.StylePrimitive{
			CrossedOut: boolPtr(true),
		},
		Emph: ansi.StylePrimitive{
			Italic: boolPtr(true),
		},
		Strong: ansi.StylePrimitive{
			Bold: boolPtr(true),
		},
		HorizontalRule: ansi.StylePrimitive{
			Color:  stringPtr(charmtone.Charcoal.Hex()),
			Format: "\n--------\n",
		},
		Item: ansi.StylePrimitive{
			BlockPrefix: "• ",
		},
		Enumeration: ansi.StylePrimitive{
			BlockPrefix: ". ",
		},
		Task: ansi.StyleTask{
			StylePrimitive: ansi.StylePrimitive{},
			Ticked:         "[✓] ",
			Unticked:       "[ ] ",
		},
		Link: ansi.StylePrimitive{
			Color:     stringPtr(charmtone.Zinc.Hex()),
			Underline: boolPtr(true),
		},
		LinkText: ansi.StylePrimitive{
			Color: stringPtr(charmtone.Guac.Hex()),
			Bold:  boolPtr(true),
		},
		Image: ansi.StylePrimitive{
			Color:     stringPtr(charmtone.Cheeky.Hex()),
			Underline: boolPtr(true),
		},
		ImageText: ansi.StylePrimitive{
			Color:  stringPtr(charmtone.Squid.Hex()),
			Format: "Image: {{.text}} →",
		},
		Code: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix:          " ",
				Suffix:          " ",
				Color:           stringPtr(charmtone.Coral.Hex()),
				BackgroundColor: stringPtr(charmtone.Charcoal.Hex()),
			},
		},
		CodeBlock: ansi.StyleCodeBlock{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{
					Color: stringPtr(charmtone.Charcoal.Hex()),
				},
				Margin: uintPtr(defaultMargin),
			},
			Chroma: &ansi.Chroma{
				Text: ansi.StylePrimitive{
					Color: stringPtr(charmtone.Smoke.Hex()),
				},
				Error: ansi.StylePrimitive{
					Color:           stringPtr(charmtone.Butter.Hex()),
					BackgroundColor: stringPtr(charmtone.Sriracha.Hex()),
				},
				Comment: ansi.StylePrimitive{
					Color: stringPtr(charmtone.Oyster.Hex()),
				},
				CommentPreproc: ansi.StylePrimitive{
					Color: stringPtr(charmtone.Bengal.Hex()),
				},
				Keyword: ansi.StylePrimitive{
					Color: stringPtr(charmtone.Malibu.Hex()),
				},
				KeywordReserved: ansi.StylePrimitive{
					Color: stringPtr(charmtone.Pony.Hex()),
				},
				KeywordNamespace: ansi.StylePrimitive{
					Color: stringPtr(charmtone.Pony.Hex()),
				},
				KeywordType: ansi.StylePrimitive{
					Color: stringPtr(charmtone.Guppy.Hex()),
				},
				Operator: ansi.StylePrimitive{
					Color: stringPtr(charmtone.Salmon.Hex()),
				},
				Punctuation: ansi.StylePrimitive{
					Color: stringPtr(charmtone.Zest.Hex()),
				},
				Name: ansi.StylePrimitive{
					Color: stringPtr(charmtone.Smoke.Hex()),
				},
				NameBuiltin: ansi.StylePrimitive{
					Color: stringPtr(charmtone.Cheeky.Hex()),
				},
				NameTag: ansi.StylePrimitive{
					Color: stringPtr(charmtone.Mauve.Hex()),
				},
				NameAttribute: ansi.StylePrimitive{
					Color: stringPtr(charmtone.Hazy.Hex()),
				},
				NameClass: ansi.StylePrimitive{
					Color:     stringPtr(charmtone.Salt.Hex()),
					Underline: boolPtr(true),
					Bold:      boolPtr(true),
				},
				NameDecorator: ansi.StylePrimitive{
					Color: stringPtr(charmtone.Citron.Hex()),
				},
				NameFunction: ansi.StylePrimitive{
					Color: stringPtr(charmtone.Guac.Hex()),
				},
				LiteralNumber: ansi.StylePrimitive{
					Color: stringPtr(charmtone.Julep.Hex()),
				},
				LiteralString: ansi.StylePrimitive{
					Color: stringPtr(charmtone.Cumin.Hex()),
				},
				LiteralStringEscape: ansi.StylePrimitive{
					Color: stringPtr(charmtone.Bok.Hex()),
				},
				GenericDeleted: ansi.StylePrimitive{
					Color: stringPtr(charmtone.Coral.Hex()),
				},
				GenericEmph: ansi.StylePrimitive{
					Italic: boolPtr(true),
				},
				GenericInserted: ansi.StylePrimitive{
					Color: stringPtr(charmtone.Guac.Hex()),
				},
				GenericStrong: ansi.StylePrimitive{
					Bold: boolPtr(true),
				},
				GenericSubheading: ansi.StylePrimitive{
					Color: stringPtr(charmtone.Squid.Hex()),
				},
				Background: ansi.StylePrimitive{
					BackgroundColor: stringPtr(charmtone.Charcoal.Hex()),
				},
			},
		},
		Table: ansi.StyleTable{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{},
			},
		},
		DefinitionDescription: ansi.StylePrimitive{
			BlockPrefix: "\n ",
		},
	}

	// PlainMarkdown style - muted colors on subtle background for thinking content.
	plainBg := stringPtr(bgBaseLighter.Hex())
	plainFg := stringPtr(fgMuted.Hex())
	s.PlainMarkdown = ansi.StyleConfig{
		Document: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color:           plainFg,
				BackgroundColor: plainBg,
			},
		},
		BlockQuote: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color:           plainFg,
				BackgroundColor: plainBg,
			},
			Indent:      uintPtr(1),
			IndentToken: stringPtr("│ "),
		},
		List: ansi.StyleList{
			LevelIndent: defaultListIndent,
		},
		Heading: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				BlockSuffix:     "\n",
				Bold:            boolPtr(true),
				Color:           plainFg,
				BackgroundColor: plainBg,
			},
		},
		H1: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix:          " ",
				Suffix:          " ",
				Bold:            boolPtr(true),
				Color:           plainFg,
				BackgroundColor: plainBg,
			},
		},
		H2: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix:          "## ",
				Color:           plainFg,
				BackgroundColor: plainBg,
			},
		},
		H3: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix:          "### ",
				Color:           plainFg,
				BackgroundColor: plainBg,
			},
		},
		H4: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix:          "#### ",
				Color:           plainFg,
				BackgroundColor: plainBg,
			},
		},
		H5: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix:          "##### ",
				Color:           plainFg,
				BackgroundColor: plainBg,
			},
		},
		H6: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix:          "###### ",
				Color:           plainFg,
				BackgroundColor: plainBg,
			},
		},
		Strikethrough: ansi.StylePrimitive{
			CrossedOut:      boolPtr(true),
			Color:           plainFg,
			BackgroundColor: plainBg,
		},
		Emph: ansi.StylePrimitive{
			Italic:          boolPtr(true),
			Color:           plainFg,
			BackgroundColor: plainBg,
		},
		Strong: ansi.StylePrimitive{
			Bold:            boolPtr(true),
			Color:           plainFg,
			BackgroundColor: plainBg,
		},
		HorizontalRule: ansi.StylePrimitive{
			Format:          "\n--------\n",
			Color:           plainFg,
			BackgroundColor: plainBg,
		},
		Item: ansi.StylePrimitive{
			BlockPrefix:     "• ",
			Color:           plainFg,
			BackgroundColor: plainBg,
		},
		Enumeration: ansi.StylePrimitive{
			BlockPrefix:     ". ",
			Color:           plainFg,
			BackgroundColor: plainBg,
		},
		Task: ansi.StyleTask{
			StylePrimitive: ansi.StylePrimitive{
				Color:           plainFg,
				BackgroundColor: plainBg,
			},
			Ticked:   "[✓] ",
			Unticked: "[ ] ",
		},
		Link: ansi.StylePrimitive{
			Underline:       boolPtr(true),
			Color:           plainFg,
			BackgroundColor: plainBg,
		},
		LinkText: ansi.StylePrimitive{
			Bold:            boolPtr(true),
			Color:           plainFg,
			BackgroundColor: plainBg,
		},
		Image: ansi.StylePrimitive{
			Underline:       boolPtr(true),
			Color:           plainFg,
			BackgroundColor: plainBg,
		},
		ImageText: ansi.StylePrimitive{
			Format:          "Image: {{.text}} →",
			Color:           plainFg,
			BackgroundColor: plainBg,
		},
		Code: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix:          " ",
				Suffix:          " ",
				Color:           plainFg,
				BackgroundColor: plainBg,
			},
		},
		CodeBlock: ansi.StyleCodeBlock{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{
					Color:           plainFg,
					BackgroundColor: plainBg,
				},
				Margin: uintPtr(defaultMargin),
			},
		},
		Table: ansi.StyleTable{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{
					Color:           plainFg,
					BackgroundColor: plainBg,
				},
			},
		},
		DefinitionDescription: ansi.StylePrimitive{
			BlockPrefix:     "\n ",
			Color:           plainFg,
			BackgroundColor: plainBg,
		},
	}

	s.Help = help.Styles{
		ShortKey:       base.Foreground(fgMuted),
		ShortDesc:      base.Foreground(fgSubtle),
		ShortSeparator: base.Foreground(border),
		Ellipsis:       base.Foreground(border),
		FullKey:        base.Foreground(fgMuted),
		FullDesc:       base.Foreground(fgSubtle),
		FullSeparator:  base.Foreground(border),
	}

	s.Diff = diffview.Style{
		DividerLine: diffview.LineStyle{
			LineNumber: lipgloss.NewStyle().
				Foreground(fgHalfMuted).
				Background(bgBaseLighter),
			Code: lipgloss.NewStyle().
				Foreground(fgHalfMuted).
				Background(bgBaseLighter),
		},
		MissingLine: diffview.LineStyle{
			LineNumber: lipgloss.NewStyle().
				Background(bgBaseLighter),
			Code: lipgloss.NewStyle().
				Background(bgBaseLighter),
		},
		EqualLine: diffview.LineStyle{
			LineNumber: lipgloss.NewStyle().
				Foreground(fgMuted).
				Background(bgBase),
			Code: lipgloss.NewStyle().
				Foreground(fgMuted).
				Background(bgBase),
		},
		InsertLine: diffview.LineStyle{
			LineNumber: lipgloss.NewStyle().
				Foreground(lipgloss.Color("#629657")).
				Background(lipgloss.Color("#2b322a")),
			Symbol: lipgloss.NewStyle().
				Foreground(lipgloss.Color("#629657")).
				Background(lipgloss.Color("#323931")),
			Code: lipgloss.NewStyle().
				Background(lipgloss.Color("#323931")),
		},
		DeleteLine: diffview.LineStyle{
			LineNumber: lipgloss.NewStyle().
				Foreground(lipgloss.Color("#a45c59")).
				Background(lipgloss.Color("#312929")),
			Symbol: lipgloss.NewStyle().
				Foreground(lipgloss.Color("#a45c59")).
				Background(lipgloss.Color("#383030")),
			Code: lipgloss.NewStyle().
				Background(lipgloss.Color("#383030")),
		},
	}

	s.FilePicker = filepicker.Styles{
		DisabledCursor:   base.Foreground(fgMuted),
		Cursor:           base.Foreground(fgBase),
		Symlink:          base.Foreground(fgSubtle),
		Directory:        base.Foreground(primary),
		File:             base.Foreground(fgBase),
		DisabledFile:     base.Foreground(fgMuted),
		DisabledSelected: base.Background(bgOverlay).Foreground(fgMuted),
		Permission:       base.Foreground(fgMuted),
		Selected:         base.Background(primary).Foreground(fgBase),
		FileSize:         base.Foreground(fgMuted),
		EmptyDirectory:   base.Foreground(fgMuted).PaddingLeft(2).SetString("Empty directory"),
	}

	// borders
	s.FocusedMessageBorder = lipgloss.Border{Left: BorderThick}

	// text presets
	s.Base = lipgloss.NewStyle().Foreground(fgBase)
	s.Muted = lipgloss.NewStyle().Foreground(fgMuted)
	s.HalfMuted = lipgloss.NewStyle().Foreground(fgHalfMuted)
	s.Subtle = lipgloss.NewStyle().Foreground(fgSubtle)

	s.WindowTooSmall = s.Muted

	// tag presets
	s.TagBase = lipgloss.NewStyle().Padding(0, 1).Foreground(white)
	s.TagError = s.TagBase.Background(redDark)
	s.TagInfo = s.TagBase.Background(blueLight)

	// Compact header styles
	s.Header.Charm = base.Foreground(secondary)
	s.Header.Diagonals = base.Foreground(primary)
	s.Header.Percentage = s.Muted
	s.Header.Keystroke = s.Muted
	s.Header.KeystrokeTip = s.Subtle
	s.Header.WorkingDir = s.Muted
	s.Header.Separator = s.Subtle

	s.CompactDetails.Title = s.Base
	s.CompactDetails.View = s.Base.Padding(0, 1, 1, 1).Border(lipgloss.RoundedBorder()).BorderForeground(borderFocus)
	s.CompactDetails.Version = s.Muted

	// panels
	s.PanelMuted = s.Muted.Background(bgBaseLighter)
	s.PanelBase = lipgloss.NewStyle().Background(bgBase)

	// code line number
	s.LineNumber = lipgloss.NewStyle().Foreground(fgMuted).Background(bgBase).PaddingRight(1).PaddingLeft(1)

	// Tool calls
	s.ToolCallPending = lipgloss.NewStyle().Foreground(greenDark).SetString(ToolPending)
	s.ToolCallError = lipgloss.NewStyle().Foreground(redDark).SetString(ToolError)
	s.ToolCallSuccess = lipgloss.NewStyle().Foreground(green).SetString(ToolSuccess)
	// Cancelled uses muted tone but same glyph as pending
	s.ToolCallCancelled = s.Muted.SetString(ToolPending)
	s.EarlyStateMessage = s.Subtle.PaddingLeft(2)

	// Tool rendering styles
	s.Tool.IconPending = base.Foreground(greenDark).SetString(ToolPending)
	s.Tool.IconSuccess = base.Foreground(green).SetString(ToolSuccess)
	s.Tool.IconError = base.Foreground(redDark).SetString(ToolError)
	s.Tool.IconCancelled = s.Muted.SetString(ToolPending)

	s.Tool.NameNormal = base.Foreground(blue)
	s.Tool.NameNested = base.Foreground(fgHalfMuted)

	s.Tool.ParamMain = s.Subtle
	s.Tool.ParamKey = s.Subtle

	// Content rendering - prepared styles that accept width parameter
	s.Tool.ContentLine = s.Muted.Background(bgBaseLighter)
	s.Tool.ContentTruncation = s.Muted.Background(bgBaseLighter)
	s.Tool.ContentCodeLine = s.Base.Background(bgBase).PaddingLeft(2)
	s.Tool.ContentCodeTruncation = s.Muted.Background(bgBase).PaddingLeft(2)
	s.Tool.ContentCodeBg = bgBase
	s.Tool.Body = base.PaddingLeft(2)

	// Deprecated - kept for backward compatibility
	s.Tool.ContentBg = s.Muted.Background(bgBaseLighter)
	s.Tool.ContentText = s.Muted
	s.Tool.ContentLineNumber = base.Foreground(fgMuted).Background(bgBase).PaddingRight(1).PaddingLeft(1)

	s.Tool.StateWaiting = base.Foreground(fgSubtle)
	s.Tool.StateCancelled = base.Foreground(fgSubtle)

	s.Tool.ErrorTag = base.Padding(0, 1).Background(red).Foreground(white)
	s.Tool.ErrorMessage = base.Foreground(fgHalfMuted)

	// Diff and multi-edit styles
	s.Tool.DiffTruncation = s.Muted.Background(bgBaseLighter).PaddingLeft(2)
	s.Tool.NoteTag = base.Padding(0, 1).Background(info).Foreground(white)
	s.Tool.NoteMessage = base.Foreground(fgHalfMuted)

	// Job header styles
	s.Tool.JobIconPending = base.Foreground(greenDark)
	s.Tool.JobIconError = base.Foreground(redDark)
	s.Tool.JobIconSuccess = base.Foreground(green)
	s.Tool.JobToolName = base.Foreground(blue)
	s.Tool.JobAction = base.Foreground(blueDark)
	s.Tool.JobPID = s.Muted
	s.Tool.JobDescription = s.Subtle

	// Agent task styles
	s.Tool.AgentTaskTag = base.Bold(true).Padding(0, 1).MarginLeft(2).Background(blueLight).Foreground(white)
	s.Tool.AgentPrompt = s.Muted

	// Agentic fetch styles
	s.Tool.AgenticFetchPromptTag = base.Bold(true).Padding(0, 1).MarginLeft(2).Background(green).Foreground(border)

	// Todo styles
	s.Tool.TodoRatio = base.Foreground(blueDark)
	s.Tool.TodoCompletedIcon = base.Foreground(green)
	s.Tool.TodoInProgressIcon = base.Foreground(greenDark)
	s.Tool.TodoPendingIcon = base.Foreground(fgMuted)

	// MCP styles
	s.Tool.MCPName = base.Foreground(blue)
	s.Tool.MCPToolName = base.Foreground(blueDark)
	s.Tool.MCPArrow = base.Foreground(blue).SetString(ArrowRightIcon)

	// Buttons
	s.ButtonFocus = lipgloss.NewStyle().Foreground(white).Background(secondary)
	s.ButtonBlur = s.Base.Background(bgSubtle)

	// Borders
	s.BorderFocus = lipgloss.NewStyle().BorderForeground(borderFocus).Border(lipgloss.RoundedBorder()).Padding(1, 2)

	// Editor
	s.EditorPromptNormalFocused = lipgloss.NewStyle().Foreground(greenDark).SetString("::: ")
	s.EditorPromptNormalBlurred = s.EditorPromptNormalFocused.Foreground(fgMuted)
	s.EditorPromptYoloIconFocused = lipgloss.NewStyle().MarginRight(1).Foreground(charmtone.Oyster).Background(charmtone.Citron).Bold(true).SetString(" ! ")
	s.EditorPromptYoloIconBlurred = s.EditorPromptYoloIconFocused.Foreground(charmtone.Pepper).Background(charmtone.Squid)
	s.EditorPromptYoloDotsFocused = lipgloss.NewStyle().MarginRight(1).Foreground(charmtone.Zest).SetString(":::")
	s.EditorPromptYoloDotsBlurred = s.EditorPromptYoloDotsFocused.Foreground(charmtone.Squid)

	s.RadioOn = s.HalfMuted.SetString(RadioOn)
	s.RadioOff = s.HalfMuted.SetString(RadioOff)

	// Logo colors
	s.LogoFieldColor = primary
	s.LogoTitleColorA = secondary
	s.LogoTitleColorB = primary
	s.LogoCharmColor = secondary
	s.LogoVersionColor = primary

	// Section
	s.Section.Title = s.Subtle
	s.Section.Line = s.Base.Foreground(charmtone.Charcoal)

	// Initialize
	s.Initialize.Header = s.Base
	s.Initialize.Content = s.Muted
	s.Initialize.Accent = s.Base.Foreground(greenDark)

	// LSP and MCP status.
	s.ItemOfflineIcon = lipgloss.NewStyle().Foreground(charmtone.Squid).SetString("●")
	s.ItemBusyIcon = s.ItemOfflineIcon.Foreground(charmtone.Citron)
	s.ItemErrorIcon = s.ItemOfflineIcon.Foreground(charmtone.Coral)
	s.ItemOnlineIcon = s.ItemOfflineIcon.Foreground(charmtone.Guac)

	// LSP
	s.LSP.ErrorDiagnostic = s.Base.Foreground(redDark)
	s.LSP.WarningDiagnostic = s.Base.Foreground(warning)
	s.LSP.HintDiagnostic = s.Base.Foreground(fgHalfMuted)
	s.LSP.InfoDiagnostic = s.Base.Foreground(info)

	// Files
	s.Files.Path = s.Muted
	s.Files.Additions = s.Base.Foreground(greenDark)
	s.Files.Deletions = s.Base.Foreground(redDark)

	// Chat
	messageFocussedBorder := lipgloss.Border{
		Left: "▌",
	}

	s.Chat.Message.NoContent = lipgloss.NewStyle().Foreground(fgBase)
	s.Chat.Message.UserBlurred = s.Chat.Message.NoContent.PaddingLeft(1).BorderLeft(true).
		BorderForeground(primary).BorderStyle(normalBorder)
	s.Chat.Message.UserFocused = s.Chat.Message.NoContent.PaddingLeft(1).BorderLeft(true).
		BorderForeground(primary).BorderStyle(messageFocussedBorder)
	s.Chat.Message.AssistantBlurred = s.Chat.Message.NoContent.PaddingLeft(2)
	s.Chat.Message.AssistantFocused = s.Chat.Message.NoContent.PaddingLeft(1).BorderLeft(true).
		BorderForeground(greenDark).BorderStyle(messageFocussedBorder)
	s.Chat.Message.Thinking = lipgloss.NewStyle().MaxHeight(10)
	s.Chat.Message.ErrorTag = lipgloss.NewStyle().Padding(0, 1).
		Background(red).Foreground(white)
	s.Chat.Message.ErrorTitle = lipgloss.NewStyle().Foreground(fgHalfMuted)
	s.Chat.Message.ErrorDetails = lipgloss.NewStyle().Foreground(fgSubtle)

	// Message item styles
	s.Chat.Message.ToolCallFocused = s.Muted.PaddingLeft(1).
		BorderStyle(messageFocussedBorder).
		BorderLeft(true).
		BorderForeground(greenDark)
	s.Chat.Message.ToolCallBlurred = s.Muted.PaddingLeft(2)
	// No padding or border for compact tool calls within messages
	s.Chat.Message.ToolCallCompact = s.Muted
	s.Chat.Message.SectionHeader = s.Base.PaddingLeft(2)
	s.Chat.Message.AssistantInfoIcon = s.Subtle
	s.Chat.Message.AssistantInfoModel = s.Muted
	s.Chat.Message.AssistantInfoProvider = s.Subtle
	s.Chat.Message.AssistantInfoDuration = s.Subtle

	// Thinking section styles
	s.Chat.Message.ThinkingBox = s.Subtle.Background(bgBaseLighter)
	s.Chat.Message.ThinkingTruncationHint = s.Muted
	s.Chat.Message.ThinkingFooterTitle = s.Muted
	s.Chat.Message.ThinkingFooterDuration = s.Subtle

	// Text selection.
	s.TextSelection = lipgloss.NewStyle().Foreground(charmtone.Salt).Background(charmtone.Charple)

	// Dialog styles
	s.Dialog.Title = base.Padding(0, 1).Foreground(primary)
	s.Dialog.TitleText = base.Foreground(primary)
	s.Dialog.TitleError = base.Foreground(red)
	s.Dialog.TitleAccent = base.Foreground(green).Bold(true)
	s.Dialog.View = base.Border(lipgloss.RoundedBorder()).BorderForeground(borderFocus)
	s.Dialog.PrimaryText = base.Padding(0, 1).Foreground(primary)
	s.Dialog.SecondaryText = base.Padding(0, 1).Foreground(fgSubtle)
	s.Dialog.HelpView = base.Padding(0, 1).AlignHorizontal(lipgloss.Left)
	s.Dialog.Help.ShortKey = base.Foreground(fgMuted)
	s.Dialog.Help.ShortDesc = base.Foreground(fgSubtle)
	s.Dialog.Help.ShortSeparator = base.Foreground(border)
	s.Dialog.Help.Ellipsis = base.Foreground(border)
	s.Dialog.Help.FullKey = base.Foreground(fgMuted)
	s.Dialog.Help.FullDesc = base.Foreground(fgSubtle)
	s.Dialog.Help.FullSeparator = base.Foreground(border)
	s.Dialog.NormalItem = base.Padding(0, 1).Foreground(fgBase)
	s.Dialog.SelectedItem = base.Padding(0, 1).Background(primary).Foreground(fgBase)
	s.Dialog.InputPrompt = base.Margin(1, 1)

	s.Dialog.List = base.Margin(0, 0, 1, 0)
	s.Dialog.ContentPanel = base.Background(bgSubtle).Foreground(fgBase).Padding(1, 2)
	s.Dialog.Spinner = base.Foreground(secondary)
	s.Dialog.ScrollbarThumb = base.Foreground(secondary)
	s.Dialog.ScrollbarTrack = base.Foreground(border)

	s.Dialog.ImagePreview = lipgloss.NewStyle().Padding(0, 1).Foreground(fgSubtle)

	s.Dialog.Arguments.Content = base.Padding(1)
	s.Dialog.Arguments.Description = base.MarginBottom(1).MaxHeight(3)
	s.Dialog.Arguments.InputLabelBlurred = base.Foreground(fgMuted)
	s.Dialog.Arguments.InputLabelFocused = base.Bold(true)
	s.Dialog.Arguments.InputRequiredMarkBlurred = base.Foreground(fgMuted).SetString("*")
	s.Dialog.Arguments.InputRequiredMarkFocused = base.Foreground(primary).Bold(true).SetString("*")

	s.Dialog.Sessions.DeletingTitle = s.Dialog.Title.Foreground(red)
	s.Dialog.Sessions.DeletingView = s.Dialog.View.BorderForeground(red)
	s.Dialog.Sessions.DeletingMessage = s.Base.Padding(1)
	s.Dialog.Sessions.DeletingTitleGradientFromColor = red
	s.Dialog.Sessions.DeletingTitleGradientToColor = s.Primary
	s.Dialog.Sessions.DeletingItemBlurred = s.Dialog.NormalItem.Foreground(fgSubtle)
	s.Dialog.Sessions.DeletingItemFocused = s.Dialog.SelectedItem.Background(red).Foreground(charmtone.Butter)

	s.Dialog.Sessions.RenamingingTitle = s.Dialog.Title.Foreground(charmtone.Zest)
	s.Dialog.Sessions.RenamingView = s.Dialog.View.BorderForeground(charmtone.Zest)
	s.Dialog.Sessions.RenamingingMessage = s.Base.Padding(1)
	s.Dialog.Sessions.RenamingTitleGradientFromColor = charmtone.Zest
	s.Dialog.Sessions.RenamingTitleGradientToColor = charmtone.Bok
	s.Dialog.Sessions.RenamingItemBlurred = s.Dialog.NormalItem.Foreground(fgSubtle)
	s.Dialog.Sessions.RenamingingItemFocused = s.Dialog.SelectedItem.UnsetBackground().UnsetForeground()
	s.Dialog.Sessions.RenamingPlaceholder = base.Foreground(charmtone.Squid)

	s.Status.Help = lipgloss.NewStyle().Padding(0, 1)
	s.Status.SuccessIndicator = base.Foreground(bgSubtle).Background(green).Padding(0, 1).Bold(true).SetString("OKAY!")
	s.Status.InfoIndicator = s.Status.SuccessIndicator
	s.Status.UpdateIndicator = s.Status.SuccessIndicator.SetString("HEY!")
	s.Status.WarnIndicator = s.Status.SuccessIndicator.Foreground(bgOverlay).Background(yellow).SetString("WARNING")
	s.Status.ErrorIndicator = s.Status.SuccessIndicator.Foreground(bgBase).Background(red).SetString("ERROR")
	s.Status.SuccessMessage = base.Foreground(bgSubtle).Background(greenDark).Padding(0, 1)
	s.Status.InfoMessage = s.Status.SuccessMessage
	s.Status.UpdateMessage = s.Status.SuccessMessage
	s.Status.WarnMessage = s.Status.SuccessMessage.Foreground(bgOverlay).Background(warning)
	s.Status.ErrorMessage = s.Status.SuccessMessage.Foreground(white).Background(redDark)

	// Completions styles
	s.Completions.Normal = base.Background(bgSubtle).Foreground(fgBase)
	s.Completions.Focused = base.Background(primary).Foreground(white)
	s.Completions.Match = base.Underline(true)

	// Attachments styles
	attachmentIconStyle := base.Foreground(bgSubtle).Background(green).Padding(0, 1)
	s.Attachments.Image = attachmentIconStyle.SetString(ImageIcon)
	s.Attachments.Text = attachmentIconStyle.SetString(TextIcon)
	s.Attachments.Normal = base.Padding(0, 1).MarginRight(1).Background(fgMuted).Foreground(fgBase)
	s.Attachments.Deleting = base.Padding(0, 1).Bold(true).Background(red).Foreground(fgBase)

	// Pills styles
	s.Pills.Base = base.Padding(0, 1)
	s.Pills.Focused = base.Padding(0, 1).BorderStyle(lipgloss.RoundedBorder()).BorderForeground(bgOverlay)
	s.Pills.Blurred = base.Padding(0, 1).BorderStyle(lipgloss.HiddenBorder())
	s.Pills.QueueItemPrefix = s.Muted.SetString("  •")
	s.Pills.HelpKey = s.Muted
	s.Pills.HelpText = s.Subtle
	s.Pills.Area = base
	s.Pills.TodoSpinner = base.Foreground(greenDark)

	return s
}

// Helper functions for style pointers
func boolPtr(b bool) *bool       { return &b }
func stringPtr(s string) *string { return &s }
func uintPtr(u uint) *uint       { return &u }
func chromaStyle(style ansi.StylePrimitive) string {
	var s strings.Builder

	if style.Color != nil {
		s.WriteString(*style.Color)
	}
	if style.BackgroundColor != nil {
		if s.Len() > 0 {
			s.WriteString(" ")
		}
		s.WriteString("bg:")
		s.WriteString(*style.BackgroundColor)
	}
	if style.Italic != nil && *style.Italic {
		if s.Len() > 0 {
			s.WriteString(" ")
		}
		s.WriteString("italic")
	}
	if style.Bold != nil && *style.Bold {
		if s.Len() > 0 {
			s.WriteString(" ")
		}
		s.WriteString("bold")
	}
	if style.Underline != nil && *style.Underline {
		if s.Len() > 0 {
			s.WriteString(" ")
		}
		s.WriteString("underline")
	}

	return s.String()
}
