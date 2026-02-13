package fsext

import (
	"os"
	"strings"

	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/syntax"
)

// Expand 是 [expand.Literal] 的封装函数。它会对输入字符串进行转义，
// 展开所有 shell 符号（如 '~'）并解析所有环境变量。
func Expand(s string) (string, error) {
	if s == "" {
		return "", nil
	}
	p := syntax.NewParser()
	word, err := p.Document(strings.NewReader(s))
	if err != nil {
		return "", err
	}
	cfg := &expand.Config{
		Env:      expand.FuncEnviron(os.Getenv),
		ReadDir2: os.ReadDir,
		GlobStar: true,
	}
	return expand.Literal(cfg, word)
}
