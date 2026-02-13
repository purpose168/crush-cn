// Package copilot 提供 GitHub Copilot OAuth 认证相关的功能
// 该包包含 Copilot 服务所需的 URL 常量定义和认证配置
package copilot

const (
	// SignupURL 是 GitHub Copilot 的注册页面地址
	// 用户通过此链接可以注册 GitHub Copilot 服务
	// 参数 editor=crush 用于标识客户端类型
	SignupURL = "https://github.com/github-copilot/signup?editor=crush"

	// FreeURL 是 GitHub Copilot Pro 免费访问指南的文档地址
	// 该文档说明了如何获取 Copilot Pro 的免费访问权限
	// 包括符合条件的用户群体（如学生、开源项目维护者等）
	FreeURL = "https://docs.github.com/en/copilot/how-tos/manage-your-account/get-free-access-to-copilot-pro"
)
