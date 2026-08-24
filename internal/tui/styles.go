package tui

import "github.com/charmbracelet/lipgloss"

// 主题配色（采用 AdaptiveColor 完美自适应深色/浅色终端）
var (
	PrimaryColor    = lipgloss.AdaptiveColor{Light: "#4F46E5", Dark: "#818CF8"} // 紫
	SecondaryColor  = lipgloss.AdaptiveColor{Light: "#0284C7", Dark: "#38BDF8"} // 蓝
	SuccessColor    = lipgloss.AdaptiveColor{Light: "#059669", Dark: "#34D399"} // 绿
	WarningColor    = lipgloss.AdaptiveColor{Light: "#D97706", Dark: "#FBBF24"} // 橙
	DangerColor     = lipgloss.AdaptiveColor{Light: "#DC2626", Dark: "#F87171"} // 红
	ErrorColor      = DangerColor
	TextColor       = lipgloss.AdaptiveColor{Light: "#111827", Dark: "#F9FAFB"} // 正文文字（浅底深字，深底白字）
	MutedTextColor  = lipgloss.AdaptiveColor{Light: "#4B5563", Dark: "#9CA3AF"} // 次级说明文字
	ActiveBgColor   = lipgloss.AdaptiveColor{Light: "#312E81", Dark: "#3730A3"} // 选中高亮背景
	CardBorderColor = lipgloss.AdaptiveColor{Light: "#6366F1", Dark: "#6366F1"} // 边框
	HighlightColor  = lipgloss.AdaptiveColor{Light: "#0284C7", Dark: "#38BDF8"} // 强调高亮色
)

// 样式定义
var (
	// 顶部大标题
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#4F46E5")).
			Padding(0, 1).
			MarginBottom(1)

	SubTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(PrimaryColor)

	// 卡片与面板样式
	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(CardBorderColor).
			Padding(0, 1)

	LogBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(CardBorderColor).
			Padding(0, 1)

	// 状态栏
	StatusLabel = lipgloss.NewStyle().
			Bold(true).
			Foreground(PrimaryColor)

	StatusPath = lipgloss.NewStyle().
			Bold(true).
			Foreground(TextColor)

	// 菜单列表项
	ItemNormalTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(TextColor)

	ItemNormalDesc = lipgloss.NewStyle().
			Foreground(MutedTextColor)

	// 徽章与标签
	BadgeSuccess = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#059669")).
			Padding(0, 1).
			Bold(true)

	BadgeWarning = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#D97706")).
			Padding(0, 1).
			Bold(true)

	BadgeDanger = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#DC2626")).
			Padding(0, 1).
			Bold(true)

	BadgeInfo = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#0284C7")).
			Padding(0, 1).
			Bold(true)

	BadgeParam = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FCD34D")).
			Background(lipgloss.Color("#1E293B")).
			Padding(0, 1).
			Bold(true)

	// 底部帮助快捷键栏
	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(PrimaryColor).
			Bold(true)

	HelpDescStyle = lipgloss.NewStyle().
			Foreground(TextColor)

	HelpBar = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(lipgloss.AdaptiveColor{Light: "#9CA3AF", Dark: "#4B5563"}).
		PaddingTop(0)
)
