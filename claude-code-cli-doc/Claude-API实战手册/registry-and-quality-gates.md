# Cookbook / Registry-And-Quality-Gates

> 来源: claudecn.com

# Registry 与质量门禁

主要内容分两部分：

- registry.yaml 是什么、为什么它比“README 里的列表”更重要；
- 贡献一个新的 cookbook/notebook，需要过哪些“质量门禁”（本地 + CI）。
## 1) registry.yaml：可机读的 Cookbook 索引

仓库用 `registry.yaml` 维护 notebook 元数据，并通过 `.github/registry_schema.json` 做 schema 校验。

### 1.1 必填字段（最小集）

每条记录至少包含：

- title
- path
- authors（GitHub 用户名，且必须在 authors.yaml 定义）
- date（YYYY-MM-DD）
- categories（至少 1 个，来自枚举）
### 1.2 分类枚举（categories）

当前 schema 允许的分类包括：

- Agent Patterns
- Claude Agent SDK
- Evals
- Fine-Tuning
- Multimodal
- Integrations
- Observability
- RAG & Retrieval
- Responses
- Skills
- Thinking
- Tools
### 1.3 一个最小 entry 示例

```yaml
- title: Your Cookbook Title
  description: What users will build/learn in 1-2 sentences.
  path: tool_use/your_notebook.ipynb
  authors:
  - your-github-handle
  date: '2026-01-23'
  categories:
  - Tools
```

## 2) authors.yaml：作者信息的“单一真相”
`authors.yaml` 维护作者的展示名、主页、头像等信息。仓库提供了排序与校验脚本，避免长期积累成“人工维护地狱”。

```bash
make sort-authors
```

## 3) 质量门禁：你在本地就应该跑过

### 3.1 代码风格与基础测试

```bash
make check
make test
```

### 3.2 Notebook 结构测试

```bash
make test-notebooks NOTEBOOK=path/to/notebook.ipynb
```

结构测试会重点关注：

- 是否从 干净内核 运行（执行计数从 1 开始）
- 是否按顺序执行（避免“跳着跑导致别人复现不了”）
- 是否存在 error output
- 是否硬编码密钥/Token
- pip install 是否过晚（建议放在前面几格）
### 3.3 Registry/Authors 的一致性校验

仓库提供 `.github/scripts/verify_registry.py` 用于检查：

- registry 中引用的 authors 是否在 authors.yaml 存在
- registry 中的 path 是否真的存在
- YAML 是否匹配 JSON schema
其中作者 URL/GitHub handle 检查需要网络；离线场景可先跑 `schema/paths/registry` 子命令。

## 4) 用 Claude Code 提升“贡献/审核”的效率

Cookbooks 仓库的 `.claude/` 目录提供了一组可直接用在 Claude Code 的 **slash commands**：

- /add-registry：根据 notebook 内容生成并补齐 registry 条目
- /notebook-review：按 rubric 做 notebook 质量 review
- /model-check：检查是否引用了过时/不公开的模型名
- /link-review：检查变更文件中的链接质量
- /review-pr / /review-pr-ci：PR review（本地/CI 场景）
此外，`.claude/skills/cookbook-audit/style_guide.md` 给出了 cookbook 写作的“教学合同”模板（问题导向、学习目标、结论回扣、dotenv、MODEL 常量、%%capture 处理 pip 输出等）。
