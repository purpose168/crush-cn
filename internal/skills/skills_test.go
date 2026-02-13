package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestParse 测试SKILL.md文件解析功能
func TestParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		content     string
		wantName    string
		wantDesc    string
		wantLicense string
		wantCompat  string
		wantMeta    map[string]string
		wantTools   string
		wantInstr   string
		wantErr     bool
	}{
		{
			name: "完整技能",
			content: `---
name: pdf-processing
description: Extracts text and tables from PDF files, fills PDF forms, and merges multiple PDFs.
license: Apache-2.0
compatibility: Requires python 3.8+, pdfplumber, pdfrw libraries
metadata:
  author: example-org
  version: "1.0"
---

# PDF Processing

## When to use this skill
Use this skill when the user needs to work with PDF files.
`,
			wantName:    "pdf-processing",
			wantDesc:    "Extracts text and tables from PDF files, fills PDF forms, and merges multiple PDFs.",
			wantLicense: "Apache-2.0",
			wantCompat:  "Requires python 3.8+, pdfplumber, pdfrw libraries",
			wantMeta:    map[string]string{"author": "example-org", "version": "1.0"},
			wantInstr:   "# PDF Processing\n\n## When to use this skill\nUse this skill when the user needs to work with PDF files.",
		},
		{
			name: "最小技能",
			content: `---
name: my-skill
description: A simple skill for testing.
---

# My Skill

Instructions here.
`,
			wantName:  "my-skill",
			wantDesc:  "A simple skill for testing.",
			wantInstr: "# My Skill\n\nInstructions here.",
		},
		{
			name:    "无前置元数据",
			content: "# Just Markdown\n\nNo frontmatter here.",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// 将内容写入临时文件
			dir := t.TempDir()
			path := filepath.Join(dir, "SKILL.md")
			require.NoError(t, os.WriteFile(path, []byte(tt.content), 0o644))

			skill, err := Parse(path)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			require.Equal(t, tt.wantName, skill.Name)
			require.Equal(t, tt.wantDesc, skill.Description)
			require.Equal(t, tt.wantLicense, skill.License)
			require.Equal(t, tt.wantCompat, skill.Compatibility)

			if tt.wantMeta != nil {
				require.Equal(t, tt.wantMeta, skill.Metadata)
			}

			require.Equal(t, tt.wantInstr, skill.Instructions)
		})
	}
}

// TestSkillValidate 测试技能验证功能
func TestSkillValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		skill   Skill
		wantErr bool
		errMsg  string
	}{
		{
			name: "有效技能",
			skill: Skill{
				Name:        "pdf-processing",
				Description: "Processes PDF files.",
				Path:        "/skills/pdf-processing",
			},
		},
		{
			name:    "缺少名称",
			skill:   Skill{Description: "Some description."},
			wantErr: true,
			errMsg:  "名称是必填项",
		},
		{
			name:    "缺少描述",
			skill:   Skill{Name: "my-skill", Path: "/skills/my-skill"},
			wantErr: true,
			errMsg:  "描述是必填项",
		},
		{
			name:    "名称过长",
			skill:   Skill{Name: strings.Repeat("a", 65), Description: "Some description."},
			wantErr: true,
			errMsg:  "超过",
		},
		{
			name:    "有效名称 - 大小写混合",
			skill:   Skill{Name: "MySkill", Description: "Some description.", Path: "/skills/MySkill"},
			wantErr: false,
		},
		{
			name:    "无效名称 - 以连字符开头",
			skill:   Skill{Name: "-my-skill", Description: "Some description."},
			wantErr: true,
			errMsg:  "字母数字",
		},
		{
			name:    "名称与目录不匹配",
			skill:   Skill{Name: "my-skill", Description: "Some description.", Path: "/skills/other-skill"},
			wantErr: true,
			errMsg:  "必须与目录",
		},
		{
			name:    "描述过长",
			skill:   Skill{Name: "my-skill", Description: strings.Repeat("a", 1025), Path: "/skills/my-skill"},
			wantErr: true,
			errMsg:  "描述超过",
		},
		{
			name:    "兼容性说明过长",
			skill:   Skill{Name: "my-skill", Description: "desc", Compatibility: strings.Repeat("a", 501), Path: "/skills/my-skill"},
			wantErr: true,
			errMsg:  "兼容性说明超过",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.skill.Validate()
			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestDiscover 测试技能发现功能
func TestDiscover(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// 创建有效技能1
	skill1Dir := filepath.Join(tmpDir, "skill-one")
	require.NoError(t, os.MkdirAll(skill1Dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(skill1Dir, "SKILL.md"), []byte(`---
name: skill-one
description: First test skill.
---
# Skill One
`), 0o644))

	// 在嵌套目录中创建有效技能2
	skill2Dir := filepath.Join(tmpDir, "nested", "skill-two")
	require.NoError(t, os.MkdirAll(skill2Dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(skill2Dir, "SKILL.md"), []byte(`---
name: skill-two
description: Second test skill.
---
# Skill Two
`), 0o644))

	// 创建无效技能（不会被包含）
	invalidDir := filepath.Join(tmpDir, "invalid-dir")
	require.NoError(t, os.MkdirAll(invalidDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(invalidDir, "SKILL.md"), []byte(`---
name: wrong-name
description: Name doesn't match directory.
---
`), 0o644))

	skills := Discover([]string{tmpDir})
	require.Len(t, skills, 2)

	names := make(map[string]bool)
	for _, s := range skills {
		names[s.Name] = true
	}
	require.True(t, names["skill-one"])
	require.True(t, names["skill-two"])
}

// TestToPromptXML 测试XML格式生成功能
func TestToPromptXML(t *testing.T) {
	t.Parallel()

	skills := []*Skill{
		{Name: "pdf-processing", Description: "Extracts text from PDFs.", SkillFilePath: "/skills/pdf-processing/SKILL.md"},
		{Name: "data-analysis", Description: "Analyzes datasets & charts.", SkillFilePath: "/skills/data-analysis/SKILL.md"},
	}

	xml := ToPromptXML(skills)

	require.Contains(t, xml, "<available_skills>")
	require.Contains(t, xml, "<name>pdf-processing</name>")
	require.Contains(t, xml, "<description>Extracts text from PDFs.</description>")
	require.Contains(t, xml, "&amp;")  // XML转义
}

// TestToPromptXMLEmpty 测试空技能列表的XML生成
func TestToPromptXMLEmpty(t *testing.T) {
	t.Parallel()
	require.Empty(t, ToPromptXML(nil))
	require.Empty(t, ToPromptXML([]*Skill{}))
}
