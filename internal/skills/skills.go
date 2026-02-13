// Package skills 实现了Agent Skills开放标准
// 规范详见 https://agentskills.io
package skills

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/charlievieth/fastwalk"
	"gopkg.in/yaml.v3"
)

// 常量定义
const (
	SkillFileName          = "SKILL.md"  // 技能文件名
	MaxNameLength          = 64          // 名称最大长度
	MaxDescriptionLength   = 1024        // 描述最大长度
	MaxCompatibilityLength = 500         // 兼容性说明最大长度
)

// namePattern 技能名称的正则表达式模式
// 要求：字母数字开头，可包含连字符，但不能有前导/尾随/连续的连字符
var namePattern = regexp.MustCompile(`^[a-zA-Z0-9]+(-[a-zA-Z0-9]+)*$`)

// Skill 表示解析后的SKILL.md文件内容
type Skill struct {
	Name          string            `yaml:"name" json:"name"`                              // 技能名称
	Description   string            `yaml:"description" json:"description"`                // 技能描述
	License       string            `yaml:"license,omitempty" json:"license,omitempty"`    // 许可证
	Compatibility string            `yaml:"compatibility,omitempty" json:"compatibility,omitempty"`  // 兼容性说明
	Metadata      map[string]string `yaml:"metadata,omitempty" json:"metadata,omitempty"`  // 元数据
	Instructions  string            `yaml:"-" json:"instructions"`                         // 技能指令（正文内容）
	Path          string            `yaml:"-" json:"path"`                                 // 技能目录路径
	SkillFilePath string            `yaml:"-" json:"skill_file_path"`                      // SKILL.md文件的完整路径
}

// Validate 检查技能是否符合规范要求
// 返回验证过程中发现的所有错误
func (s *Skill) Validate() error {
	var errs []error

	// 验证名称
	if s.Name == "" {
		errs = append(errs, errors.New("名称是必填项"))
	} else {
		if len(s.Name) > MaxNameLength {
			errs = append(errs, fmt.Errorf("名称超过%d个字符", MaxNameLength))
		}
		if !namePattern.MatchString(s.Name) {
			errs = append(errs, errors.New("名称必须是字母数字，可包含连字符，但不能有前导/尾随/连续的连字符"))
		}
		if s.Path != "" && !strings.EqualFold(filepath.Base(s.Path), s.Name) {
			errs = append(errs, fmt.Errorf("名称%q必须与目录%q匹配", s.Name, filepath.Base(s.Path)))
		}
	}

	// 验证描述
	if s.Description == "" {
		errs = append(errs, errors.New("描述是必填项"))
	} else if len(s.Description) > MaxDescriptionLength {
		errs = append(errs, fmt.Errorf("描述超过%d个字符", MaxDescriptionLength))
	}

	// 验证兼容性说明长度
	if len(s.Compatibility) > MaxCompatibilityLength {
		errs = append(errs, fmt.Errorf("兼容性说明超过%d个字符", MaxCompatibilityLength))
	}

	return errors.Join(errs...)
}

// Parse 解析SKILL.md文件
// 参数:
//   - path: SKILL.md文件的路径
// 返回值: 解析后的Skill对象和可能的错误
func Parse(path string) (*Skill, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// 分离前置元数据和正文
	frontmatter, body, err := splitFrontmatter(string(content))
	if err != nil {
		return nil, err
	}

	var skill Skill
	if err := yaml.Unmarshal([]byte(frontmatter), &skill); err != nil {
		return nil, fmt.Errorf("解析前置元数据失败: %w", err)
	}

	skill.Instructions = strings.TrimSpace(body)
	skill.Path = filepath.Dir(path)
	skill.SkillFilePath = path

	return &skill, nil
}

// splitFrontmatter 从Markdown内容中提取YAML前置元数据和正文
// 参数:
//   - content: Markdown文件内容
// 返回值:
//   - frontmatter: YAML前置元数据部分
//   - body: 正文部分
//   - err: 解析错误
func splitFrontmatter(content string) (frontmatter, body string, err error) {
	// 将换行符统一为\n以便一致解析
	content = strings.ReplaceAll(content, "\r\n", "\n")
	if !strings.HasPrefix(content, "---\n") {
		return "", "", errors.New("未找到YAML前置元数据")
	}

	rest := strings.TrimPrefix(content, "---\n")
	before, after, ok := strings.Cut(rest, "\n---")
	if !ok {
		return "", "", errors.New("前置元数据未正确闭合")
	}

	return before, after, nil
}

// Discover 在给定路径中查找所有有效的技能
// 参数:
//   - paths: 要搜索的目录路径列表
// 返回值: 发现的所有有效技能列表
func Discover(paths []string) []*Skill {
	var skills []*Skill
	var mu sync.Mutex
	seen := make(map[string]bool)

	for _, base := range paths {
		// 我们使用fastwalk并设置Follow: true而不是filepath.WalkDir，
		// 因为WalkDir不会跟随任何深度的符号链接目录——只跟随入口点。
		// 这确保了符号链接子目录中的技能也能被发现。
		// fastwalk是并发的，所以我们用mu保护共享状态（seen, skills）。
		conf := fastwalk.Config{
			Follow:  true,
			ToSlash: fastwalk.DefaultToSlash(),
		}
		fastwalk.Walk(&conf, base, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() || d.Name() != SkillFileName {
				return nil
			}
			mu.Lock()
			if seen[path] {
				mu.Unlock()
				return nil
			}
			seen[path] = true
			mu.Unlock()
			skill, err := Parse(path)
			if err != nil {
				slog.Warn("解析技能文件失败", "path", path, "error", err)
				return nil
			}
			if err := skill.Validate(); err != nil {
				slog.Warn("技能验证失败", "path", path, "error", err)
				return nil
			}
			slog.Debug("成功加载技能", "name", skill.Name, "path", path)
			mu.Lock()
			skills = append(skills, skill)
			mu.Unlock()
			return nil
		})
	}

	return skills
}

// ToPromptXML 生成用于注入系统提示的XML格式
// 参数:
//   - skills: 技能列表
// 返回值: XML格式的字符串
func ToPromptXML(skills []*Skill) string {
	if len(skills) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("<available_skills>\n")
	for _, s := range skills {
		sb.WriteString("  <skill>\n")
		fmt.Fprintf(&sb, "    <name>%s</name>\n", escape(s.Name))
		fmt.Fprintf(&sb, "    <description>%s</description>\n", escape(s.Description))
		fmt.Fprintf(&sb, "    <location>%s</location>\n", escape(s.SkillFilePath))
		sb.WriteString("  </skill>\n")
	}
	sb.WriteString("</available_skills>")
	return sb.String()
}

// escape 转义XML特殊字符
func escape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;", "'", "&apos;")
	return r.Replace(s)
}
