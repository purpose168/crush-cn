package model

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/purpose168/crush-cn/internal/diff"
	"github.com/purpose168/crush-cn/internal/fsext"
	"github.com/purpose168/crush-cn/internal/history"
	"github.com/purpose168/crush-cn/internal/session"
	"github.com/purpose168/crush-cn/internal/ui/common"
	"github.com/purpose168/crush-cn/internal/ui/styles"
	"github.com/purpose168/crush-cn/internal/ui/util"
)

// loadSessionMsg 是表示会话及其文件已加载的消息。
type loadSessionMsg struct {
	session   *session.Session
	files     []SessionFile
	readFiles []string
}

// lspFilePaths 从已修改和已读取的文件中返回去重后的文件路径，用于启动LSP服务器。
func (msg loadSessionMsg) lspFilePaths() []string {
	seen := make(map[string]struct{}, len(msg.files)+len(msg.readFiles))
	paths := make([]string, 0, len(msg.files)+len(msg.readFiles))
	for _, f := range msg.files {
		p := f.LatestVersion.Path
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		paths = append(paths, p)
	}
	for _, p := range msg.readFiles {
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		paths = append(paths, p)
	}
	return paths
}

// SessionFile 跟踪会话中文件的第一个和最新版本，以及总增删行数。
type SessionFile struct {
	FirstVersion  history.File
	LatestVersion history.File
	Additions     int
	Deletions     int
}

// loadSession 加载会话及其关联文件，并计算会话中每个文件的差异统计（增删行数）。
// 它返回一个 tea.Cmd，执行时会获取会话数据并返回包含已处理会话文件的 sessionFilesLoadedMsg。
func (m *UI) loadSession(sessionID string) tea.Cmd {
	return func() tea.Msg {
		session, err := m.com.App.Sessions.Get(context.Background(), sessionID)
		if err != nil {
			return util.ReportError(err)
		}

		sessionFiles, err := m.loadSessionFiles(sessionID)
		if err != nil {
			return util.ReportError(err)
		}

		readFiles, err := m.com.App.FileTracker.ListReadFiles(context.Background(), sessionID)
		if err != nil {
			slog.Error("加载会话的已读取文件失败", "error", err)
		}

		return loadSessionMsg{
			session:   &session,
			files:     sessionFiles,
			readFiles: readFiles,
		}
	}
}

func (m *UI) loadSessionFiles(sessionID string) ([]SessionFile, error) {
	files, err := m.com.App.History.ListBySession(context.Background(), sessionID)
	if err != nil {
		return nil, err
	}

	filesByPath := make(map[string][]history.File)
	for _, f := range files {
		filesByPath[f.Path] = append(filesByPath[f.Path], f)
	}
	sessionFiles := make([]SessionFile, 0, len(filesByPath))
	for _, versions := range filesByPath {
		if len(versions) == 0 {
			continue
		}

		first := versions[0]
		last := versions[0]
		for _, v := range versions {
			if v.Version < first.Version {
				first = v
			}
			if v.Version > last.Version {
				last = v
			}
		}

		_, additions, deletions := diff.GenerateDiff(first.Content, last.Content, first.Path)

		sessionFiles = append(sessionFiles, SessionFile{
			FirstVersion:  first,
			LatestVersion: last,
			Additions:     additions,
			Deletions:     deletions,
		})
	}

	slices.SortFunc(sessionFiles, func(a, b SessionFile) int {
		if a.LatestVersion.UpdatedAt > b.LatestVersion.UpdatedAt {
			return -1
		}
		if a.LatestVersion.UpdatedAt < b.LatestVersion.UpdatedAt {
			return 1
		}
		return 0
	})
	return sessionFiles, nil
}

// handleFileEvent 处理文件更改事件，使用新文件信息更新会话文件列表。
func (m *UI) handleFileEvent(file history.File) tea.Cmd {
	if m.session == nil || file.SessionID != m.session.ID {
		return nil
	}

	return func() tea.Msg {
		sessionFiles, err := m.loadSessionFiles(m.session.ID)
		// 无法加载会话文件
		if err != nil {
			return util.NewErrorMsg(err)
		}

		return sessionFilesUpdatesMsg{
			sessionFiles: sessionFiles,
		}
	}
}

// filesInfo 为侧边栏渲染已修改文件部分，显示文件及其增删计数。
func (m *UI) filesInfo(cwd string, width, maxItems int, isSection bool) string {
	t := m.com.Styles

	title := t.Subtle.Render("已修改文件")
	if isSection {
		title = common.Section(t, "已修改文件", width)
	}
	list := t.Subtle.Render("无")
	var filesWithChanges []SessionFile
	for _, f := range m.sessionFiles {
		if f.Additions == 0 && f.Deletions == 0 {
			continue
		}
		filesWithChanges = append(filesWithChanges, f)
	}
	if len(filesWithChanges) > 0 {
		list = fileList(t, cwd, filesWithChanges, width, maxItems)
	}

	return lipgloss.NewStyle().Width(width).Render(fmt.Sprintf("%s\n\n%s", title, list))
}

// fileList 渲染带有差异统计的文件列表，截断至maxItems并在需要时显示"...以及其余N项"消息。
func fileList(t *styles.Styles, cwd string, filesWithChanges []SessionFile, width, maxItems int) string {
	if maxItems <= 0 {
		return ""
	}
	var renderedFiles []string
	filesShown := 0

	for _, f := range filesWithChanges {
		// 跳过没有更改的文件
		if filesShown >= maxItems {
			break
		}

		// 构建带有颜色的状态字符串
		var statusParts []string
		if f.Additions > 0 {
			statusParts = append(statusParts, t.Files.Additions.Render(fmt.Sprintf("+%d", f.Additions)))
		}
		if f.Deletions > 0 {
			statusParts = append(statusParts, t.Files.Deletions.Render(fmt.Sprintf("-%d", f.Deletions)))
		}
		extraContent := strings.Join(statusParts, " ")

		// 格式化文件路径
		filePath := f.FirstVersion.Path
		if rel, err := filepath.Rel(cwd, filePath); err == nil {
			filePath = rel
		}
		filePath = fsext.DirTrim(filePath, 2)
		filePath = ansi.Truncate(filePath, width-(lipgloss.Width(extraContent)-2), "…")

		line := t.Files.Path.Render(filePath)
		if extraContent != "" {
			line = fmt.Sprintf("%s %s", line, extraContent)
		}

		renderedFiles = append(renderedFiles, line)
		filesShown++
	}

	if len(filesWithChanges) > maxItems {
		remaining := len(filesWithChanges) - maxItems
		renderedFiles = append(renderedFiles, t.Subtle.Render(fmt.Sprintf("以及其余 %d 项", remaining)))
	}

	return lipgloss.JoinVertical(lipgloss.Left, renderedFiles...)
}

// startLSPs 为给定的文件路径启动LSP服务器。
func (m *UI) startLSPs(paths []string) tea.Cmd {
	if len(paths) == 0 {
		return nil
	}

	return func() tea.Msg {
		ctx := context.Background()
		for _, path := range paths {
			m.com.App.LSPManager.Start(ctx, path)
		}
		return nil
	}
}
