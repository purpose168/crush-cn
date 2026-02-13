package cmd

import (
	"bytes"
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/browser"
	"github.com/purpose168/crush-cn/internal/config"
	"github.com/purpose168/crush-cn/internal/db"
	"github.com/spf13/cobra"
)

//go:embed stats/index.html
var statsTemplate string

//go:embed stats/index.css
var statsCSS string

//go:embed stats/index.js
var statsJS string

//go:embed stats/header.svg
var headerSVG string

//go:embed stats/heartbit.svg
var heartbitSVG string

//go:embed stats/footer.svg
var footerSVG string

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "显示使用统计信息",
	Long:  "生成并显示使用统计信息，包括令牌使用情况、成本和活动模式",
	RunE:  runStats,
}

// 星期几的名称，用于星期统计。
var dayNames = []string{"星期日", "星期一", "星期二", "星期三", "星期四", "星期五", "星期六"}

// Stats 包含所有统计数据。
type Stats struct {
	GeneratedAt       time.Time          `json:"generated_at"`
	Total             TotalStats         `json:"total"`
	UsageByDay        []DailyUsage       `json:"usage_by_day"`
	UsageByModel      []ModelUsage       `json:"usage_by_model"`
	UsageByHour       []HourlyUsage      `json:"usage_by_hour"`
	UsageByDayOfWeek  []DayOfWeekUsage   `json:"usage_by_day_of_week"`
	RecentActivity    []DailyActivity    `json:"recent_activity"`
	AvgResponseTimeMs float64            `json:"avg_response_time_ms"`
	ToolUsage         []ToolUsage        `json:"tool_usage"`
	HourDayHeatmap    []HourDayHeatmapPt `json:"hour_day_heatmap"`
}

// TotalStats 包含总统计信息。
type TotalStats struct {
	TotalSessions         int64   `json:"total_sessions"`
	TotalPromptTokens     int64   `json:"total_prompt_tokens"`
	TotalCompletionTokens int64   `json:"total_completion_tokens"`
	TotalTokens           int64   `json:"total_tokens"`
	TotalCost             float64 `json:"total_cost"`
	TotalMessages         int64   `json:"total_messages"`
	AvgTokensPerSession   float64 `json:"avg_tokens_per_session"`
	AvgMessagesPerSession float64 `json:"avg_messages_per_session"`
}

