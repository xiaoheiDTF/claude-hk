# Claude-Code / Workflows / Security-Review

> 来源: claudecn.com

# 安全审查工作流：从 Secrets 到 OWASP

安全审查的目标不是“写一篇安全科普”，而是把风险变成**可执行的检查清单 + 可阻断的门禁**。下文整理为团队可落地流程。

## 什么时候必须做安全审查

满足任意一条就应该做：

- 新增/修改 API 端点
- 处理用户输入（查询参数、表单、文件上传）
- 认证/鉴权相关变更
- 数据库查询/写入逻辑调整
- 引入/升级依赖（可能带 CVE）
- 资金/支付/交易等敏感流程
## 一条推荐的安全审查闭环（可直接照做）

- 先跑自动化检查：依赖漏洞、Secrets 扫描、简单规则扫描
- 再做人工审查：按 OWASP Top 10 的视角过一遍高风险点
- 输出分级报告：CRITICAL/HIGH 必须阻断合并
- 补测试与验证：至少覆盖“错误路径与边界条件”
## 自动化检查（示例节选）

来自 `security-reviewer` 的建议命令（节选）：

```bash
# High severity only
npm audit --audit-level=high
```

以及针对 Secrets 的简单 grep（节选）：

```bash
grep -r "api[_-]?key\\|password\\|secret\\|token" --include="*.js" --include="*.ts" --include="*.json" .
```

提醒：这些检查会有误报/漏报，不能替代人工审查；同时不要把真实密钥写进 repo（见 [安全指南](https://claudecn.com/docs/claude-code/reference/security/)）。

## 人工审查重点：三类“最常见且最致命”的问题

### 1) Hardcoded Secrets（CRITICAL）

`security-reviewer` 示例（节选）：

```javascript
// ❌ CRITICAL: Hardcoded secrets
const apiKey = "sk-proj-xxxxx"
const password = "admin123"
const token = "ghp_xxxxxxxxxxxx"
```

### 2) SQL Injection（CRITICAL）
`security-reviewer` 示例（节选）：

```javascript
// ❌ CRITICAL: SQL injection vulnerability
const query = `SELECT * FROM users WHERE id = ${userId}`
await db.query(query)
```

### 3) SSRF（HIGH）
`security-reviewer` 示例（节选）：

```javascript
// ✅ CORRECT: Validate and whitelist URLs
const allowedDomains = ['api.example.com', 'cdn.example.com']
const url = new URL(userProvidedUrl)
if (!allowedDomains.includes(url.hostname)) {
  throw new Error('Invalid URL')
}
const response = await fetch(url.toString())
```

## 输出格式：按严重度分组的“可执行报告”
`security-reviewer` 给出了一份报告模板。建议你至少固定这三项：

- Summary：Critical/High/Medium/Low 数量 + 风险等级
- Blocking：CRITICAL/HIGH（必须修复）逐条列定位与修复建议
- Checklist：一眼看完是否过线
如果你想把“安全审查”嵌入团队闭环，建议和 [团队质量门禁：Plan → TDD → Build Fix → Review](https://claudecn.com/docs/claude-code/workflows/quality-gates/) 一起使用：有 CRITICAL/HIGH 就不合并。

## 参考

无
