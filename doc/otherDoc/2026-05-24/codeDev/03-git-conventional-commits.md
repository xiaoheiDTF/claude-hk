# Conventional Commits 约定式提交规范

> **创建时间：** 2026-05-24  
> **所属模块：** codeDev / Git规范  
> **简述：** 详解Conventional Commits规范、格式、类型定义，以及如何在项目中落地使用（包含配置husky和commitlint）。

---

## 一、什么是 Conventional Commits？

**Conventional Commits**（约定式提交）是一种用于给提交信息增加人机可读含义的规范。它通过统一的格式，使得：

- **自动生成 CHANGELOG** — 根据类型筛选feature和fix
- **语义化版本控制** — `fix` → PATCH, `feat` → MINOR, `BREAKING CHANGE` → MAJOR
- **清晰的代码历史** — 一眼看出每次提交的目的和范围
- **自动化工具集成** — CI/CD、Release工具可基于提交类型做决策

---

## 二、提交信息格式

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

### 2.1 核心元素

| 元素 | 说明 | 必需 |
|------|------|------|
| **type** | 提交类型（feat/fix/docs等） | ✅ |
| **scope** | 影响范围（可选） | ❌ |
| **description** | 简短描述 | ✅ |
| **body** | 详细说明（可选） | ❌ |
| **footer** | 破坏性变更/关闭Issue（可选） | ❌ |

### 2.2 完整示例

```
feat(auth): add OAuth2 login support

Implement Google and GitHub OAuth2 login flow.
Add JWT token generation and refresh mechanism.

BREAKING CHANGE: login API response format changed
Closes #123
```

---

## 三、Type 类型定义

| Type | 含义 | 版本影响 |
|------|------|----------|
| `feat` | 新功能 | MINOR |
| `fix` | Bug修复 | PATCH |
| `docs` | 仅文档变更 | 无 |
| `style` | 代码格式调整（不影响功能） | 无 |
| `refactor` | 代码重构 | 无 |
| `perf` | 性能优化 | PATCH |
| `test` | 测试相关 | 无 |
| `chore` | 构建/工具/依赖变更 | 无 |
| `ci` | CI/CD配置变更 | 无 |
| `build` | 构建系统变更 | 无 |
| `revert` | 回滚提交 | 视回滚内容 |

### 特殊规则

```
# 破坏性变更（触发 MAJOR 版本升级）
feat(api)!: remove deprecated endpoints
# 或
feat(api): remove deprecated endpoints

BREAKING CHANGE: v1 API endpoints removed

# 关闭Issue
fix(auth): resolve login timeout issue

Closes #456
Refs #789
```

---

## 四、编写规范

### ✅ 正确的提交信息

```
feat(user): add password reset functionality

fix(api): handle nil pointer in response handler

docs(readme): update installation instructions

refactor(service): extract validation logic

test(repository): add missing error path tests
```

### ❌ 错误的提交信息

```
update code              # 缺少type
fix bug                  # 描述太模糊
feat:                    # 缺少描述
feat: Add new Feature    # description首字母大写（应为小写）
feat:user auth           # scope前后缺少空格
```

### 规则总结

1. **type后必须有空格**，然后是可选的scope
2. **scope前后用括号包围**，后面紧跟冒号
3. **冒号后必须有空格**
4. **description首字母小写**，不用句号结尾
5. **使用祈使语气**："add" 而不是 "added" 或 "adds"
6. **描述在50字符以内**（标题行）

---

## 五、在Go项目中落地

### 5.1 安装 commitlint

```bash
# 需要Node.js环境
npm install --save-dev @commitlint/config-conventional @commitlint/cli
```

### 5.2 配置 commitlint

创建 `commitlint.config.js`：

```javascript
module.exports = {
  extends: ['@commitlint/config-conventional'],
  rules: {
    'type-enum': [2, 'always', [
      'feat', 'fix', 'docs', 'style', 'refactor',
      'perf', 'test', 'chore', 'ci', 'build', 'revert'
    ]],
    'scope-case': [0],
    'subject-case': [0],
    'header-max-length': [2, 'always', 72]
  }
};
```

