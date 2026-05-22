# Claude-Code / Workflows / Plan-Mode

> 来源: claudecn.com

# 计划模式

计划模式（Plan Mode）让 Claude 通过只读操作分析代码库，创建详细计划后再执行更改。适合探索代码库、规划复杂变更或安全审查代码。

## 何时使用计划模式

- 多步实现：功能需要修改多个文件
- 代码探索：在更改前彻底研究代码库
- 交互式开发：与 Claude 讨论方案方向
---

## 启用计划模式

### 会话中切换

按 **Shift+Tab** 循环切换权限模式：

```
Normal Mode → Auto-Accept Mode → Plan Mode
```

- Normal Mode：默认模式，每次操作需确认
- Auto-Accept Mode：自动接受编辑（底部显示 ⏵⏵ accept edits on）
- Plan Mode：只读模式（底部显示 ⏸ plan mode on）
### 启动时指定

```bash
claude --permission-mode plan
```

### Headless 模式查询

```bash
claude --permission-mode plan -p "分析认证系统并建议改进"
```

### 设为默认

```json
// .claude/settings.json
{
  "permissions": {
    "defaultMode": "plan"
  }
}
```

---

## 典型工作流

### 规划复杂重构

```bash
claude --permission-mode plan
```

```
> 我需要将认证系统重构为使用 OAuth2。创建详细的迁移计划。
```

Claude 分析当前实现并创建全面计划。通过追问细化：

```
> 如何保持向后兼容？
> 数据库迁移应该怎么处理？
```

### 让 Claude 采访你
对于大型功能，从最小规格开始，让 Claude 提问来完善细节：

```
> 开始前先采访我关于这个功能：用户通知系统
```

```
> 帮我通过提问来思考认证需求
```

```
> 问我澄清问题来完善这个规格：支付处理
```

Claude 使用 `AskUserQuestion` 工具提出选择题，收集需求、澄清模糊点、了解偏好。这种协作方式比预先想好所有需求更有效。

在其他模式中鼓励此行为，在 `CLAUDE.md` 中添加：

```markdown
当存在多种有效方法时，始终先提出澄清问题。
```

---

## 与其他功能配合

### 计划模式 + Subagents
在计划模式下，Claude 需要研究代码库时会委托给 Plan Subagent，该代理有只读工具访问权限，在独立上下文中收集信息返回主对话。

### 计划模式 + Extended Thinking

结合扩展思考获得更深入的分析：

```bash
claude --permission-mode plan
```

```
> ultrathink: 设计 API 的缓存层
```

Claude 会进行更深入的推理和分析。

---

## 最佳实践

- 先规划后执行：对于影响多个文件的变更，先用计划模式分析
- 记录决策：计划模式生成的分析可以保存为设计文档
- 迭代细化：不断追问直到计划足够详细
- 转换执行：计划满意后，按 Shift+Tab 切换到 Normal Mode 执行
## 进一步阅读

- 先从工作原理建立直觉：/docs/claude-code/advanced/agent-loop/
- 显式 Todo（结构化规划）：/docs/claude-code/advanced/agent-loop/v2-explicit-planning-todo/
---

## 下一步
[Subagents使用专用子代理处理任务
](../../advanced/subagents/)[上下文管理提供精准上下文
](../context-management/)[高效实践专家级使用技巧
](../efficient-practices/)
