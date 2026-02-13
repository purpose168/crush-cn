package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"time"

	"charm.land/log/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/term"
	"github.com/nxadm/tail"
	"github.com/purpose168/crush-cn/internal/config"
	"github.com/spf13/cobra"
)

const defaultTailLines = 1000

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "查看 crush 日志",
	Long:  `查看 Crush 生成的日志。此命令允许您查看日志输出，用于调试和监控。`,

	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := cmd.Flags().GetString("cwd")
		if err != nil {
			return fmt.Errorf("获取当前工作目录失败: %v", err)
		}

		dataDir, err := cmd.Flags().GetString("data-dir")
		if err != nil {
			return fmt.Errorf("获取数据目录失败: %v", err)
		}

		follow, err := cmd.Flags().GetBool("follow")
		if err != nil {
			return fmt.Errorf("获取 follow 标志失败: %v", err)
		}

		tailLines, err := cmd.Flags().GetInt("tail")
		if err != nil {
			return fmt.Errorf("获取 tail 标志失败: %v", err)
		}

		log.SetLevel(log.DebugLevel)
		log.SetOutput(os.Stdout)
		if !term.IsTerminal(os.Stdout.Fd()) {
			log.SetColorProfile(colorprofile.NoTTY)
		}

		cfg, err := config.Load(cwd, dataDir, false)
		if err != nil {
			return fmt.Errorf("加载配置失败: %v", err)
		}
		logsFile := filepath.Join(cfg.Options.DataDirectory, "logs", "crush.log")
		_, err = os.Stat(logsFile)
		if os.IsNotExist(err) {
			log.Warn("看起来您不在 crush 项目中。未找到日志。")
			return nil
		}

		if follow {
			return followLogs(cmd.Context(), logsFile, tailLines)
		}

		return showLogs(logsFile, tailLines)
	},
}

func init() {
	logsCmd.Flags().BoolP("follow", "f", false, "跟踪日志输出")
	logsCmd.Flags().IntP("tail", "t", defaultTailLines, "只显示最后 N 行，默认值: 1000（出于性能考虑）")
}

func followLogs(ctx context.Context, logsFile string, tailLines int) error {
	t, err := tail.TailFile(logsFile, tail.Config{
		Follow: false,
		ReOpen: false,
		Logger: tail.DiscardingLogger,
	})
	if err != nil {
		return fmt.Errorf("无法追踪日志文件: %v", err)
	}

	var lines []string
	for line := range t.Lines {
		if line.Err != nil {
			continue
		}
		lines = append(lines, line.Text)
		if len(lines) > tailLines {
			lines = lines[len(lines)-tailLines:]
		}
	}
	t.Stop()

	for _, line := range lines {
		printLogLine(line)
	}

	if len(lines) == tailLines {
		fmt.Fprintf(os.Stderr, "\n显示最后 %d 行。完整日志位于: %s\n", tailLines, logsFile)
		fmt.Fprintf(os.Stderr, "正在跟踪新的日志条目...\n\n")
	}

	t, err = tail.TailFile(logsFile, tail.Config{
		Follow:   true,
		ReOpen:   true,
		Logger:   tail.DiscardingLogger,
		Location: &tail.SeekInfo{Offset: 0, Whence: io.SeekEnd},
	})
	if err != nil {
		return fmt.Errorf("无法追踪日志文件: %v", err)
	}
	defer t.Stop()

	for {
		select {
		case line := <-t.Lines:
			if line.Err != nil {
				continue
			}
			printLogLine(line.Text)
		case <-ctx.Done():
			return nil
		}
	}
}

func showLogs(logsFile string, tailLines int) error {
	t, err := tail.TailFile(logsFile, tail.Config{
		Follow:      false,
		ReOpen:      false,
		Logger:      tail.DiscardingLogger,
		MaxLineSize: 0,
	})
	if err != nil {
		return fmt.Errorf("无法追踪日志文件: %v", err)
	}
	defer t.Stop()

	var lines []string
	for line := range t.Lines {
		if line.Err != nil {
			continue
		}
		lines = append(lines, line.Text)
		if len(lines) > tailLines {
			lines = lines[len(lines)-tailLines:]
		}
	}

	for _, line := range lines {
		printLogLine(line)
	}

	if len(lines) == tailLines {
		fmt.Fprintf(os.Stderr, "\n显示最后 %d 行。完整日志位于: %s\n", tailLines, logsFile)
	}

	return nil
}

func printLogLine(lineText string) {
	var data map[string]any
	if err := json.Unmarshal([]byte(lineText), &data); err != nil {
		return
	}
	msg := data["msg"]
	level := data["level"]
	var otherData []any
	var keys []string
	for k := range data {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		switch k {
		case "msg", "level", "time":
			continue
		case "source":
			source, ok := data[k].(map[string]any)
			if !ok {
				continue
			}
			sourceFile := fmt.Sprintf("%s:%d", source["file"], int(source["line"].(float64)))
			otherData = append(otherData, "source", sourceFile)

		default:
			otherData = append(otherData, k, data[k])
		}
	}
	log.SetTimeFunction(func(_ time.Time) time.Time {
		// parse the timestamp from the log line if available
		t, err := time.Parse(time.RFC3339, data["time"].(string))
		if err != nil {
			return time.Now() // fallback to current time if parsing fails
		}
		return t
	})
	switch level {
	case "INFO":
		log.Info(msg, otherData...)
	case "DEBUG":
		log.Debug(msg, otherData...)
	case "ERROR":
		log.Error(msg, otherData...)
	case "WARN":
		log.Warn(msg, otherData...)
	default:
		log.Info(msg, otherData...)
	}
}
