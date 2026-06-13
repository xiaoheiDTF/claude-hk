---
name: 999-2-git-push
description: 按规范格式提交代码并推送到远程仓库（commit + push）
user-invocable: true
allowed-tools:
  - Bash
  - Read
  - Glob
  - Grep
---

# Git Push Skill

按照统一规范格式提交代码 **并推送到远程仓库**。

## 提交格式

```
<type>: <主描述>

- 具体修改描述1
- 具体修改描述2
```

### type 类型

- `fix`: 修复 bug
- `feat`: 新功能
- `update`: 更新
- `style`: 代码格式改变（不影响功能）
- `refactor`: 重构（不新增功能也不修复 bug）
- `perf`: 性能优化
- `test`: 增加测试代码
- `docs`: 文档变更
- `revert`: 撤销上一次 commit
- `build`: 构建工具或构建过程变动
- `chore`: 其他杂项变动

### description 规则

1. **主描述**：简述本次主要修改内容，不超过 50 个字符，使用中文
2. **子描述**：每条以 `- ` 开头，描述一个具体的修改点
3. 所有文字必须使用中文
4. 主描述 + 子描述必须完整覆盖本次 add 的所有变更

### 完整示例

```bash
git commit -m "feat: 新增用户注册功能

- 添加注册表单页面及表单验证逻辑
- 实现邮箱确认发送与校验接口"

git commit -m "fix: 修复订单金额计算错误

- 修正满减优惠叠加时的重复计算问题
- 处理金额为零时的边界条件"

git commit -m "update: 优化首页加载性能

- 图片组件改为懒加载
- 首屏接口数据增加本地缓存"
```

## 操作流程

1. **查看变更**：执行 `git status` 和 `git diff`（未暂存）+ `git diff --cached`（已暂存），仔细阅读每个文件的改动内容
2. **分析并分类**：根据 diff 结果，将变更按逻辑分组（见下方分类规则）
3. **分批暂存**：每次 `git add` 一个分组，然后执行一次 `git commit`
4. **推送到远程**：所有分组提交完成后，执行 `git push`

## git add 分类规则

**必须先 `git diff` 查看所有变更，再决定如何分组 add。**

### 分类优先级（从高到低）

1. **按 type 归类**：不同 type 的变更必须分开提交
   - 修复 bug 的文件 → 单独一个 `fix` 提交
   - 新功能的文件 → 单独一个 `feat` 提交
   - 格式调整的文件 → 单独一个 `style` 提交

2. **按目录/模块归类**：同一 type 下，按所属目录或模块分组
   - 同一个页面/组件相关的文件 add 在一起
   - 同一个 API 模块的文件 add 在一起

3. **按功能关联归类**：同一目录下按功能相关性细分
   - 某个功能的逻辑 + 对应的测试 → add 在一起
   - 某个页面的样式 + 模板 + 逻辑 → add 在一起
   - 配置文件变更 + 相关构建脚本 → add 在一起

4. **按影响范围归类**：跨目录但功能关联的变更
   - 后端接口 + 前端调用方 → add 在一起
   - 数据库迁移 + 对应的模型代码 → add 在一起

### 分类示例

```
变更文件：
  src/api/user.js          → 接口新增
  src/api/order.js         → bug 修复
  src/pages/Login.vue      → 登录页新功能
  src/pages/Login.vue      → 登录页样式调整
  src/utils/format.js      → 工具函数优化
  tests/api/user.test.js   → 用户接口测试
  .eslintrc.js             → lint 配置调整

分组方案：
  第1批 (fix):   git add src/api/order.js
                 → fix: 修复订单接口参数校验问题
                 → - 修正金额字段缺失时的空指针异常

  第2批 (feat):  git add src/api/user.js src/pages/Login.vue tests/api/user.test.js
                 → feat: 新增用户登录功能
                 → - 添加登录页面及表单交互逻辑
                 → - 新增用户登录接口及单元测试

  第3批 (update): git add src/utils/format.js
                  → update: 优化日期格式化工具函数
                  → - 重构日期解析逻辑提升可读性

  第4批 (style):  git add .eslintrc.js
                  → chore: 调整 ESLint 缩进规则配置
                  → - 统一缩进为2空格
```

## 规则

1. 提交信息必须使用中文
2. **禁止** `git add .` 或 `git add -A`，必须按分组 add 具体文件
3. 每次 add 前必须先 `git diff` 查看变更内容
4. 主描述不超过 50 个字符，子描述每条独立一行以 `- ` 开头
5. 不要提交敏感文件（.env、密钥等）
6. 推送前确认远程分支状态，有冲突先解决
