# Agent-Skills / Quickstart

> 来源: claudecn.com

# Agent Skills 快速开始

通过本教程学习如何使用 Agent Skills 在 10 分钟内通过 Claude API 创建文档。

本教程将展示如何使用 Agent Skills 创建 PowerPoint 演示文稿。你将学习如何启用 Skills、发起简单请求并访问生成的文件。

## 前提条件

- Anthropic API 密钥
- Python 3.7+ 或已安装 curl
- 基本的 API 请求知识
## 什么是 Agent Skills？

预构建的 Agent Skills 为 Claude 提供专门的能力，用于创建文档、分析数据和处理文件等任务。Anthropic 在 API 中提供以下预构建 Agent Skills：

- PowerPoint (pptx)：创建和编辑演示文稿
- Excel (xlsx)：创建和分析电子表格
- Word (docx)：创建和编辑文档
- PDF (pdf)：生成 PDF 文档
**想要创建自定义 Skills？** 继续阅读本站的 [最佳实践](../best-practices/) 和 [示例](../examples/) 页面，先建立稳定的结构和写法，再参考官方的 [Agent Skills Cookbook](https://github.com/anthropics/claude-cookbooks/tree/main/skills)。

## 步骤 1: 列出可用的 Skills

首先，让我们看看有哪些可用的 Skills。我们将使用 Skills API 列出所有 Anthropic 管理的 Skills：

```python
import anthropic

client = anthropic.Anthropic()

# 列出 Anthropic 管理的 Skills
skills = client.beta.skills.list(
    source="anthropic",
    betas=["skills-2025-10-02"]
)

for skill in skills.data:
    print(f"{skill.id}: {skill.display_title}")
```

你将看到以下 Skills：`pptx`、`xlsx`、`docx` 和 `pdf`。此 API 返回每个 Skill 的元数据：名称和描述。Claude 在启动时加载这些元数据以了解有哪些 Skills 可用。这是**渐进式披露**的第一级，Claude 无需加载完整指令即可发现 Skills。

## 步骤 2: 创建演示文稿

现在我们将使用 PowerPoint Skill 创建一个关于可再生能源的演示文稿。我们使用 Messages API 中的 `container` 参数指定 Skills：

```python
import anthropic

client = anthropic.Anthropic()

# 使用 PowerPoint Skill 创建消息
response = client.beta.messages.create(
    model="claude-sonnet-4-5-20250929",
    max_tokens=4096,
    betas=["code-execution-2025-08-25", "skills-2025-10-02"],
    container={
        "skills": [
            {
                "type": "anthropic",
                "skill_id": "pptx",
                "version": "latest"
            }
        ]
    },
    messages=[{
        "role": "user",
        "content": "创建一个关于可再生能源的演示文稿，包含 5 张幻灯片"
    }],
    tools=[{
        "type": "code_execution_20250825",
        "name": "code_execution"
    }]
)

print(response.content)
```

让我们分解一下每个部分的作用：

- container.skills：指定 Claude 可以使用哪些 Skills
- type: "anthropic"：表示这是 Anthropic 管理的 Skill
- skill_id: "pptx"：PowerPoint Skill 标识符
- version: "latest"：Skill 版本设置为最新发布的版本
- tools：启用代码执行（Skills 必需）
- Beta 头部：code-execution-2025-08-25 和 skills-2025-10-02
当你发起此请求时，Claude 会自动将你的任务匹配到相关的 Skill。由于你请求创建演示文稿，Claude 判断 PowerPoint Skill 相关并加载其完整指令：这是渐进式披露的第二级。然后 Claude 执行 Skill 的代码来创建你的演示文稿。

## 步骤 3: 下载创建的文件

演示文稿在代码执行容器中创建并保存为文件。响应包含一个带有文件 ID 的文件引用。提取文件 ID 并使用 Files API 下载：

```python
# 从响应中提取文件 ID
file_id = None
for block in response.content:
    if block.type == 'tool_use' and block.name == 'code_execution':
        # 文件 ID 在工具结果中
        for result_block in block.content:
            if hasattr(result_block, 'file_id'):
                file_id = result_block.file_id
                break

if file_id:
    # 下载文件
    file_content = client.beta.files.download(
        file_id=file_id,
        betas=["files-api-2025-04-14"]
    )
    
    # 保存到磁盘
    with open("renewable_energy.pptx", "wb") as f:
        file_content.write_to_file(f.name)
    
    print(f"演示文稿已保存到 renewable_energy.pptx")
```

有关处理生成文件的完整详细信息，请参阅[代码执行工具文档](https://docs.claude.com/en/docs/agents-and-tools/tool-use/code-execution-tool#retrieve-generated-files)。

## 尝试更多示例

现在你已经使用 Skills 创建了第一个文档，试试这些变体：

### 创建电子表格

```python
response = client.beta.messages.create(
    model="claude-sonnet-4-5-20250929",
    max_tokens=4096,
    betas=["code-execution-2025-08-25", "skills-2025-10-02"],
    container={
        "skills": [
            {
                "type": "anthropic",
                "skill_id": "xlsx",
                "version": "latest"
            }
        ]
    },
    messages=[{
        "role": "user",
        "content": "创建一个季度销售跟踪电子表格，包含示例数据"
    }],
    tools=[{
        "type": "code_execution_20250825",
        "name": "code_execution"
    }]
)
```

### 创建 Word 文档

```python
response = client.beta.messages.create(
    model="claude-sonnet-4-5-20250929",
    max_tokens=4096,
    betas=["code-execution-2025-08-25", "skills-2025-10-02"],
    container={
        "skills": [
            {
                "type": "anthropic",
                "skill_id": "docx",
                "version": "latest"
            }
        ]
    },
    messages=[{
        "role": "user",
        "content": "写一份关于可再生能源优势的 2 页报告"
    }],
    tools=[{
        "type": "code_execution_20250825",
        "name": "code_execution"
    }]
)
```

### 生成 PDF

```python
response = client.beta.messages.create(
    model="claude-sonnet-4-5-20250929",
    max_tokens=4096,
    betas=["code-execution-2025-08-25", "skills-2025-10-02"],
    container={
        "skills": [
            {
                "type": "anthropic",
                "skill_id": "pdf",
                "version": "latest"
            }
        ]
    },
    messages=[{
        "role": "user",
        "content": "生成一个 PDF 发票模板"
    }],
    tools=[{
        "type": "code_execution_20250825",
        "name": "code_execution"
    }]
)
```

## 下一步
现在你已经使用了预构建的 Agent Skills，可以：

- API 指南 - 通过 Claude API 使用 Skills
- 创建自定义 Skills - 上传你自己的专门任务 Skills
- 创作指南 - 学习编写有效 Skills 的最佳实践
- 在 Claude Code 中使用 Skills - 了解 Claude Code 中的 Skills
- Skills 示例 - 查看站内整理后的技能结构和示例
- Agent Skills Cookbook - 查看官方示例 Skills 和实现模式
