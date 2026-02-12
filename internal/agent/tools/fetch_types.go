package tools

// AgenticFetchToolName 是智能抓取工具的名称
const AgenticFetchToolName = "agentic_fetch"

// WebFetchToolName 是网络抓取工具的名称
const WebFetchToolName = "web_fetch"

// WebSearchToolName 是子代理的网络搜索工具名称
const WebSearchToolName = "web_search"

// LargeContentThreshold 是将内容保存到文件的大小阈值
const LargeContentThreshold = 50000 // 50KB

// AgenticFetchParams 定义智能抓取工具的参数
type AgenticFetchParams struct {
	URL    string `json:"url,omitempty" description:"要抓取内容的URL（可选 - 如果未提供，代理将搜索网络）"`
	Prompt string `json:"prompt" description:"描述要查找或提取的信息的提示词"`
}

// AgenticFetchPermissionsParams 定义智能抓取工具的权限参数
type AgenticFetchPermissionsParams struct {
	URL    string `json:"url,omitempty"`
	Prompt string `json:"prompt"`
}

// WebFetchParams 定义网络抓取工具的参数
type WebFetchParams struct {
	URL string `json:"url" description:"要抓取内容的URL"`
}

// WebSearchParams 定义网络搜索工具的参数
type WebSearchParams struct {
	Query      string `json:"query" description:"在网络上查找信息的搜索查询"`
	MaxResults int    `json:"max_results,omitempty" description:"要返回的最大结果数（默认：10，最大：20）"`
}

// FetchParams 定义简单抓取工具的参数
type FetchParams struct {
	URL     string `json:"url" description:"要抓取内容的URL"`
	Format  string `json:"format" description:"返回内容的格式（text、markdown或html）"`
	Timeout int    `json:"timeout,omitempty" description:"可选的超时时间（秒），最大120"`
}

// FetchPermissionsParams 定义简单抓取工具的权限参数
type FetchPermissionsParams struct {
	URL     string `json:"url"`
	Format  string `json:"format"`
	Timeout int    `json:"timeout,omitempty"`
}