// DailyUsage 包含每日使用统计信息。
type DailyUsage struct {
	Day              string  `json:"day"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	TotalTokens      int64   `json:"total_tokens"`
	Cost             float64 `json:"cost"`
	SessionCount     int64   `json:"session_count"`
}

// ModelUsage 包含按模型统计的使用信息。
type ModelUsage struct {
	Model        string `json:"model"`
	Provider     string `json:"provider"`
	MessageCount int64  `json:"message_count"`
}

// HourlyUsage 包含按小时统计的使用信息。
type HourlyUsage struct {
	Hour         int   `json:"hour"`
	SessionCount int64 `json:"session_count"`
}

// DayOfWeekUsage 包含按星期几统计的使用信息。
type DayOfWeekUsage struct {
	DayOfWeek        int    `json:"day_of_week"`
	DayName          string `json:"day_name"`
	SessionCount     int64  `json:"session_count"`
	PromptTokens     int64  `json:"prompt_tokens"`
	CompletionTokens int64  `json:"completion_tokens"`
}

// DailyActivity 包含每日活动统计信息。
type DailyActivity struct {
	Day          string  `json:"day"`
	SessionCount int64   `json:"session_count"`
	TotalTokens  int64   `json:"total_tokens"`
	Cost         float64 `json:"cost"`
}

// ToolUsage 包含工具使用统计信息。
type ToolUsage struct {
	ToolName  string `json:"tool_name"`
	CallCount int64  `json:"call_count"`
}

// HourDayHeatmapPt 包含小时/天热力图数据点。
type HourDayHeatmapPt struct {
	DayOfWeek    int   `json:"day_of_week"`
	Hour         int   `json:"hour"`
	SessionCount int64 `json:"session_count"`
}

// runStats 执行 stats 命令，生成并显示统计信息。
func runStats(cmd *cobra.Command, _ []string) error {
	dataDir, _ := cmd.Flags().GetString("data-dir")
	ctx := cmd.Context()

	if dataDir == "" {
		cfg, err := config.Init("", "", false)
		if err != nil {
			return fmt.Errorf("初始化配置失败: %w", err)
		}
		dataDir = cfg.Options.DataDirectory
	}

	conn, err := db.Connect(ctx, dataDir)
	if err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}
	defer conn.Close()

	stats, err := gatherStats(ctx, conn)
	if err != nil {
		return fmt.Errorf("收集统计信息失败: %w", err)
	}

	if stats.Total.TotalSessions == 0 {
		return fmt.Errorf("无可用数据: 数据库中未找到会话")
	}

	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("获取当前用户失败: %w", err)
	}
	username := currentUser.Username
	project, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取当前目录失败: %w", err)
	}
	project = strings.Replace(project, currentUser.HomeDir, "~", 1)

	htmlPath := filepath.Join(dataDir, "stats/index.html")
	if err := generateHTML(stats, project, username, htmlPath); err != nil {
		return fmt.Errorf("生成 HTML 失败: %w", err)
	}

	fmt.Printf("统计信息已生成: %s\n", htmlPath)

	if err := browser.OpenFile(htmlPath); err != nil {
		fmt.Printf("无法打开浏览器: %v\n", err)
		fmt.Println("请手动打开文件。")
	}

	return nil
}

// gatherStats 从数据库收集所有统计信息。
func gatherStats(ctx context.Context, conn *sql.DB) (*Stats, error) {
	queries := db.New(conn)

	stats := &Stats{
		GeneratedAt: time.Now(),
	}

	// 总统计信息。
	total, err := queries.GetTotalStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取总统计信息: %w", err)
	}
	stats.Total = TotalStats{
		TotalSessions:         total.TotalSessions,
		TotalPromptTokens:     toInt64(total.TotalPromptTokens),
		TotalCompletionTokens: toInt64(total.TotalCompletionTokens),
		TotalTokens:           toInt64(total.TotalPromptTokens) + toInt64(total.TotalCompletionTokens),
		TotalCost:             toFloat64(total.TotalCost),
		TotalMessages:         toInt64(total.TotalMessages),
		AvgTokensPerSession:   toFloat64(total.AvgTokensPerSession),
		AvgMessagesPerSession: toFloat64(total.AvgMessagesPerSession),
	}

	// 按天使用情况。
	dailyUsage, err := queries.GetUsageByDay(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取按天使用情况: %w", err)
	}
	for _, d := range dailyUsage {
		prompt := nullFloat64ToInt64(d.PromptTokens)
		completion := nullFloat64ToInt64(d.CompletionTokens)
		stats.UsageByDay = append(stats.UsageByDay, DailyUsage{
			Day:              fmt.Sprintf("%v", d.Day),
			PromptTokens:     prompt,
			CompletionTokens: completion,
			TotalTokens:      prompt + completion,
			Cost:             d.Cost.Float64,
			SessionCount:     d.SessionCount,
		})
	}

	// 按模型使用情况。
	modelUsage, err := queries.GetUsageByModel(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取按模型使用情况: %w", err)
	}
	for _, m := range modelUsage {
		stats.UsageByModel = append(stats.UsageByModel, ModelUsage{
			Model:        m.Model,
			Provider:     m.Provider,
			MessageCount: m.MessageCount,
		})
	}

	// 按小时使用情况。
	hourlyUsage, err := queries.GetUsageByHour(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取按小时使用情况: %w", err)
	}
	for _, h := range hourlyUsage {
		stats.UsageByHour = append(stats.UsageByHour, HourlyUsage{
			Hour:         int(h.Hour),
			SessionCount: h.SessionCount,
		})
	}

	// 按星期几使用情况。
	dowUsage, err := queries.GetUsageByDayOfWeek(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取按星期几使用情况: %w", err)
	}
	for _, d := range dowUsage {
		stats.UsageByDayOfWeek = append(stats.UsageByDayOfWeek, DayOfWeekUsage{
			DayOfWeek:        int(d.DayOfWeek),
			DayName:          dayNames[int(d.DayOfWeek)],
			SessionCount:     d.SessionCount,
			PromptTokens:     nullFloat64ToInt64(d.PromptTokens),
			CompletionTokens: nullFloat64ToInt64(d.CompletionTokens),
		})
	}

	// 最近活动（最近30天）。
	recent, err := queries.GetRecentActivity(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取最近活动: %w", err)
	}
	for _, r := range recent {
		stats.RecentActivity = append(stats.RecentActivity, DailyActivity{
			Day:          fmt.Sprintf("%v", r.Day),
			SessionCount: r.SessionCount,
			TotalTokens:  nullFloat64ToInt64(r.TotalTokens),
			Cost:         r.Cost.Float64,
		})
	}

	// 平均响应时间。
	avgResp, err := queries.GetAverageResponseTime(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取平均响应时间: %w", err)
	}
	stats.AvgResponseTimeMs = toFloat64(avgResp) * 1000

	// 工具使用情况。
	toolUsage, err := queries.GetToolUsage(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取工具使用情况: %w", err)
	}
	for _, t := range toolUsage {
		if name, ok := t.ToolName.(string); ok && name != "" {
			stats.ToolUsage = append(stats.ToolUsage, ToolUsage{
				ToolName:  name,
				CallCount: t.CallCount,
			})
		}
	}

	// 小时/天热力图。
	heatmap, err := queries.GetHourDayHeatmap(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取小时/天热力图: %w", err)
	}
	for _, h := range heatmap {
		stats.HourDayHeatmap = append(stats.HourDayHeatmap, HourDayHeatmapPt{
			DayOfWeek:    int(h.DayOfWeek),
			Hour:         int(h.Hour),
			SessionCount: h.SessionCount,
		})
	}

	return stats, nil
}

// toInt64 将任意类型转换为 int64。
func toInt64(v any) int64 {
	switch val := v.(type) {
	case int64:
		return val
	case float64:
		return int64(val)
	case int:
		return int64(val)
	default:
		return 0
	}
}

// toFloat64 将任意类型转换为 float64。
func toFloat64(v any) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int64:
		return float64(val)
	case int:
		return float64(val)
	default:
		return 0
	}
}

// nullFloat64ToInt64 将 sql.NullFloat64 转换为 int64。
func nullFloat64ToInt64(n sql.NullFloat64) int64 {
	if n.Valid {
		return int64(n.Float64)
	}
	return 0
}

// generateHTML 生成包含统计信息的 HTML 文件。
func generateHTML(stats *Stats, projName, username, path string) error {
	statsJSON, err := json.Marshal(stats)
	if err != nil {
		return err
	}

	tmpl, err := template.New("stats").Parse(statsTemplate)
	if err != nil {
		return fmt.Errorf("解析模板: %w", err)
	}

	data := struct {
		StatsJSON   template.JS
		CSS         template.CSS
		JS          template.JS
		Header      template.HTML
		Heartbit    template.HTML
		Footer      template.HTML
		GeneratedAt string
		ProjectName string
		Username    string
	}{
		StatsJSON:   template.JS(statsJSON),
		CSS:         template.CSS(statsCSS),
		JS:          template.JS(statsJS),
		Header:      template.HTML(headerSVG),
		Heartbit:    template.HTML(heartbitSVG),
		Footer:      template.HTML(footerSVG),
		GeneratedAt: stats.GeneratedAt.Format("2006-01-02"),
		ProjectName: projName,
		Username:    username,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("执行模板: %w", err)
	}

	// 确保父目录存在。
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("创建目录: %w", err)
	}

	return os.WriteFile(path, buf.Bytes(), 0o644)
}
