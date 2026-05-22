# Claude-Code / Getting-Started / Installation

> 来源: claudecn.com

# 安装配置

本文介绍如何在不同系统上安装 Claude Code，以及完成首次认证。

如果你需要配置第三方 Provider（Anthropic 兼容接口）、或想做“多 Provider/多模型一键切换”，可以参考这篇实战文章：
[/blog/claude-code-install-and-provider-switching/](https://claudecn.com/blog/claude-code-install-and-provider-switching/)

## 系统要求
在安装之前，请确保你的系统满足以下要求：

| 要求 | 说明 |
| --- | --- |
| **[ 操作系统](#)** | macOS 10.15+、Ubuntu 20.04+/Debian 10+、Windows 10+ (通过 WSL2) |
| **硬件** | 4GB+ RAM |
| **[ 软件](#)** | Node.js 18+ |
| **网络** | 需要互联网连接 |

Windows 用户需要通过 WSL2（Windows Subsystem for Linux）运行 Claude Code。

## 安装方式
选择适合你系统的安装方式：

### NPM 安装（推荐）

如果你已经安装了 Node.js，这是最简单的方式：

```bash
npm install -g @anthropic-ai/claude-code
```

### Homebrew 安装
适用于 macOS 和 Linux：

```bash
brew install --cask claude-code
```

### 脚本安装
**macOS、Linux、WSL：**

```bash
curl -fsSL https://claude.ai/install.sh | bash
```

**Windows PowerShell：**

```powershell
irm https://claude.ai/install.ps1 | iex
```

**WinGet (Windows)：**

```powershell
winget install Anthropic.ClaudeCode
```

## 验证安装
安装完成后，运行以下命令确认安装成功：

```bash
claude --version
```

如果看到版本号输出，说明安装成功。

## VS Code 扩展

如果你使用 VS Code，可以安装官方扩展获得集成体验：

- 打开 VS Code
- 按 Cmd+Shift+X（macOS）或 Ctrl+Shift+X（Windows/Linux）打开扩展面板
- 搜索 “Claude Code”
- 点击安装
或者在命令行安装：

```bash
code --install-extension anthropic.claude-code
```

## 首次启动和认证
安装完成后，在终端运行：

```bash
claude
```

首次启动时，Claude Code 会引导你完成认证：

- 选择认证方式（Anthropic API 或 Claude Pro/Max 订阅）
- 根据提示在浏览器中完成登录
- 返回终端，认证自动完成
### 认证方式对比

| 方式 | 适合 | 计费方式 |
| --- | --- | --- |
| **Claude Pro/Max 订阅** | 个人用户 | 订阅制，包含一定额度 |
| **Anthropic API** | 开发者、企业用户 | 按使用量计费 |

## 常见问题排查

### Node.js 版本过低

```bash
# 检查 Node.js 版本
node --version

# 如果低于 18，请升级
# 使用 nvm 管理版本
nvm install 18
nvm use 18
```

### 权限问题
如果 NPM 全局安装遇到权限问题：

```bash
# 方法 1：使用 sudo（不推荐）
sudo npm install -g @anthropic-ai/claude-code

# 方法 2：修复 npm 权限（推荐）
mkdir ~/.npm-global
npm config set prefix '~/.npm-global'
echo 'export PATH=~/.npm-global/bin:$PATH' >> ~/.bashrc
source ~/.bashrc
```

### 网络问题
如果下载缓慢或失败，可以尝试使用你所在网络环境可用的 npm 镜像源（例如公司内部 registry）：

```bash
npm config set registry https://registry.example.com
npm install -g @anthropic-ai/claude-code
```

### WSL2 问题
Windows 用户如果 WSL2 未正确配置：

- 确保 WSL2 已启用：wsl --install
- 确保使用 WSL2 而非 WSL1：wsl --set-default-version 2
- 在 WSL 终端中运行安装命令
## 更新 Claude Code

保持 Claude Code 更新以获得最新功能和修复：

```bash
# NPM 安装的更新方式
npm update -g @anthropic-ai/claude-code

# Homebrew 安装的更新方式
brew upgrade --cask claude-code
```

## 下一步
安装完成后，让我们进行 [第一次对话](../first-conversation/)。
