package tools

import "runtime"

// safeCommands 是一个被认为是安全的命令列表，这些命令通常用于信息获取和系统状态查看
var safeCommands = []string{
	// Bash 内置命令和核心工具
	"cal",
	"date",
	"df",
	"du",
	"echo",
	"env",
	"free",
	"groups",
	"hostname",
	"id",
	"kill",
	"killall",
	"ls",
	"nice",
	"nohup",
	"printenv",
	"ps",
	"pwd",
	"set",
	"time",
	"timeout",
	"top",
	"type",
	"uname",
	"unset",
	"uptime",
	"whatis",
	"whereis",
	"which",
	"whoami",

	// Git 命令
	"git blame",
	"git branch",
	"git config --get",
	"git config --list",
	"git describe",
	"git diff",
	"git grep",
	"git log",
	"git ls-files",
	"git ls-remote",
	"git remote",
	"git rev-parse",
	"git shortlog",
	"git show",
	"git status",
	"git tag",
}

// init 函数在包初始化时执行，用于在Windows系统上添加Windows特有的安全命令
func init() {
	if runtime.GOOS == "windows" {
		safeCommands = append(
			safeCommands,
			// Windows 特有命令
			"ipconfig",
			"nslookup",
			"ping",
			"systeminfo",
			"tasklist",
			"where",
		)
	}
}
