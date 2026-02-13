package commands

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/purpose168/crush-cn/internal/agent/tools/mcp"
	"github.com/purpose168/crush-cn/internal/config"
	"github.com/purpose168/crush-cn/internal/home"
)

var namedArgPattern = regexp.MustCompile(`\$([A-Z][A-Z0-9_]*)`)

const (
	userCommandPrefix    = "user:"
	projectCommandPrefix = "project:"
)

// Argument 表示命令参数及其元数据。
type Argument struct {
	ID          string
	Title       string
	Description string
	Required    bool
}

// MCPPrompt 表示从 MCP 服务器加载的自定义命令。
type MCPPrompt struct {
	ID          string
	Title       string
	Description string
	PromptID    string
	ClientID    string
	Arguments   []Argument
}

// CustomCommand 表示从 markdown 文件加载的用户自定义命令。
type CustomCommand struct {
	ID        string
	Name      string
	Content   string
	Arguments []Argument
}

type commandSource struct {
	path   string
	prefix string
}

// LoadCustomCommands 从多个源加载自定义命令，包括
// XDG 配置目录、主目录和项目目录。
func LoadCustomCommands(cfg *config.Config) ([]CustomCommand, error) {
	return loadAll(buildCommandSources(cfg))
}

// LoadMCPPrompts 从可用的 MCP 服务器加载自定义命令。
func LoadMCPPrompts() ([]MCPPrompt, error) {
	var commands []MCPPrompt
	for mcpName, prompts := range mcp.Prompts() {
		for _, prompt := range prompts {
			key := mcpName + ":" + prompt.Name
			var args []Argument
			for _, arg := range prompt.Arguments {
				title := arg.Title
				if title == "" {
					title = arg.Name
				}
				args = append(args, Argument{
					ID:          arg.Name,
					Title:       title,
					Description: arg.Description,
					Required:    arg.Required,
				})
			}
			commands = append(commands, MCPPrompt{
				ID:          key,
				Title:       prompt.Title,
				Description: prompt.Description,
				PromptID:    prompt.Name,
				ClientID:    mcpName,
				Arguments:   args,
			})
		}
	}
	return commands, nil
}

func buildCommandSources(cfg *config.Config) []commandSource {
	var sources []commandSource

	// XDG 配置目录
	if dir := getXDGCommandsDir(); dir != "" {
		sources = append(sources, commandSource{
			path:   dir,
			prefix: userCommandPrefix,
		})
	}

	// 主目录
	if home := home.Dir(); home != "" {
		sources = append(sources, commandSource{
			path:   filepath.Join(home, ".crush", "commands"),
			prefix: userCommandPrefix,
		})
	}

	// 项目目录
	sources = append(sources, commandSource{
		path:   filepath.Join(cfg.Options.DataDirectory, "commands"),
		prefix: projectCommandPrefix,
	})

	return sources
}

func loadAll(sources []commandSource) ([]CustomCommand, error) {
	var commands []CustomCommand

	for _, source := range sources {
		if cmds, err := loadFromSource(source); err == nil {
			commands = append(commands, cmds...)
		}
	}

	return commands, nil
}

func loadFromSource(source commandSource) ([]CustomCommand, error) {
	if err := ensureDir(source.path); err != nil {
		return nil, err
	}

	var commands []CustomCommand

	err := filepath.WalkDir(source.path, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !isMarkdownFile(d.Name()) {
			return err
		}

		cmd, err := loadCommand(path, source.path, source.prefix)
		if err != nil {
			return nil // 跳过无效文件
		}

		commands = append(commands, cmd)
		return nil
	})

	return commands, err
}

func loadCommand(path, baseDir, prefix string) (CustomCommand, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return CustomCommand{}, err
	}

	id := buildCommandID(path, baseDir, prefix)

	return CustomCommand{
		ID:        id,
		Name:      id,
		Content:   string(content),
		Arguments: extractArgNames(string(content)),
	}, nil
}

func extractArgNames(content string) []Argument {
	matches := namedArgPattern.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]bool)
	var args []Argument

	for _, match := range matches {
		arg := match[1]
		if !seen[arg] {
			seen[arg] = true
			// 对于普通自定义命令，所有参数都是必需的
			args = append(args, Argument{ID: arg, Title: arg, Required: true})
		}
	}

	return args
}

func buildCommandID(path, baseDir, prefix string) string {
	relPath, _ := filepath.Rel(baseDir, path)
	parts := strings.Split(relPath, string(filepath.Separator))

	// 从最后一部分移除 .md 扩展名
	if len(parts) > 0 {
		lastIdx := len(parts) - 1
		parts[lastIdx] = strings.TrimSuffix(parts[lastIdx], filepath.Ext(parts[lastIdx]))
	}

	return prefix + strings.Join(parts, ":")
}

func getXDGCommandsDir() string {
	xdgHome := os.Getenv("XDG_CONFIG_HOME")
	if xdgHome == "" {
		if home := home.Dir(); home != "" {
			xdgHome = filepath.Join(home, ".config")
		}
	}
	if xdgHome != "" {
		return filepath.Join(xdgHome, "crush", "commands")
	}
	return ""
}

func ensureDir(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.MkdirAll(path, 0o755)
	}
	return nil
}

func isMarkdownFile(name string) bool {
	return strings.HasSuffix(strings.ToLower(name), ".md")
}

func GetMCPPrompt(cfg *config.Config, clientID, promptID string, args map[string]string) (string, error) {
	// TODO: 我们应该向下传递上下文
	result, err := mcp.GetPromptMessages(context.Background(), cfg, clientID, promptID, args)
	if err != nil {
		return "", err
	}
	return strings.Join(result, " "), nil
}
