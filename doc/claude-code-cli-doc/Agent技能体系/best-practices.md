# Agent-Skills / Best-Practices

> 来源: claudecn.com

# Agent Skills 最佳实践

本指南提供创建高质量 Agent Skills 的最佳实践，帮助你编写清晰、高效、易维护的 Skills。

如果你已经不只是想“把单个 Skill 写好”，而是想把一组 Skills 组织成团队真正会使用的入口，建议配合阅读：[团队索引页模板](team-index-template/)。

## 核心原则

### 1. 清晰的描述

`description` 字段是 Claude 决定何时使用 Skill 的关键。一个好的描述应该：

**✅ 好的描述**：

```yaml
---
name: PDF Processing
description: Extract text and tables from PDF files, fill forms, merge documents. Use when working with PDF files or when the user mentions PDFs, forms, or document extraction.
---
```

**❌ 不好的描述**：

```yaml
---
name: PDF Processing
description: Handles PDF files.
---
```

### 描述编写技巧

- 说明功能：清楚地描述 Skill 能做什么
- 触发条件：明确何时应该使用这个 Skill
- 关键词：包含用户可能使用的术语
- 简洁明了：保持在 1024 字符以内**模板**：

```
[功能描述]. Use when [触发场景] or when the user mentions [关键词].
```

## 结构化指令

### 使用清晰的层级结构
**✅ 推荐做法**：

```markdown
# Skill Name

## 快速开始

### 基本用法
步骤 1：准备工作
步骤 2：执行任务

### 高级用法
步骤 1：配置选项
步骤 2：优化性能

## 最佳实践

### 性能优化
- 建议 1
- 建议 2

### 错误处理
- 常见错误及解决方案
```

### 提供具体示例
**✅ 包含可运行的代码**：

```markdown
## 示例：提取 PDF 文本

```python
import pdfplumber

with pdfplumber.open("document.pdf") as pdf:
    for page in pdf.pages:
        text = page.extract_text()
        print(text)
```

**输出**：
```
Page 1 content here...
```
```

**❌ 避免抽象描述**：

```markdown
## 示例
使用相关库读取 PDF 文件并提取内容。
```

## 渐进式披露

### 主指令简洁明了
`SKILL.md` 的主体应该包含：

- 常用功能的快速开始
- 基本工作流程
- 常见用例
```markdown
# PDF Processing

## 快速开始

提取 PDF 文本：
```python
import pdfplumber
with pdfplumber.open("doc.pdf") as pdf:
    text = pdf.pages[0].extract_text()
```

## 高级功能

需要表单填写？查看 [FORMS.md](FORMS.md)
需要 PDF 合并？查看 [MERGE.md](MERGE.md)
```

### 复杂内容单独文件
将高级功能、详细 API 参考等放在单独文件：

```
pdf-skill/
├── SKILL.md          # 主指令（< 5k tokens）
├── FORMS.md          # 表单处理详细指南
├── MERGE.md          # PDF 合并指南
├── REFERENCE.md      # 完整 API 参考
└── scripts/
    └── utilities.py  # 实用脚本
```

## 代码与脚本

### 何时使用脚本
**使用脚本的场景**：

- 确定性操作（文件格式转换、数据验证）
- 复杂算法（加密、压缩）
- 需要特定库的操作
**使用指令的场景**：

- 需要灵活判断的任务
- 上下文相关的决策
- 创意性工作
### 脚本示例

```markdown
## 数据验证

使用验证脚本确保数据格式正确：

```bash
python scripts/validate.py data.json
```

脚本会检查：
- JSON 格式有效性
- 必需字段存在
- 数据类型正确
```

## 错误处理

### 明确说明限制
**✅ 清楚说明边界**：

```markdown
## 限制

本 Skill 有以下限制：
- 不支持网络请求
- 仅支持预安装的 Python 包
- 最大文件大小：10MB
- 支持的格式：PDF, DOCX, TXT
```

### 提供解决方案

```markdown
## 常见问题

### 问题：文件过大导致处理失败

**症状**：Error: File size exceeds limit

**解决方案**：
```python
# 分批处理大文件
def process_large_file(filepath, chunk_size=1000):
    with open(filepath, 'r') as f:
        while True:
            chunk = f.read(chunk_size)
            if not chunk:
                break
            process_chunk(chunk)
```
```

## 命名规范

### Skill 命名

- 使用描述性名称
- 避免通用词汇
- 反映核心功能**✅ 好的命名**：

- Database Query Optimizer
- API Error Handler
- Document Format Converter
**❌ 不好的命名**：

- Helper
- Utility
- Tool
### 文件命名

- 使用清晰的文件名
- 用大写表示重要性
- 保持一致性
```
skill-directory/
├── SKILL.md          # 主文件（大写）
├── README.md         # 说明文件
├── ADVANCED.md       # 高级功能
├── REFERENCE.md      # API 参考
└── scripts/          # 小写目录
    └── helper.py
```

## 内容组织

### 从简单到复杂

