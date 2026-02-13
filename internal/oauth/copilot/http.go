package copilot

// HTTP 请求相关的常量配置
// 这些常量用于构建与 GitHub Copilot API 通信时所需的 HTTP 头信息
const (
	// userAgent: 用户代理字符串，标识客户端为 GitHub Copilot Chat
	// 用于服务器识别客户端类型和版本
	userAgent = "GitHubCopilotChat/0.32.4"

	// editorVersion: 编辑器版本标识
	// 表示当前使用的编辑器为 Visual Studio Code 1.105.1 版本
	editorVersion = "vscode/1.105.1"

	// editorPluginVersion: 编辑器插件版本
	// 表示 Copilot Chat 插件的版本号为 0.32.4
	editorPluginVersion = "copilot-chat/0.32.4"

	// integrationID: 集成标识符
	// 用于标识此客户端集成的来源为 VS Code 的聊天功能
	integrationID = "vscode-chat"
)

// Headers 返回用于 GitHub Copilot API 请求的标准 HTTP 头信息映射
//
// 返回值:
//   - map[string]string: 包含以下键值对的映射:
//     - "User-Agent": 用户代理字符串，标识客户端类型
//     - "Editor-Version": 编辑器版本信息
//     - "Editor-Plugin-Version": 编辑器插件版本信息
//     - "Copilot-Integration-Id": Copilot 集成标识符
//
// 这些头信息用于在请求 GitHub Copilot API 时进行身份验证和客户端识别，
// 确保请求能够被服务器正确处理和授权。
func Headers() map[string]string {
	return map[string]string{
		"User-Agent":             userAgent,           // 用户代理：标识客户端为 GitHub Copilot Chat
		"Editor-Version":         editorVersion,       // 编辑器版本：标识使用的编辑器版本
		"Editor-Plugin-Version":  editorPluginVersion, // 编辑器插件版本：标识 Copilot Chat 插件版本
		"Copilot-Integration-Id": integrationID,       // Copilot 集成ID：标识集成来源
	}
}
