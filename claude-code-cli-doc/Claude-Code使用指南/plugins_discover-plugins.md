# Claude-Code / Plugins / Discover-Plugins

> 来源: claudecn.com

# 发现和安装插件

安装插件之前，先想清楚“我要解决什么问题”。对大多数人来说，真正需要做的判断只有三个：从哪里装、装完怎么验、什么时候该换来源。

## 三种最常见的安装来源

### 1. 从官方目录开始

如果你要的是常见开发流程、常见集成或常见增强能力，优先从官方目录开始最省心：

```bash
/plugin marketplace list
/plugin install code-simplifier@claude-plugins-official
```

这种方式更适合“先装一个就能立刻试用”的场景。

### 2. 从团队或私有来源安装

如果你的能力明显带有组织特征，例如内部命令、内部审查流程、私有 MCP 服务或团队规范，团队插件通常更合适：

```bash
/plugin marketplace add example-team/plugins
/plugin install plugin-name@example-team-plugins
```

### 3. 直接从本地或源码安装
如果你正在做实验、调试或开发自己的插件，可以直接装本地目录或源码来源：

```bash
/plugin install ./my-plugin
```

或者：

```bash
/plugin install owner/repo
```

## 怎么判断该从哪种来源开始

- 想快速试一个成熟能力：先从官方目录开始
- 想沉淀团队规则：优先团队或私有来源
- 想边做边调：直接从本地或源码安装
## 安装后别只看“装成功”
先列出已安装插件：

```bash
/plugin list
```

然后做一个小任务验证：

- 你需要的命令能否直接调用
- 插件依赖的工具是否已经准备完成
- 真实任务里是否比原来更顺手
如果装上以后只是“多了一层概念”，却没有让任务更快更稳，那它就还不值得留在日常工作流里。

## 常用管理命令

### 更新目录索引

```bash
/plugin marketplace update claude-plugins-official
```

### 升级插件

```bash
/plugin upgrade <plugin-name>
```

### 卸载插件

```bash
/plugin uninstall <plugin-name>
```

## 常见问题

### /plugin 命令不可用
先检查 Claude Code 版本是否满足插件支持要求，再更新到较新的可用版本。

### 安装成功，但能力没有出现

优先检查三件事：

- 插件是否真的启用
- 依赖的本地工具或服务是否存在
- 当前任务是否触发了对应能力
### 装了很多插件后感觉变重

这通常说明插件太多、边界不清。回到真实任务，保留那些能稳定提高效率的插件，其余先移除。

## 下一步

- 想先理解插件本身：看 先理解插件
- 想判断什么时候该从官方目录开始：看 选择插件来源
- 想开始封装自己的插件：看 创建插件
