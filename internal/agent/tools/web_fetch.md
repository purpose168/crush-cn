从网络 URL 获取内容（供子代理使用）。

<usage>
- 提供要获取的 URL
- 该工具获取内容并将其作为 markdown 返回
- 当您需要从当前页面跟踪链接时使用此工具
- 获取后，分析内容以回答用户的问题
</usage>

<features>
- 自动将 HTML 转换为 markdown 以便于分析
- 对于大页面（>50KB），将内容保存到临时文件并提供路径
- 然后您可以使用 grep/view 工具在文件中搜索
- 处理 UTF-8 内容验证
</features>

<limitations>
- 最大响应大小：5MB
- 仅支持 HTTP 和 HTTPS 协议
- 无法处理身份验证或 cookie
- 某些网站可能会阻止自动请求
</limitations>

<tips>
- 对于保存到文件的大页面，首先使用 grep 查找相关部分
- 不要获取不必要的页面 - 仅在需要回答问题时获取
- 专注于从获取的内容中提取特定信息
</tips>
