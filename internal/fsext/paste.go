package fsext

import (
	"os"
	"strings"
)

func ParsePastedFiles(s string) []string {
	s = strings.TrimSpace(s)

	// 注意：Rio 在 Windows 上出于某种原因会添加 NULL 字符。
	s = strings.ReplaceAll(s, "\x00", "")

	switch {
	case attemptStat(s):
		return strings.Split(s, "\n")
	case os.Getenv("WT_SESSION") != "":
		return windowsTerminalParsePastedFiles(s)
	default:
		return unixParsePastedFiles(s)
	}
}

func attemptStat(s string) bool {
	for path := range strings.SplitSeq(s, "\n") {
		if info, err := os.Stat(path); err != nil || info.IsDir() {
			return false
		}
	}
	return true
}

func windowsTerminalParsePastedFiles(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}

	var (
		paths    []string
		current  strings.Builder
		inQuotes = false
	)
	for i := range len(s) {
		ch := s[i]

		switch {
		case ch == '"':
			if inQuotes {
				// 引号部分结束
				if current.Len() > 0 {
					paths = append(paths, current.String())
					current.Reset()
				}
				inQuotes = false
			} else {
				// 引号部分开始
				inQuotes = true
			}
		case inQuotes:
			current.WriteByte(ch)
		case ch != ' ':
			// 引号外的文本不被允许
			return nil
		}
	}

	// 如果引号正确关闭，添加任何剩余内容
	if current.Len() > 0 && !inQuotes {
		paths = append(paths, current.String())
	}

	// 如果引号未关闭，返回空（格式错误的输入）
	if inQuotes {
		return nil
	}

	return paths
}

func unixParsePastedFiles(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}

	var (
		paths   []string
		current strings.Builder
		escaped = false
	)
	for i := range len(s) {
		ch := s[i]

		switch {
		case escaped:
			// 反斜杠之后，按原样添加字符（包括空格）
			current.WriteByte(ch)
			escaped = false
		case ch == '\\':
			// 检查此反斜杠是否在字符串末尾
			if i == len(s)-1 {
				// 尾部反斜杠，按字面值处理
				current.WriteByte(ch)
			} else {
				// 转义序列开始
				escaped = true
			}
		case ch == ' ':
			// 空格分隔路径（除非被转义）
			if current.Len() > 0 {
				paths = append(paths, current.String())
				current.Reset()
			}
		default:
			current.WriteByte(ch)
		}
	}

	// 如果存在尾部反斜杠则处理
	if escaped {
		current.WriteByte('\\')
	}

	// 添加最后一个路径（如果有）
	if current.Len() > 0 {
		paths = append(paths, current.String())
	}

	return paths
}
