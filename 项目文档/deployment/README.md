# Crush-CN 部署文档

## 概述

本文档详细说明 Crush-CN 的环境配置、构建流程、安装部署和配置说明。Crush-CN 是一个强大的终端 AI 助手，支持多种操作系统和架构。

## 目录

1. [环境配置](01-环境配置.md)
2. [构建流程](02-构建流程.md)
3. [安装部署](03-安装部署.md)
4. [配置说明](04-配置说明.md)

## 快速开始

### 系统要求

| 组件 | 最低要求 | 推荐配置 |
|------|----------|----------|
| Go | 1.25.5 | 最新稳定版 |
| 内存 | 512MB | 1GB+ |
| 磁盘 | 100MB | 500MB+ |

### 支持的平台

| 操作系统 | 架构 | 支持状态 |
|----------|------|----------|
| Linux | amd64, arm64, 386, arm | 完全支持 |
| macOS | amd64, arm64 | 完全支持 |
| Windows | amd64, arm64, 386 | 完全支持 |
| FreeBSD | amd64, arm64, 386, arm | 完全支持 |
| OpenBSD | amd64, arm64, 386, arm | 完全支持 |
| NetBSD | amd64, arm64, 386, arm | 完全支持 |
| Android | arm64 | 完全支持 |

### 快速安装

```bash
# macOS/Linux - Homebrew
brew install charmbracelet/tap/crush

# Windows - Winget
winget install charmbracelet.crush

# Windows - Scoop
scoop bucket add charm https://github.com/charmbracelet/scoop-bucket.git
scoop install crush

# NPM
npm install -g @charmland/crush

# Arch Linux
yay -S crush-bin

# FreeBSD
pkg install crush
```

## 文档说明

### 环境配置

详细说明开发环境的搭建，包括：
- Go 语言环境安装
- 开发工具配置
- 依赖管理
- 环境变量设置

### 构建流程

详细说明项目的构建过程，包括：
- Taskfile 任务
- 构建命令
- 测试运行
- 代码检查

### 安装部署

详细说明各种安装方式，包括：
- 包管理器安装
- 二进制安装
- 源码编译安装
- Docker 部署

### 配置说明

详细说明应用配置，包括：
- 配置文件位置
- 配置项说明
- LLM 提供商配置
- MCP 服务器配置

## 相关链接

- [项目主页](https://charm.sh/crush)
- [GitHub 仓库](https://github.com/purpose168/crush-cn)
- [问题反馈](https://github.com/purpose168/crush-cn/issues)
- [发布页面](https://github.com/purpose168/crush-cn/releases)
