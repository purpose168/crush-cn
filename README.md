# Crush

<p align="center">
    <a href="https://stuff.charm.sh/crush/charm-crush.png"><img width="450" alt="Charm Crush Logo" src="https://github.com/user-attachments/assets/cf8ca3ce-8b02-43f0-9d0f-5a331488da4b" /></a><br />
    <a href="https://github.com/charmbracelet/crush/releases"><img src="https://img.shields.io/github/release/charmbracelet/crush" alt="Latest Release"></a>
    <a href="https://github.com/charmbracelet/crush/actions"><img src="https://github.com/charmbracelet/crush/actions/workflows/build.yml/badge.svg" alt="Build Status"></a>
</p>

<p align="center">你编程的新搭档，现已登陆你最爱的终端。<br />无缝集成你的工具、代码与工作流，支持你选择的 LLM 模型。</p>

<p align="center"><img width="800" alt="Crush Demo" src="https://github.com/user-attachments/assets/58280caf-851b-470a-b6f7-d5c4ea8a1968" /></p>

## 特性

- **多模型支持：** 从广泛的 LLM 模型中选择，或通过兼容 OpenAI 或 Anthropic 的 API 添加自定义模型
- **灵活切换：** 在会话中途切换 LLM 模型，同时保留上下文
- **基于会话：** 为每个项目维护多个工作会话和上下文
- **LSP 增强：** Crush 像你一样使用 LSP 提供额外的上下文信息
- **可扩展：** 通过 MCP（`http`、`stdio` 和 `sse`）添加功能
- **全平台支持：** 在 macOS、Linux、Windows（PowerShell 和 WSL）、Android、FreeBSD、OpenBSD 和 NetBSD 的所有终端中提供一流支持
- **工业级：** 构建在 Charm 生态系统之上，为超过 25k 个应用提供支持，从领先的开源项目到关键业务基础设施

## 安装

使用包管理器安装：

```bash
# Homebrew - macOS 和 Linux 上的包管理器
brew install charmbracelet/tap/crush

# NPM - Node.js 包管理器
npm install -g @charmland/crush

# Arch Linux (btw) - 使用 yay AUR 助手
yay -S crush-bin

# Nix - Nix 包管理器
nix run github:numtide/nix-ai-tools#crush

# FreeBSD - FreeBSD 包管理器
pkg install crush
```

Windows 用户：

```bash
# Winget - Windows 包管理器
winget install charmbracelet.crush

# Scoop - Windows 命令行安装工具
scoop bucket add charm https://github.com/charmbracelet/scoop-bucket.git
scoop install crush
```

<details>
<summary><strong>Nix (NUR)</strong></summary>

Crush 可通过官方 Charm [NUR](https://github.com/nix-community/NUR) 仓库获取，路径为 `nur.repos.charmbracelet.crush`，这是在 Nix 中获取最新版 Crush 的最佳方式。

你也可以通过 NUR 使用 `nix-shell` 来尝试 Crush：

```bash
# 添加 NUR 通道
nix-channel --add https://github.com/nix-community/NUR/archive/main.tar.gz nur
nix-channel --update

# 在 Nix shell 中获取 Crush
nix-shell -p '(import <nur> { pkgs = import <nixpkgs> {}; }).repos.charmbracelet.crush'
```

### 通过 NUR 使用 NixOS 和 Home Manager 模块

Crush 通过 NUR 提供 NixOS 和 Home Manager 模块。
你可以在 flake 中直接导入这些模块。由于它会自动检测是 Home Manager 还是 NixOS 环境，所以导入方式完全相同 :) 

