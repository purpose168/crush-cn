package shell

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCommandBlocking(t *testing.T) {
	// 测试命令拦截功能的测试用例
	tests := []struct {
		name        string // 测试用例名称
		blockFuncs  []BlockFunc // 拦截函数列表
		command     string // 要执行的命令
		shouldBlock bool // 是否应该被拦截
	}{
		{
			name: "拦截简单命令",
			blockFuncs: []BlockFunc{
				func(args []string) bool {
					// 如果命令的第一个参数是"curl"，则拦截该命令
					return len(args) > 0 && args[0] == "curl"
				},
			},
			command:     "curl https://example.com",
			shouldBlock: true,
		},
		{
			name: "允许非拦截命令",
			blockFuncs: []BlockFunc{
				func(args []string) bool {
					// 如果命令的第一个参数是"curl"，则拦截该命令
					return len(args) > 0 && args[0] == "curl"
				},
			},
			command:     "echo hello",
			shouldBlock: false,
		},
		{
			name: "拦截子命令",
			blockFuncs: []BlockFunc{
				func(args []string) bool {
					// 如果命令是"brew install"，则拦截该命令
					return len(args) >= 2 && args[0] == "brew" && args[1] == "install"
				},
			},
			command:     "brew install wget",
			shouldBlock: true,
		},
		{
			name: "允许不同的子命令",
			blockFuncs: []BlockFunc{
				func(args []string) bool {
					// 如果命令是"brew install"，则拦截该命令
					return len(args) >= 2 && args[0] == "brew" && args[1] == "install"
				},
			},
			command:     "brew list",
			shouldBlock: false,
		},
		{
			name: "拦截使用 -g 参数的 npm 全局安装",
			blockFuncs: []BlockFunc{
				ArgumentsBlocker("npm", []string{"install"}, []string{"-g"}),
			},
			command:     "npm install -g typescript",
			shouldBlock: true,
		},
		{
			name: "拦截使用 --global 参数的 npm 全局安装",
			blockFuncs: []BlockFunc{
				ArgumentsBlocker("npm", []string{"install"}, []string{"--global"}),
			},
			command:     "npm install --global typescript",
			shouldBlock: true,
		},
		{
			name: "允许 npm 本地安装",
			blockFuncs: []BlockFunc{
				ArgumentsBlocker("npm", []string{"install"}, []string{"-g"}),
				ArgumentsBlocker("npm", []string{"install"}, []string{"--global"}),
			},
			command:     "npm install typescript",
			shouldBlock: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 为每个测试创建临时目录
			tmpDir := t.TempDir()

			shell := NewShell(&Options{
				WorkingDir: tmpDir,
				BlockFuncs: tt.blockFuncs,
			})

			_, _, err := shell.Exec(t.Context(), tt.command)

			if tt.shouldBlock {
				if err == nil {
					t.Errorf("预期命令应被拦截，但实际被允许执行")
				} else if !strings.Contains(err.Error(), "not allowed for security reasons") {
					t.Errorf("预期安全错误，但得到: %v", err)
				}
			} else {
				// 对于非拦截命令，可能会遇到其他错误（如命令未找到）
				// 但不应该收到安全错误
				if err != nil && strings.Contains(err.Error(), "not allowed for security reasons") {
					t.Errorf("命令意外被拦截: %v", err)
				}
			}
		})
	}
}

