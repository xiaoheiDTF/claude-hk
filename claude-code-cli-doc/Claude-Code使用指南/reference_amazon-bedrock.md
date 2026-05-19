# Claude-Code / Reference / Amazon-Bedrock

> 来源: claudecn.com

# Amazon Bedrock

介绍如何把 Claude Code 配置到 **Amazon Bedrock** 上运行（包含凭证、IAM、常见排错），以及如何用 **Bedrock Guardrails** 做内容过滤。

## 前提条件

- 你的 AWS 账号已开通 Bedrock 能力，并已获得所需 Claude 模型的访问权限
- 你有可用的 AWS 凭证（Access Key / SSO / Console / Bedrock API key 等）
- 你的 IAM 权限允许调用 Bedrock（见下文）
## 配置步骤

### 1) 首次使用：提交 Use Case（一次性）

Anthropic 模型在 Bedrock 上首次使用需要提交 use case 信息（每个账号通常只需一次）。在 Bedrock 控制台进入 Chat/Text playground，选择任意 Anthropic 模型后按提示填写即可。

### 2) 配置 AWS 凭证

Claude Code 使用 AWS SDK 默认凭证链。你可以选择：

- AWS CLI：aws configure
- 环境变量：AWS_ACCESS_KEY_ID / AWS_SECRET_ACCESS_KEY / AWS_SESSION_TOKEN
- SSO：aws sso login --profile 
，并设置 AWS_PROFILE=
- Bedrock API key：AWS_BEARER_TOKEN_BEDROCK=
#### 高级：自动刷新 SSO/企业凭证
当 Claude Code 检测到凭证过期（本地时间戳或 Bedrock 返回凭证错误）时，可以自动执行你配置的脚本来刷新凭证：

- awsAuthRefresh：适用于“会更新 ~/.aws 目录”的命令（例如 aws sso login ...）
- awsCredentialExport：适用于“无法写入 ~/.aws、只能直接输出临时凭证 JSON”的场景
示例（把刷新脚本写进 `settings.json`）：

```json
{
  "awsAuthRefresh": "aws sso login --profile myprofile",
  "env": {
    "AWS_PROFILE": "myprofile"
  }
}
```

### 3) 启用 Bedrock（环境变量）
至少需要设置：

```bash
export CLAUDE_CODE_USE_BEDROCK=1
export AWS_REGION=us-east-1
```

可选：为“小模型/快模型（Haiku）”单独指定 region：

```bash
export ANTHROPIC_SMALL_FAST_MODEL_AWS_REGION=us-west-2
```

`AWS_REGION` 是必需项；Claude Code 不会从 `~/.aws` 配置文件读取该值。

## Bedrock 场景的推荐 Token 配置
官方推荐：

```bash
export CLAUDE_CODE_MAX_OUTPUT_TOKENS=4096
export MAX_THINKING_TOKENS=1024
```

## IAM 权限（最小可用示例）
给 Claude Code 配一个能调用 Bedrock 的 IAM policy（示例）：

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "AllowModelAndInferenceProfileAccess",
      "Effect": "Allow",
      "Action": [
        "bedrock:InvokeModel",
        "bedrock:InvokeModelWithResponseStream",
        "bedrock:ListInferenceProfiles"
      ],
      "Resource": [
        "arn:aws:bedrock:*:*:inference-profile/*",
        "arn:aws:bedrock:*:*:application-inference-profile/*",
        "arn:aws:bedrock:*:*:foundation-model/*"
      ]
    },
    {
      "Sid": "AllowMarketplaceSubscription",
      "Effect": "Allow",
      "Action": [
        "aws-marketplace:ViewSubscriptions",
        "aws-marketplace:Subscribe"
      ],
      "Resource": "*",
      "Condition": {
        "StringEquals": {
          "aws:CalledViaLast": "bedrock.amazonaws.com"
        }
      }
    }
  ]
}
```

更严谨的做法是把 `Resource` 收敛到你实际使用的 inference profile ARN。

## AWS Guardrails（内容过滤）

Bedrock Guardrails 可用于为 Claude Code 增加内容过滤能力。你在 Bedrock 控制台创建 Guardrail 并发布版本后，把 Guardrail 相关 header 写进 `settings.json` 的 `env`（通过 `ANTHROPIC_CUSTOM_HEADERS` 注入）即可：

```json
{
  "env": {
    "ANTHROPIC_CUSTOM_HEADERS": "X-Amzn-Bedrock-GuardrailIdentifier: your-guardrail-id\nX-Amzn-Bedrock-GuardrailVersion: 1"
  }
}
```

如果你在使用 cross-region inference profiles，Guardrail 也需要启用 Cross-Region inference。

## 常见排错

- 区域问题：先确认目标 region 是否可用 inference profiles；必要时切换 AWS_REGION
- 报错 “on-demand throughput isn’t supported”：需要用 inference profile ID 指定模型
- Claude Code 在 Bedrock 上使用 Invoke API（不使用 Converse API）
## 参考链接

- AWS Bedrock 文档（概览）
- AWS Bedrock Guardrails 文档
- AWS Bedrock Inference Profiles 文档
- AWS Bedrock IAM / 安全文档