```markdown
# Skill Name

## 最简单的用例
一行代码完成基本任务：
```python
result = simple_function(input)
```

## 常见场景
添加错误处理和选项：
```python
try:
    result = function(input, option=True)
except Exception as e:
    handle_error(e)
```

## 高级用法
详见 [ADVANCED.md](ADVANCED.md)
```

### 使用模板和模式
提供可复用的模板：

```markdown
## 常用模式

### 模式 1：批量处理
```python
def batch_process(items):
    results = []
    for item in items:
        result = process_item(item)
        results.append(result)
    return results
```

### 模式 2：错误重试
```python
def retry_operation(func, max_attempts=3):
    for attempt in range(max_attempts):
        try:
            return func()
        except Exception as e:
            if attempt == max_attempts - 1:
                raise e
            time.sleep(2 ** attempt)
```
```

## 文档质量

### 包含使用场景

```markdown
## 使用场景

### 场景 1：客户支持
自动分类和路由客户邮件

```python
email_category = classify_email(email_content)
if email_category == "billing":
    route_to_billing_team(email)
elif email_category == "technical":
    route_to_tech_support(email)
```

### 场景 2：内容审核
检测和标记不适当内容

```python
moderation_result = moderate_content(user_post)
if moderation_result["flagged"]:
    notify_moderators(user_post, moderation_result["reasons"])
```
```

### 提供完整示例
每个主要功能都应该有完整的、可运行的示例：

```markdown
## 完整示例：PDF 报告生成

```python
from reportlab.lib.pagesizes import letter
from reportlab.pdfgen import canvas

def generate_report(data, output_path):
    """生成 PDF 报告
    
    Args:
        data: 报告数据字典
        output_path: 输出文件路径
    
    Returns:
        str: 生成的文件路径
    """
    c = canvas.Canvas(output_path, pagesize=letter)
    
    # 添加标题
    c.setFont("Helvetica-Bold", 16)
    c.drawString(100, 750, data["title"])
    
    # 添加内容
    c.setFont("Helvetica", 12)
    y_position = 700
    for item in data["items"]:
        c.drawString(100, y_position, item)
        y_position -= 20
    
    c.save()
    return output_path

# 使用示例
report_data = {
    "title": "Monthly Sales Report",
    "items": [
        "Total Sales: $50,000",
        "New Customers: 120",
        "Growth Rate: 15%"
    ]
}

output_file = generate_report(report_data, "report.pdf")
print(f"报告已生成：{output_file}")
```
```

## 测试与验证

### 包含验证步骤

```markdown
## 验证结果

完成操作后，验证结果：

```python
# 1. 检查文件是否创建
assert os.path.exists(output_file), "输出文件未创建"

# 2. 验证文件大小
file_size = os.path.getsize(output_file)
assert file_size > 0, "输出文件为空"

# 3. 验证内容
with open(output_file, 'r') as f:
    content = f.read()
    assert "expected_content" in content, "内容验证失败"

print("✅ 所有验证通过")
```
```

## 性能优化

### 提供优化建议

```markdown
## 性能优化

### 大文件处理
处理大文件时使用流式处理：

```python
# ❌ 不推荐：一次性加载
with open('large_file.txt', 'r') as f:
    content = f.read()  # 可能耗尽内存
    process(content)

# ✅ 推荐：流式处理
with open('large_file.txt', 'r') as f:
    for line in f:  # 逐行处理
        process(line)
```

### 批量操作
批量处理提高效率：

```python
# ❌ 不推荐：逐个处理
for item in items:
    database.insert(item)  # 多次数据库连接

# ✅ 推荐：批量插入
database.bulk_insert(items)  # 一次连接
```

```
## 安全考虑

### 输入验证

````markdown
## 安全性

### 输入验证
始终验证用户输入：

```python
def safe_process(user_input):
    # 验证输入类型
    if not isinstance(user_input, str):
        raise ValueError("输入必须是字符串")
    
    # 验证输入长度
    if len(user_input) > 10000:
        raise ValueError("输入过长")
    
    # 清理输入
    sanitized = user_input.strip()
    
    return process(sanitized)
```

```
### 敏感信息处理

```markdown
## 敏感信息处理

### 避免记录敏感数据
```python
# ❌ 不要记录密码
logger.info(f"User login: {username}, password: {password}")

# ✅ 只记录必要信息
logger.info(f"User login attempt: {username}")
```

### 加密存储
```python
from cryptography.fernet import Fernet

# 加密敏感数据
key = Fernet.generate_key()
cipher = Fernet(key)
encrypted = cipher.encrypt(sensitive_data.encode())
```
```

## 维护与更新

### 版本管理

在 Skill 中记录版本和更新历史：

```markdown
# Skill Name

**版本**: 2.1.0  
**最后更新**: 2025-10-20

## 更新历史

### v2.1.0 (2025-10-20)
- 添加批量处理支持
- 改进错误处理
- 性能优化