func TestArgumentsBlocker(t *testing.T) {
	// 测试基于参数的命令拦截功能
	tests := []struct {
		name        string // 测试用例名称
		cmd         string // 命令名称
		args        []string // 命令参数
		flags       []string // 命令标志
		input       []string // 输入参数
		shouldBlock bool // 是否应该被拦截
	}{
		// 基本命令拦截测试
		{
			name:        "拦截精确匹配的命令",
			cmd:         "npm",
			args:        []string{"install"},
			flags:       nil,
			input:       []string{"npm", "install", "package"},
			shouldBlock: true,
		},
		{
			name:        "允许不同的命令",
			cmd:         "npm",
			args:        []string{"install"},
			flags:       nil,
			input:       []string{"yarn", "install", "package"},
			shouldBlock: false,
		},
		{
			name:        "允许不同的子命令",
			cmd:         "npm",
			args:        []string{"install"},
			flags:       nil,
			input:       []string{"npm", "list"},
			shouldBlock: false,
		},

		// 基于标志的拦截测试
		{
			name:        "拦截带单个标志的命令",
			cmd:         "npm",
			args:        []string{"install"},
			flags:       []string{"-g"},
			input:       []string{"npm", "install", "-g", "typescript"},
			shouldBlock: true,
		},
		{
			name:        "拦截标志在不同位置的命令",
			cmd:         "npm",
			args:        []string{"install"},
			flags:       []string{"-g"},
			input:       []string{"npm", "install", "typescript", "-g"},
			shouldBlock: true,
		},
		{
			name:        "允许不带必需标志的命令",
			cmd:         "npm",
			args:        []string{"install"},
			flags:       []string{"-g"},
			input:       []string{"npm", "install", "typescript"},
			shouldBlock: false,
		},
		{
			name:        "拦截带多个标志的命令",
			cmd:         "pip",
			args:        []string{"install"},
			flags:       []string{"--user"},
			input:       []string{"pip", "install", "--user", "--upgrade", "package"},
			shouldBlock: true,
		},

		// 复杂参数模式测试
		{
			name:        "拦截多参数子命令",
			cmd:         "yarn",
			args:        []string{"global", "add"},
			flags:       nil,
			input:       []string{"yarn", "global", "add", "typescript"},
			shouldBlock: true,
		},
		{
			name:        "允许部分多参数匹配",
			cmd:         "yarn",
			args:        []string{"global", "add"},
			flags:       nil,
			input:       []string{"yarn", "global", "list"},
			shouldBlock: false,
		},

		// 边界情况测试
		{
			name:        "处理空输入",
			cmd:         "npm",
			args:        []string{"install"},
			flags:       nil,
			input:       []string{},
			shouldBlock: false,
		},
		{
			name:        "处理仅命令的情况",
			cmd:         "npm",
			args:        []string{"install"},
			flags:       nil,
			input:       []string{"npm"},
			shouldBlock: false,
		},
		{
			name:        "拦截带 -S 标志的 pacman 命令",
			cmd:         "pacman",
			args:        nil,
			flags:       []string{"-S"},
			input:       []string{"pacman", "-S", "package"},
			shouldBlock: true,
		},
		{
			name:        "允许不带 -S 标志的 pacman 命令",
			cmd:         "pacman",
			args:        nil,
			flags:       []string{"-S"},
			input:       []string{"pacman", "-Q", "package"},
			shouldBlock: false,
		},

		// `go test -exec` 测试
		{
			name:        "go test exec",
			cmd:         "go",
			args:        []string{"test"},
			flags:       []string{"-exec"},
			input:       []string{"go", "test", "-exec", "bash -c 'echo hello'"},
			shouldBlock: true,
		},
		{
			name:        "go test exec",
			cmd:         "go",
			args:        []string{"test"},
			flags:       []string{"-exec"},
			input:       []string{"go", "test", `-exec="bash -c 'echo hello'"`},
			shouldBlock: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocker := ArgumentsBlocker(tt.cmd, tt.args, tt.flags)
			result := blocker(tt.input)
			require.Equal(t, tt.shouldBlock, result,
				"预期 block=%v，输入 %v", tt.shouldBlock, tt.input)
		})
	}
}

func TestCommandsBlocker(t *testing.T) {
	// 测试命令黑名单拦截功能
	tests := []struct {
		name        string // 测试用例名称
		banned      []string // 被禁止的命令列表
		input       []string // 输入参数
		shouldBlock bool // 是否应该被拦截
	}{
		{
			name:        "拦截单个被禁止的命令",
			banned:      []string{"curl"},
			input:       []string{"curl", "https://example.com"},
			shouldBlock: true,
		},
		{
			name:        "允许非禁止命令",
			banned:      []string{"curl", "wget"},
			input:       []string{"echo", "hello"},
			shouldBlock: false,
		},
		{
			name:        "从多个禁止命令中拦截",
			banned:      []string{"curl", "wget", "nc"},
			input:       []string{"wget", "https://example.com"},
			shouldBlock: true,
		},
		{
			name:        "处理空输入",
			banned:      []string{"curl"},
			input:       []string{},
			shouldBlock: false,
		},
		{
			name:        "区分大小写匹配",
			banned:      []string{"curl"},
			input:       []string{"CURL", "https://example.com"},
			shouldBlock: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocker := CommandsBlocker(tt.banned)
			result := blocker(tt.input)
			require.Equal(t, tt.shouldBlock, result,
				"预期 block=%v，输入 %v", tt.shouldBlock, tt.input)
		})
	}
}

func TestSplitArgsFlags(t *testing.T) {
	// 测试参数和标志分离功能
	tests := []struct {
		name      string // 测试用例名称
		input     []string // 输入参数
		wantArgs  []string // 期望的参数
		wantFlags []string // 期望的标志
	}{
		{
			name:      "仅包含参数",
			input:     []string{"install", "package", "another"},
			wantArgs:  []string{"install", "package", "another"},
			wantFlags: []string{},
		},
		{
			name:      "仅包含标志",
			input:     []string{"-g", "--verbose", "-f"},
			wantArgs:  []string{},
			wantFlags: []string{"-g", "--verbose", "-f"},
		},
		{
			name:      "参数和标志混合",
			input:     []string{"install", "-g", "package", "--verbose"},
			wantArgs:  []string{"install", "package"},
			wantFlags: []string{"-g", "--verbose"},
		},
		{
			name:      "空输入",
			input:     []string{},
			wantArgs:  []string{},
			wantFlags: []string{},
		},
		{
			name:      "单短划线标志",
			input:     []string{"-S", "package"},
			wantArgs:  []string{"package"},
			wantFlags: []string{"-S"},
		},
		{
			name:      "带等号的标志",
			input:     []string{"-exec=bash", "package"},
			wantArgs:  []string{"package"},
			wantFlags: []string{"-exec"},
		},
		{
			name:      "带等号的长标志",
			input:     []string{"--config=/path/to/config", "run"},
			wantArgs:  []string{"run"},
			wantFlags: []string{"--config"},
		},
		{
			name:      "带复杂值的标志",
			input:     []string{`-exec="bash -c 'echo hello'"`, "test"},
			wantArgs:  []string{"test"},
			wantFlags: []string{"-exec"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, flags := splitArgsFlags(tt.input)
			require.Equal(t, tt.wantArgs, args, "参数不匹配")
			require.Equal(t, tt.wantFlags, flags, "标志不匹配")
		})
	}
}
