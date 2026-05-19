# Claude-Code / Workflows / Code-Review

> 来源: claudecn.com

# 代码审查工作流：分级输出与合并门禁

代码审查的目标不是“挑刺”，而是让团队在合并前把风险变成显式结论：哪些必须修、哪些建议修、哪些可以后补。一个高复用的做法是：**覆盖未提交改动**、**按严重度分级并阻断高风险**。

## 1) 审查范围：针对“未提交改动”

社区命令把入口固定为：

```text
Get changed files: git diff --name-only HEAD
```

这能避免“只看最新提交”遗漏累计改动。

## 2) 审查维度：安全优先，其次质量与可维护性

`/code-review` 的检查点（节选）包括：

- Security（CRITICAL）：硬编码密钥、注入风险（SQL/XSS）、依赖漏洞、路径遍历等
- Code Quality（HIGH）：函数过大、文件过大、深层嵌套、错误处理缺失、console.log、TODO/FIXME
- Best Practices（MEDIUM）：可变数据、可访问性、测试缺失等
## 3) 输出格式：定位 + 问题 + 建议修复

建议你在团队里固定输出结构：

- 按文件分组
- 每条问题包含：严重度、定位信息、问题描述、修复建议
- 结论必须明确：是否阻断合并
## 4) 合并门禁：CRITICAL/HIGH 直接阻断

社区命令的规则非常直白：

```text
Block commit if CRITICAL or HIGH issues found
Never approve code with security vulnerabilities!
```

落地建议：把这条写进团队流程（例如 PR 模板或 `CLAUDE.md`），并配合 [团队质量门禁：Plan → TDD → Build Fix → Review](https://claudecn.com/docs/claude-code/workflows/quality-gates/) 一起执行。

## 5) 常见误区

- 审查只关注风格，不关注输入校验/鉴权边界
- 审查给结论但不给修复路径（导致反复沟通）
- 改动太大导致审查无法有效进行（建议先拆小 PR）
## 参考

无