### v2.0.0 (2025-09-15)
- 重大更新：API 接口变更
- 添加新功能 XYZ
- 弃用旧方法 ABC
```

## 检查清单

在发布 Skill 前，检查以下项目：

### 必需项 ✅

- [ ] **清晰的 description**：包含功能、触发条件、关键词
- [ ] **YAML Frontmatter**：name 和 description 字段正确
- [ ] **结构化内容**：使用标题、列表、代码块
- [ ] **可运行示例**：至少一个完整的工作示例
- [ ] **限制说明**：明确说明 Skill 的限制

### 推荐项 🌟

- [ ] **分级内容**：简单→复杂的组织方式
- [ ] **外部引用**：复杂内容单独文件
- [ ] **错误处理**：常见问题和解决方案
- [ ] **性能建议**：优化技巧和最佳实践
- [ ] **验证步骤**：如何确认操作成功
- [ ] **版本信息**：记录版本和更新历史

### 优化项 ⭐

- [ ] **使用场景**：真实世界的应用案例
- [ ] **完整示例**：端到端的实现
- [ ] **测试代码**：如何测试功能
- [ ] **安全指南**：输入验证和数据保护
- [ ] **集成指南**：如何与其他工具配合

## 实例：高质量 Skill

以下是一个遵循最佳实践的完整示例：

````markdown
---
name: Email Classifier
description: Classify and route customer support emails by topic, urgency, and sentiment. Use when processing customer emails, support tickets, or when the user mentions email classification, routing, or customer support automation.
---

# Email Classifier

**版本**: 1.0.0  
**最后更新**: 2025-10-20

## 快速开始

分类单个邮件：

```python
classification = classify_email(email_content)
print(f"类别: {classification['category']}")
print(f"紧急度: {classification['urgency']}")
print(f"情感: {classification['sentiment']}")
```

## 功能特性

- **主题分类**：技术、账单、销售、其他
- **紧急度评估**：低、中、高、紧急
- **情感分析**：正面、中性、负面
- **自动路由**：基于分类结果路由

## 使用场景

### 场景 1：客服邮件自动分类

```python
def process_support_email(email):
    result = classify_email(email['content'])
    
    if result['urgency'] == 'urgent':
        notify_on_call_team(email)
    elif result['category'] == 'billing':
        route_to_billing(email)
    elif result['category'] == 'technical':
        create_support_ticket(email, priority=result['urgency'])
```

### 场景 2：批量邮件处理

```python
def process_email_batch(emails):
    results = []
    for email in emails:
        classification = classify_email(email['content'])
        results.append({
            'email_id': email['id'],
            'classification': classification
        })
    return results
```

## 高级功能

### 自定义分类规则
详见 [CUSTOM_RULES.md](CUSTOM_RULES.md)

### 集成 CRM 系统
详见 [CRM_INTEGRATION.md](CRM_INTEGRATION.md)

## 限制

- 邮件长度：最大 50,000 字符
- 批量处理：每次最多 100 封邮件
- 语言支持：英语、中文
- 无网络访问：使用本地模型

## 常见问题

### 问题：分类结果不准确

**症状**：邮件被分类到错误的类别

**解决方案**：
1. 确保邮件内容完整
2. 检查是否包含足够的上下文
3. 考虑提供历史数据作为参考

### 问题：处理速度慢

**症状**：批量处理时间过长

**解决方案**：
```python
# 使用并行处理
from concurrent.futures import ThreadPoolExecutor

def fast_batch_process(emails):
    with ThreadPoolExecutor(max_workers=4) as executor:
        results = list(executor.map(classify_email, emails))
    return results
```

## 测试

验证分类功能：

```python
# 测试用例
test_email = "I need urgent help with my billing issue"
result = classify_email(test_email)

assert result['category'] == 'billing', "类别错误"
assert result['urgency'] == 'urgent', "紧急度错误"
print("✅ 测试通过")
```
```

## 总结
编写高质量 Agent Skills 的关键：

- 清晰的描述 - Claude 知道何时使用
- 结构化内容 - 易于理解和导航
- 具体示例 - 可直接使用的代码
- 渐进式披露 - 从简单到复杂
- 完善的文档 - 限制、错误、最佳实践
遵循这些最佳实践，你的 Skills 将更加：

- ✅ 易于使用
- ✅ 高效可靠
- ✅ 易于维护
- ✅ 适应性强
## 下一步

- 创建第一个 Skill - 实践应用
- Skills 示例集 - 学习官方 Skills 实现
- Skills 示例 - 更多站内整理后的示例
- Skills Cookbook - 更多官方示例
- API 指南 - 深入了解
## 高级参考

### 输出模式（Output Patterns）

参考 `skill-creator` 的 `output-patterns.md`（82 行），学习：

- 结构化输出模式
- 数据格式设计
- 错误响应处理
- 一致性输出规范
### 工作流设计（Workflows）

参考 `skill-creator` 的 `workflows.md`（28 行），学习：

- 多步骤工作流设计
- 状态管理模式
- 异步操作处理
- 工作流编排最佳实践
完整参考文档请访问 [Anthropic Skills 仓库](https://github.com/anthropics/skills/tree/main/skills/skill-creator/references)。