```nix
{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    nur.url = "github:nix-community/NUR";
  };

  outputs = { self, nixpkgs, nur, ... }: {
    nixosConfigurations.your-hostname = nixpkgs.lib.nixosSystem {
      system = "x86_64-linux";
      modules = [
        nur.modules.nixos.default
        nur.repos.charmbracelet.modules.crush
        {
          programs.crush = {
            enable = true;
            settings = {
              providers = {
                openai = {
                  id = "openai";
                  name = "OpenAI";
                  base_url = "https://api.openai.com/v1";
                  type = "openai";
                  api_key = "sk-fake123456789abcdef...";
                  models = [
                    {
                      id = "gpt-4";
                      name = "GPT-4";
                    }
                  ];
                };
              };
              lsp = {
                go = { command = "gopls"; enabled = true; };
                nix = { command = "nil"; enabled = true; };
              };
              options = {
                context_paths = [ "/etc/nixos/configuration.nix" ];
                tui = { compact_mode = true; };
                debug = false;
              };
            };
          };
        }
      ];
    };
  };
}
```

</details>

<details>
<summary><strong>Debian/Ubuntu</strong></summary>

```bash
# 创建密钥环目录
sudo mkdir -p /etc/apt/keyrings
# 下载并添加 Charm GPG 密钥
curl -fsSL https://repo.charm.sh/apt/gpg.key | sudo gpg --dearmor -o /etc/apt/keyrings/charm.gpg
# 添加 Charm 软件源
echo "deb [signed-by=/etc/apt/keyrings/charm.gpg] https://repo.charm.sh/apt/ * *" | sudo tee /etc/apt/sources.list.d/charm.list
# 更新包列表并安装 Crush
sudo apt update && sudo apt install crush
```

</details>

<details>
<summary><strong>Fedora/RHEL</strong></summary>

```bash
# 创建 Charm YUM 仓库配置文件
echo '[charm]
name=Charm
baseurl=https://repo.charm.sh/yum/
enabled=1
gpgcheck=1
gpgkey=https://repo.charm.sh/yum/gpg.key' | sudo tee /etc/yum.repos.d/charm.repo
# 安装 Crush
sudo yum install crush
```

</details>

或者，直接下载：

- [Packages][releases] 提供 Debian 和 RPM 格式的安装包
- [Binaries][releases] 提供适用于 Linux、macOS、Windows、FreeBSD、OpenBSD 和 NetBSD 的二进制文件

[releases]: https://github.com/charmbracelet/crush/releases

或者使用 Go 安装：

```
# 使用 Go 命令安装最新版 Crush
go install github.com/charmbracelet/crush@latest
```

> [!WARNING]
> 使用 Crush 可能会提高你的工作效率，初次使用时你可能会沉浸其中。如果这种症状持续存在，请加入 [Discord][discord]，与我们一起沉浸其中。

## 快速开始

最快捷的开始方式是获取你首选提供者的 API 密钥，如 Anthropic、OpenAI、Groq、OpenRouter 或 Vercel AI Gateway，然后直接启动 Crush。系统会提示你输入 API 密钥。

此外，你也可以为首选提供者设置环境变量。

| 环境变量                    | 提供者                                            |
| --------------------------- | -------------------------------------------------- |
| `ANTHROPIC_API_KEY`         | Anthropic                                          |
| `OPENAI_API_KEY`            | OpenAI                                             |
| `VERCEL_API_KEY`            | Vercel AI Gateway                                  |
| `GEMINI_API_KEY`            | Google Gemini                                      |
| `SYNTHETIC_API_KEY`         | Synthetic                                          |
| `ZAI_API_KEY`               | Z.ai                                               |
| `MINIMAX_API_KEY`           | MiniMax                                            |
| `HF_TOKEN`                  | Hugging Face Inference                             |
| `CEREBRAS_API_KEY`          | Cerebras                                           |
| `OPENROUTER_API_KEY`        | OpenRouter                                         |
| `GROQ_API_KEY`              | Groq                                               |
| `VERTEXAI_PROJECT`          | Google Cloud VertexAI (Gemini)                     |
| `VERTEXAI_LOCATION`         | Google Cloud VertexAI (Gemini)                     |
| `AWS_ACCESS_KEY_ID`         | Amazon Bedrock (Claude)                            |
| `AWS_SECRET_ACCESS_KEY`     | Amazon Bedrock (Claude)                            |
| `AWS_REGION`                | Amazon Bedrock (Claude)                            |
| `AWS_PROFILE`               | Amazon Bedrock (Custom Profile)                    |
| `AWS_BEARER_TOKEN_BEDROCK`  | Amazon Bedrock                                     |
| `AZURE_OPENAI_API_ENDPOINT` | Azure OpenAI models                                |
| `AZURE_OPENAI_API_KEY`      | Azure OpenAI models (optional when using Entra ID) |
| `AZURE_OPENAI_API_VERSION`  | Azure OpenAI models                                |