### 5.3 配置 Husky（提交前校验）

```bash
npm install --save-dev husky
npx husky install
```

创建 `.husky/commit-msg`：

```bash
#!/bin/sh
. "$(dirname "$0")/_/husky.sh"

npx --no -- commitlint --edit ${1}
```

### 5.4 Go项目替代方案：git-hooks

如果不使用Node.js，可用纯Go方案：

```bash
# 安装 lefthook（Go编写，无需Node）
go install github.com/evilmartians/lefthook@latest
```

创建 `lefthook.yml`：

```yaml
commit-msg:
  commands:
    lint-commit-msg:
      run: |
        echo "{extends: ['@commitlint/config-conventional']}" > /tmp/commitlint.config.js
        npx commitlint --config /tmp/commitlint.config.js --edit {1}
```

### 5.5 Go项目极简方案：自定义hook

`.git/hooks/commit-msg`：

```bash
#!/bin/bash
# 简易Conventional Commits校验

commit_msg_file=$1
commit_msg=$(head -n1 "$commit_msg_file")

pattern="^(feat|fix|docs|style|refactor|perf|test|chore|ci|build|revert)(\(.+\))?!?: .+$"

if ! echo "$commit_msg" | grep -qE "$pattern"; then
    echo "ERROR: Commit message does not follow Conventional Commits!"
    echo "Expected format: <type>[scope]: <description>"
    echo "Allowed types: feat, fix, docs, style, refactor, perf, test, chore, ci, build, revert"
    exit 1
fi
```

---

## 六、自动生成 CHANGELOG

### 使用 git-cliff

```bash
cargo install git-cliff  # 或使用预编译二进制
```

配置 `cliff.toml`：

```toml
[changelog]
header = "# Changelog\n\n"
body = """
{% if version %}\n## [{{ version | trim_start_matches(pat="v") }}] - {{ timestamp | date(format="%Y-%m-%d") }}\n{% else %}\n## [unreleased]\n{% endif %}\n{% for group, commits in commits | group_by(attribute="group") %}\n### {{ group | upper_first }}\n{% for commit in commits %}\n- {{ commit.message | upper_first }}\n{% endfor %}\n{% endfor %}\n"""

[git]
conventional_commits = true
filter_unconventional = true
```

生成：
```bash
git-cliff --output CHANGELOG.md
```

---

## 七、语义化版本自动发布

结合 Conventional Commits 自动计算版本号：

```bash
# 使用 semantic-release 或 standard-version
# 或使用 Go 工具：
go install github.com/caarlos0/svu@latest

# 查看下一个版本
svu next

# 创建带版本号的tag
svu tag
```

---

## 八、PR标题规范

PR标题也应遵循Conventional Commits格式：

```
feat(auth): implement OAuth2 login
^--^ ^---^  ^--------------------^
|     |       |
|     |       +-> PR描述
|     +-> 影响范围
+-> 类型
```

GitHub PR模板：

```markdown
## 类型
- [ ] feat: 新功能
- [ ] fix: Bug修复
- [ ] docs: 文档更新
- [ ] test: 测试相关
- [ ] refactor: 代码重构
- [ ] perf: 性能优化

## 描述
<!-- 描述本次变更的内容 -->

## 关联Issue
Closes #xxx

## 测试
- [ ] 单元测试通过
- [ ] 集成测试通过
- [ ] 覆盖率 ≥95%
```

---

## 九、Checklist

- [ ] 提交前使用 `git log --oneline` 检查历史格式
- [ ] 每个提交只做一件事，保持原子性
- [ ] 配置 commit-msg hook 强制规范
- [ ] CI中校验PR标题格式
- [ ] 配置自动CHANGELOG生成
- [ ] 团队共享提交规范文档