### 顺便一提

你希望在 Crush 中看到某个提供者吗？是否有现有的模型需要更新？

Crush 的默认模型列表由 [Catwalk](https://github.com/charmbracelet/catwalk) 管理，这是一个社区支持的、兼容 Crush 的开源模型仓库，欢迎你贡献。

<a href="https://github.com/charmbracelet/catwalk"><img width="174" height="174" alt="Catwalk Badge" src="https://github.com/user-attachments/assets/95b49515-fe82-4409-b10d-5beb0873787d" /></a>

## 配置

Crush 无需配置即可正常运行。当然，如果你需要或想要自定义 Crush，可以在项目本地或全局添加配置，优先级如下：

1. `.crush.json`
2. `crush.json`
3. `$HOME/.config/crush/crush.json`

配置本身以 JSON 对象形式存储：

```json
{
  "this-setting": { "this": "that" },
  "that-setting": ["ceci", "cela"]
}
```

另外，Crush 还在以下位置存储临时数据，如应用状态：

```bash
# Unix - Unix 系统上的临时数据存储位置
$HOME/.local/share/crush/crush.json

# Windows - Windows 系统上的临时数据存储位置
%LOCALAPPDATA%\crush\crush.json
```

> [!TIP]
> 你可以通过设置以下环境变量来覆盖用户和数据配置位置：
> * `CRUSH_GLOBAL_CONFIG`
> * `CRUSH_GLOBAL_DATA`

### LSP

Crush 可以使用 LSP（语言服务器协议）获取额外的上下文信息，帮助它做出决策，就像你一样。你可以手动添加 LSP：

```json
{
  "$schema": "https://charm.land/crush.json",
  "lsp": {
    "go": {
      "command": "gopls",  // Go 语言服务器命令
      "env": {
        "GOTOOLCHAIN": "go1.24.5"  // 设置 Go 工具链版本
      }
    },
    "typescript": {
      "command": "typescript-language-server",  // TypeScript 语言服务器命令
      "args": ["--stdio"]  // 使用标准输入输出模式
    },
    "nix": {
      "command": "nil"  // Nix 语言服务器命令
    }
  }
}
```

### MCP

Crush 还支持通过三种传输类型的 Model Context Protocol (MCP) 服务器：`stdio` 用于命令行服务器，`http` 用于 HTTP 端点，`sse` 用于服务器发送事件。支持使用 `$(echo $VAR)` 语法进行环境变量展开。

```json
{
  "$schema": "https://charm.land/crush.json",
  "mcp": {
    "filesystem": {
      "type": "stdio",  // 使用标准输入输出传输
      "command": "node",  // 运行 Node.js 命令
      "args": ["/path/to/mcp-server.js"],  // MCP 服务器脚本路径
      "timeout": 120,  // 超时时间（秒）
      "disabled": false,  // 是否禁用
      "disabled_tools": ["some-tool-name"],  // 禁用的工具列表
      "env": {
        "NODE_ENV": "production"  // 设置环境变量
      }
    },
    "github": {
      "type": "http",  // 使用 HTTP 传输
      "url": "https://api.githubcopilot.com/mcp/",  // MCP 服务器 URL
      "timeout": 120,  // 超时时间（秒）
      "disabled": false,  // 是否禁用
      "disabled_tools": ["create_issue", "create_pull_request"],  // 禁用的工具列表
      "headers": {
        "Authorization": "Bearer $GH_PAT"  // 授权头
      }
    },
    "streaming-service": {
      "type": "sse",  // 使用服务器发送事件传输
      "url": "https://example.com/mcp/sse",  // MCP 服务器 SSE URL
      "timeout": 120,  // 超时时间（秒）
      "disabled": false,  // 是否禁用
      "headers": {
        "API-Key": "$(echo $API_KEY)"  // 使用环境变量展开获取 API 密钥
      }
    }
  }
}
```

### 忽略文件

默认情况下，Crush 会尊重 `.gitignore` 文件，但你也可以创建 `.crushignore` 文件来指定 Crush 应该忽略的其他文件和目录。这对于排除你希望保留在版本控制中但不希望 Crush 在提供上下文时考虑的文件很有用。

`.crushignore` 文件使用与 `.gitignore` 相同的语法，可以放在项目根目录或子目录中。

### 允许工具

默认情况下，Crush 在运行工具调用前会请求你的权限。如果需要，你可以允许工具在不提示权限的情况下执行。请谨慎使用此功能。

```json
{
  "$schema": "https://charm.land/crush.json",
  "permissions": {
    "allowed_tools": [  // 允许自动执行的工具列表
      "view",  // 查看文件
      "ls",  // 列出目录
      "grep",  // 搜索内容
      "edit",  // 编辑文件
      "mcp_context7_get-library-doc"  // 获取库文档
    ]
  }
}
```

你也可以通过使用 `--yolo` 标志运行 Crush 来完全跳过所有权限提示。请非常谨慎地使用此功能。

### 禁用内置工具

如果你想完全阻止 Crush 使用某些内置工具，可以通过 `options.disabled_tools` 列表禁用它们。禁用的工具对代理完全隐藏。

```json
{
  "$schema": "https://charm.land/crush.json",
  "options": {
    "disabled_tools": [  // 禁用的内置工具列表
      "bash",  // 禁用 bash 工具
      "sourcegraph"  // 禁用 Sourcegraph 工具
    ]
  }
}
```

要禁用 MCP 服务器的工具，请参阅 [MCP 配置部分](#mcps)。

### Agent Skills

Crush 支持 [Agent Skills](https://agentskills.io) 开放标准，通过可重用的技能包扩展代理功能。技能是包含 `SKILL.md` 文件的文件夹，其中包含 Crush 可以发现并按需激活的指令。

技能从以下位置发现：

- Unix 系统上的 `~/.config/crush/skills/`（默认，可通过 `CRUSH_SKILLS_DIR` 覆盖）
- Windows 系统上的 `%LOCALAPPDATA%\crush\skills\`（默认，可通过 `CRUSH_SKILLS_DIR` 覆盖）
- 通过 `options.skills_paths` 配置的其他路径

```jsonc
{
  "$schema": "https://charm.land/crush.json",
  "options": {
    "skills_paths": [  // 技能包搜索路径
      "~/.config/crush/skills", // Windows: "%LOCALAPPDATA%\\crush\\skills",
      "./project-skills"  // 项目本地技能路径
    ]
  }
}
```

你可以从 [anthropics/skills](https://github.com/anthropics/skills) 获取示例技能：

```bash
# Unix - 在 Unix 系统上安装示例技能
mkdir -p ~/.config/crush/skills
cd ~/.config/crush/skills
git clone https://github.com/anthropics/skills.git _temp
mv _temp/skills/* . && rm -rf _temp
```

```powershell
# Windows (PowerShell) - 在 Windows PowerShell 上安装示例技能
mkdir -Force "$env:LOCALAPPDATA\crush\skills"
cd "$env:LOCALAPPDATA\crush\skills"
git clone https://github.com/anthropics/skills.git _temp
mv _temp/skills/* . ; rm -r -force _temp
```

### 初始化

当初始化项目时，Crush 会分析你的代码库并创建一个上下文文件，帮助它在未来的会话中更有效地工作。默认情况下，此文件名为 `AGENTS.md`，但你可以使用 `initialize_as` 选项自定义名称和位置：

```json
{
  "$schema": "https://charm.land/crush.json",
  "options": {
    "initialize_as": "AGENTS.md"  // 初始化文件名称
  }
}
```

如果你喜欢不同的命名约定或想将文件放在特定目录（例如 `CRUSH.md` 或 `docs/LLMs.md`），这很有用。Crush 会在文件中填充项目特定的上下文，如构建命令、代码模式和初始化期间发现的约定。

### 归因设置

默认情况下，Crush 会在它创建的 Git 提交和拉取请求中添加归因信息。你可以使用 `attribution` 选项自定义此行为：

```json
{
  "$schema": "https://charm.land/crush.json",
  "options": {
    "attribution": {
      "trailer_style": "co-authored-by",  // 归因尾部样式
      "generated_with": true  // 是否添加生成标记
    }
  }
}
```

- `trailer_style`：控制添加到提交消息的归因尾部（默认：`assisted-by`）
	- `assisted-by`：添加 `Assisted-by: [Model Name] via Crush <crush@charm.land>`（包含模型名称）
	- `co-authored-by`：添加 `Co-Authored-By: Crush <crush@charm.land>`
	- `none`：无归因尾部
- `generated_with`：当为 true（默认）时，在提交消息和 PR 描述中添加 `💘 Generated with Crush` 行

### 自定义提供者

Crush 支持为兼容 OpenAI 和兼容 Anthropic 的 API 配置自定义提供者。

> [!NOTE]
> 请注意，我们支持两种 OpenAI "类型"。请确保选择正确的类型以确保最佳体验！
> * `openai` 应在通过 OpenAI 代理或路由请求时使用。
> * `openai-compat` 应在使用具有 OpenAI 兼容 API 的非 OpenAI 提供者时使用。

#### 兼容 OpenAI 的 API

以下是 Deepseek 的示例配置，它使用兼容 OpenAI 的 API。不要忘记在环境中设置 `DEEPSEEK_API_KEY`。

```json
{
  "$schema": "https://charm.land/crush.json",
  "providers": {
    "deepseek": {
      "type": "openai-compat",  // 使用 OpenAI 兼容模式
      "base_url": "https://api.deepseek.com/v1",  // Deepseek API 基础 URL
      "api_key": "$DEEPSEEK_API_KEY",  // API 密钥（从环境变量获取）
      "models": [
        {
          "id": "deepseek-chat",  // 模型 ID
          "name": "Deepseek V3",  // 模型名称
          "cost_per_1m_in": 0.27,  // 每百万输入令牌的成本
          "cost_per_1m_out": 1.1,  // 每百万输出令牌的成本
          "cost_per_1m_in_cached": 0.07,  // 每百万缓存输入令牌的成本
          "cost_per_1m_out_cached": 1.1,  // 每百万缓存输出令牌的成本
          "context_window": 64000,  // 上下文窗口大小
          "default_max_tokens": 5000  // 默认最大令牌数
        }
      ]
    }
  }
}
```

#### 兼容 Anthropic 的 API

自定义兼容 Anthropic 的提供者遵循以下格式：

```json
{
  "$schema": "https://charm.land/crush.json",
  "providers": {
    "custom-anthropic": {
      "type": "anthropic",  // 使用 Anthropic 模式
      "base_url": "https://api.anthropic.com/v1",  // Anthropic API 基础 URL
      "api_key": "$ANTHROPIC_API_KEY",  // API 密钥（从环境变量获取）
      "extra_headers": {
        "anthropic-version": "2023-06-01"  // 额外的请求头
      },
      "models": [
        {
          "id": "claude-sonnet-4-20250514",  // 模型 ID
          "name": "Claude Sonnet 4",  // 模型名称
          "cost_per_1m_in": 3,  // 每百万输入令牌的成本
          "cost_per_1m_out": 15,  // 每百万输出令牌的成本
          "cost_per_1m_in_cached": 3.75,  // 每百万缓存输入令牌的成本
          "cost_per_1m_out_cached": 0.3,  // 每百万缓存输出令牌的成本
          "context_window": 200000,  // 上下文窗口大小
          "default_max_tokens": 50000,  // 默认最大令牌数
          "can_reason": true,  // 是否支持推理
          "supports_attachments": true  // 是否支持附件
        }
      ]
    }
  }
}
```

### Amazon Bedrock

Crush 目前支持通过 Bedrock 运行 Anthropic 模型，禁用缓存。

- 配置 AWS 后（即 `aws configure`），Bedrock 提供者会出现
- Crush 还期望设置 `AWS_REGION` 或 `AWS_DEFAULT_REGION`
- 要使用特定的 AWS 配置文件，请在环境中设置 `AWS_PROFILE`，例如 `AWS_PROFILE=myprofile crush`
- 除了 `aws configure` 外，你也可以只设置 `AWS_BEARER_TOKEN_BEDROCK`

### Vertex AI Platform

当设置了 `VERTEXAI_PROJECT` 和 `VERTEXAI_LOCATION` 时，Vertex AI 会出现在可用提供者列表中。你还需要进行身份验证：

```bash
# 登录 Google Cloud 应用默认凭据
gcloud auth application-default login
```

要向配置添加特定模型，请按如下方式配置：

```json
{
  "$schema": "https://charm.land/crush.json",
  "providers": {
    "vertexai": {
      "models": [
        {
          "id": "claude-sonnet-4@20250514",  // 模型 ID
          "name": "VertexAI Sonnet 4",  // 模型名称
          "cost_per_1m_in": 3,  // 每百万输入令牌的成本
          "cost_per_1m_out": 15,  // 每百万输出令牌的成本
          "cost_per_1m_in_cached": 3.75,  // 每百万缓存输入令牌的成本
          "cost_per_1m_out_cached": 0.3,  // 每百万缓存输出令牌的成本
          "context_window": 200000,  // 上下文窗口大小
          "default_max_tokens": 50000,  // 默认最大令牌数
          "can_reason": true,  // 是否支持推理
          "supports_attachments": true  // 是否支持附件
        }
      ]
    }
  }
}
```

### 本地模型

本地模型也可以通过兼容 OpenAI 的 API 进行配置。以下是两个常见示例：

#### Ollama

```json
{
  "providers": {
    "ollama": {
      "name": "Ollama",  // 提供者名称
      "base_url": "http://localhost:11434/v1/",  // Ollama API URL
      "type": "openai-compat",  // 使用 OpenAI 兼容模式
      "models": [
        {
          "name": "Qwen 3 30B",  // 模型名称
          "id": "qwen3:30b",  // 模型 ID
          "context_window": 256000,  // 上下文窗口大小
          "default_max_tokens": 20000  // 默认最大令牌数
        }
      ]
    }
  }
}
```

#### LM Studio

```json
{
  "providers": {
    "lmstudio": {
      "name": "LM Studio",  // 提供者名称
      "base_url": "http://localhost:1234/v1/",  // LM Studio API URL
      "type": "openai-compat",  // 使用 OpenAI 兼容模式
      "models": [
        {
          "name": "Qwen 3 30B",  // 模型名称
          "id": "qwen/qwen3-30b-a3b-2507",  // 模型 ID
          "context_window": 256000,  // 上下文窗口大小
          "default_max_tokens": 20000  // 默认最大令牌数
        }
      ]
    }
  }
}
```

## 日志

有时你需要查看日志。幸运的是，Crush 会记录各种信息。日志存储在项目相对路径 `./.crush/logs/crush.log` 中。

CLI 还包含一些辅助命令，使查看最近的日志变得更加容易：

```bash
# 打印最后 1000 行日志
crush logs

# 打印最后 500 行日志
crush logs --tail 500

# 实时跟踪日志
crush logs --follow
```

想要更详细的日志？使用 `--debug` 标志运行 `crush`，或在配置中启用它：

```json
{
  "$schema": "https://charm.land/crush.json",
  "options": {
    "debug": true,  // 启用调试日志
    "debug_lsp": true  // 启用 LSP 调试日志
  }
}
```

## 提供者自动更新

默认情况下，Crush 会自动从 [Catwalk](https://github.com/charmbracelet/catwalk)（开源 Crush 提供者数据库）检查最新的提供者和模型列表。这意味着当有新的提供者和模型可用，或当模型元数据更改时，Crush 会自动更新你的本地配置。

### 禁用自动提供者更新

对于互联网访问受限的用户，或喜欢在隔离环境中工作的用户，这可能不是你想要的功能，此功能可以被禁用。

要禁用自动提供者更新，请在 `crush.json` 配置中设置 `disable_provider_auto_update`：

```json
{
  "$schema": "https://charm.land/crush.json",
  "options": {
    "disable_provider_auto_update": true  // 禁用自动提供者更新
  }
}
```

或设置 `CRUSH_DISABLE_PROVIDER_AUTO_UPDATE` 环境变量：

```bash
# 设置环境变量禁用自动提供者更新
export CRUSH_DISABLE_PROVIDER_AUTO_UPDATE=1
```

### 手动更新提供者

可以使用 `crush update-providers` 命令手动更新提供者：

```bash
# 从 Catwalk 远程更新提供者
crush update-providers

# 从自定义 Catwalk 基础 URL 更新提供者
crush update-providers https://example.com/

# 从本地文件更新提供者
crush update-providers /path/to/local-providers.json

# 将提供者重置为 Crush 构建时嵌入的版本
crush update-providers embedded

# 查看更多信息
crush update-providers --help
```

## 指标

Crush 记录假名使用指标（与设备特定哈希关联），维护者依靠这些指标来指导开发和支持优先级。这些指标仅包含使用元数据；提示和响应永远不会被收集。

关于具体收集内容的详细信息在源代码中（[这里](https://github.com/charmbracelet/crush/tree/main/internal/event)
和 [这里](https://github.com/charmbracelet/crush/blob/main/internal/llm/agent/event.go)）。

你可以通过在环境中设置以下环境变量随时选择退出指标收集：

```bash
# 设置环境变量禁用指标收集
export CRUSH_DISABLE_METRICS=1
```

或在配置中设置以下内容：

```json
{
  "options": {
    "disable_metrics": true  // 禁用指标收集
  }
}
```

Crush 还尊重 `DO_NOT_TRACK` 约定，可以通过 `export DO_NOT_TRACK=1` 启用。

## 贡献

请参阅 [贡献指南](https://github.com/charmbracelet/crush?tab=contributing-ov-file#contributing)。

## 你觉得怎么样？

我们很想听听你对这个项目的想法。需要帮助吗？我们来帮你。你可以在以下平台找到我们：

- [Twitter](https://twitter.com/charmcli)
- [Slack](https://charm.land/slack)
- [Discord][discord]
- [Fediverse](https://mastodon.social/@charmcli)
- [Bluesky](https://bsky.app/profile/charm.land)

[discord]: https://charm.land/discord

## 许可证

[FSL-1.1-MIT](https://github.com/charmbracelet/crush/raw/main/LICENSE.md)

---

属于 [Charm](https://charm.land)。

<a href="https://charm.land/"><img alt="The Charm logo" width="400" src="https://stuff.charm.sh/charm-banner-next.jpg" /></a>

<!--prettier-ignore-->
Charm热爱开源 • Charm loves open source
